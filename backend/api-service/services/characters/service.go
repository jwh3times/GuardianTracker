package characters

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/membershipstate"
)

const bungieBaseURL = "https://www.bungie.net"

// Service fetches and shapes a user's Destiny 2 characters.
type Service struct {
	bungieClient *bungie.Client
	itemFacts    EquipmentItemReader
	cache        cache.Cache
	cacheTTL     time.Duration

	// refresh fences this owner's per-membership cache against a membership
	// data refresh: a roster loaded before the refresh may answer its own
	// request but may not be left behind afterwards (ADR 0018).
	refresh *membershipstate.Publication
}

// EquipmentItemReader is the entire Items seam Characters consumes. Canonical
// item meaning stays owned by Items; Characters adds only equipped placement
// and the owner-specific instance stat.
type EquipmentItemReader interface {
	Lookup(ctx context.Context, hashes []uint32) (map[uint32]items.AcquisitionFacts, error)
}

func NewService(bungieClient *bungie.Client, itemFacts EquipmentItemReader, c cache.Cache, cacheTTL time.Duration) *Service {
	s := &Service{bungieClient: bungieClient, itemFacts: itemFacts, cache: c, cacheTTL: cacheTTL}
	// The callback runs inside the publication's critical section, so it only
	// deletes the entry and never calls back into the publication.
	s.refresh = membershipstate.New(func(membershipType int, membershipID string) {
		if s.cache == nil {
			return
		}
		s.cache.Delete(charactersCacheKey(membershipType, membershipID))
	})
	return s
}

// ErrCharacterNotFound means the requested character is not in the
// authenticated membership's component-200 roster.
var ErrCharacterNotFound = errors.New("characters: character not found")

// EquipmentState tells the frontend whether Bungie returned component 205.
// A ready result may still contain zero items; unavailable is reserved for an
// absent, null, or server-disabled component.
type EquipmentState string

const (
	EquipmentReady       EquipmentState = "ready"
	EquipmentUnavailable EquipmentState = "unavailable"
)

// EquipmentDetail is the complete owner-only equipment projection for one
// character. It deliberately excludes item-instance IDs: they are needed only
// to join component 300 inside this module and have no product meaning.
type EquipmentDetail struct {
	CharacterID string          `json:"characterId"`
	State       EquipmentState  `json:"state"`
	Items       []EquipmentItem `json:"items"`
}

// EquipmentItem combines canonical manifest facts with character-specific
// equipped placement and optional Power.
type EquipmentItem struct {
	ItemHash string `json:"itemHash"`
	Slot     string `json:"slot"`
	Group    string `json:"group"`
	Name     string `json:"name"`
	ItemType string `json:"itemType"`
	Rarity   string `json:"rarity"`
	Icon     string `json:"icon"`
	Power    *int   `json:"power,omitempty"`

	order int
}

type equipmentSlot struct {
	label string
	group string
	order int
}

// These stable bucket hashes were verified against
// DestinyInventoryBucketDefinition in Manifest
// 244213.26.06.29.2000-1-bnet.65864. Unknown future buckets remain visible in
// the Equipment group instead of being dropped.
var equipmentSlots = map[uint32]equipmentSlot{
	3284755031: {label: "Subclass", group: "Equipment", order: 10},
	1498876634: {label: "Kinetic", group: "Weapons", order: 20},
	2465295065: {label: "Energy", group: "Weapons", order: 30},
	953998645:  {label: "Power", group: "Weapons", order: 40},
	3448274439: {label: "Helmet", group: "Armor", order: 50},
	3551918588: {label: "Gauntlets", group: "Armor", order: 60},
	14239492:   {label: "Chest", group: "Armor", order: 70},
	20886954:   {label: "Legs", group: "Armor", order: 80},
	1585787867: {label: "Class item", group: "Armor", order: 90},
	4023194814: {label: "Ghost", group: "Equipment", order: 100},
	284967655:  {label: "Ship", group: "Equipment", order: 110},
	2025709351: {label: "Vehicle", group: "Equipment", order: 120},
	1506418338: {label: "Artifact", group: "Equipment", order: 130},
	4274335291: {label: "Emblem", group: "Equipment", order: 140},
}

// Character is the frontend-facing representation of a Destiny 2 character.
type Character struct {
	CharacterID          string `json:"characterId"`
	ClassType            int    `json:"classType"`
	ClassName            string `json:"className"`
	RaceName             string `json:"raceName"`
	Light                int    `json:"light"`
	EmblemPath           string `json:"emblemPath"`
	EmblemBackgroundPath string `json:"emblemBackgroundPath"`
	DateLastPlayed       string `json:"dateLastPlayed"`
}

// charactersCacheKey holds a user's shaped character roster. Not scoped by
// manifest version: character components carry no manifest-resolved labels.
func charactersCacheKey(membershipType int, membershipID string) string {
	return fmt.Sprintf("characters:%d:%s", membershipType, membershipID)
}

// GetCharacters returns the user's characters sorted most-recently-played first.
func (s *Service) GetCharacters(ctx context.Context, membershipType int, membershipID, accessToken string) ([]Character, error) {
	return membershipstate.Load(ctx, s.refresh, s.cache, membershipType, membershipID,
		charactersCacheKey(membershipType, membershipID), s.cacheTTL,
		func() ([]Character, error) {
			resp, err := s.bungieClient.GetCharacters(ctx, membershipType, membershipID, accessToken)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch characters: %w", err)
			}

			chars := make([]Character, 0, len(resp.Response.Characters.Data))
			for _, c := range resp.Response.Characters.Data {
				chars = append(chars, Character{
					CharacterID:          c.CharacterID,
					ClassType:            c.ClassType,
					ClassName:            bungie.GetClassName(c.ClassType),
					RaceName:             bungie.GetRaceName(c.RaceType),
					Light:                c.Light,
					EmblemPath:           absoluteURL(c.EmblemPath),
					EmblemBackgroundPath: absoluteURL(c.EmblemBackgroundPath),
					DateLastPlayed:       c.DateLastPlayed,
				})
			}

			// Bungie returns ISO-8601 UTC timestamps; lexicographic sort is chronological.
			sort.Slice(chars, func(i, j int) bool { return chars[i].DateLastPlayed > chars[j].DateLastPlayed })
			return chars, nil
		})
}

// GetEquipment returns one authenticated member's equipped items for one
// character. A single verified GetProfile request keeps components 200, 205,
// and 300 from different response moments from being combined.
func (s *Service) GetEquipment(ctx context.Context, membershipType int, membershipID, characterID, accessToken string) (*EquipmentDetail, error) {
	resp, err := s.bungieClient.GetProfile(ctx, membershipType, membershipID, accessToken, []int{
		bungie.ComponentCharacters,
		bungie.ComponentCharacterEquipment,
		bungie.ComponentItemInstances,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch character equipment: %w", err)
	}

	detail := &EquipmentDetail{
		CharacterID: characterID,
		State:       EquipmentUnavailable,
		Items:       []EquipmentItem{},
	}

	characters := resp.Response.Characters
	if componentAvailable(characters.Data, characters.Disabled) {
		if _, found := (*characters.Data)[characterID]; !found {
			return nil, ErrCharacterNotFound
		}
	}

	equipment := resp.Response.CharacterEquipment
	if !componentAvailable(equipment.Data, equipment.Disabled) {
		return detail, nil
	}
	characterEquipment, found := (*equipment.Data)[characterID]
	if !found {
		return nil, ErrCharacterNotFound
	}

	detail.State = EquipmentReady
	if len(characterEquipment.Items) == 0 {
		return detail, nil
	}

	hashes := make([]uint32, 0, len(characterEquipment.Items))
	for _, equipped := range characterEquipment.Items {
		hashes = append(hashes, equipped.ItemHash)
	}
	if s.itemFacts == nil {
		return nil, errors.New("characters: item facts reader is unavailable")
	}
	facts, err := s.itemFacts.Lookup(ctx, hashes)
	if err != nil {
		return nil, fmt.Errorf("resolve equipped items: %w", err)
	}

	instances := resp.Response.ItemComponents.Instances
	var instanceData map[string]bungie.DestinyItemInstanceComponent
	if componentAvailable(instances.Data, instances.Disabled) {
		instanceData = *instances.Data
	}

	detail.Items = make([]EquipmentItem, 0, len(characterEquipment.Items))
	for _, equipped := range characterEquipment.Items {
		fact, resolved := facts[equipped.ItemHash]
		slot, knownSlot := equipmentSlots[equipped.BucketHash]
		if !knownSlot {
			slot = equipmentSlot{label: fact.ItemType, group: "Equipment", order: 1000}
			if slot.label == "" {
				slot.label = "Equipped item"
			}
		}

		item := EquipmentItem{
			ItemHash: strconv.FormatUint(uint64(equipped.ItemHash), 10),
			Slot:     slot.label,
			Group:    slot.group,
			Name:     fact.Name,
			ItemType: fact.ItemType,
			Rarity:   fact.Rarity,
			Icon:     fact.Icon,
			order:    slot.order,
		}
		if !resolved {
			item.Name = "Unknown item"
			item.ItemType = slot.label
			item.Rarity = "Common"
		}
		if instance, ok := instanceData[equipped.ItemInstanceID]; ok && instance.PrimaryStat != nil {
			power := instance.PrimaryStat.Value
			item.Power = &power
		}
		detail.Items = append(detail.Items, item)
	}

	sort.SliceStable(detail.Items, func(i, j int) bool {
		if detail.Items[i].order != detail.Items[j].order {
			return detail.Items[i].order < detail.Items[j].order
		}
		if detail.Items[i].Name != detail.Items[j].Name {
			return detail.Items[i].Name < detail.Items[j].Name
		}
		return detail.Items[i].ItemHash < detail.Items[j].ItemHash
	})
	return detail, nil
}

func componentAvailable[T any](data *T, disabled *bool) bool {
	return data != nil && (disabled == nil || !*disabled)
}

// InvalidateCache retires this owner's cached roster for one membership as a
// single transition: the generation advances and the entry is deleted together,
// so a load already in flight can still answer its own request but can no
// longer become reusable. Implements the Collections refresh participant
// contract.
func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	s.refresh.Advance(membershipType, membershipID)
}

func absoluteURL(path string) string {
	if path == "" {
		return ""
	}
	return bungieBaseURL + path
}
