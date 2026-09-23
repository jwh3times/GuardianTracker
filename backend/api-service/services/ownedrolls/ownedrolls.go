// Package ownedrolls owns one question: which weapon rolls does this membership
// currently hold?
//
// It reads the player's vault, character inventories and equipped items along
// with each instanced item's socket states, and resolves the plug in every perk
// column to the perk's display name. What a perk column is, and what a plug is
// called, are the Manifest's to say — this package asks and does not decide.
//
// The answer is deliberately typed rather than a bare slice. "Your profile did
// not give us your inventory" and "you own no weapons" are different facts, and
// a caller that cannot tell them apart will render the first as the second.
package ownedrolls

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

// Components are the profile components an owned-roll read needs.
//
// 310 (reusablePlugs) is deliberately absent. It answers "what may I swap to
// now", which is a different question from "what did this weapon roll", and the
// accepted owner capture measured it as a large share of a 4.9 MB response.
var Components = []int{
	bungie.ComponentProfileInventory,
	bungie.ComponentCharacterInventories,
	bungie.ComponentCharacterEquipment,
	bungie.ComponentItemSockets,
}

// OwnedRoll is one weapon the membership currently holds, with the perks
// currently sitting in its perk columns.
//
// Perks are display names in the Manifest's spelling, sorted — the same
// normalisation a saved roll target uses, so the two are directly comparable.
type OwnedRoll struct {
	ItemHash   uint32
	InstanceID string
	Perks      []string
}

var (
	// ErrNoCredential means there is no usable Bungie authorization for this
	// membership. Unlike a vendor read, an inventory read cannot fall back to
	// anything public: without the credential there is no answer at all.
	ErrNoCredential = errors.New("ownedrolls: no Bungie authorization for this membership")

	// ErrInventoryUnavailable means the profile read returned no usable
	// inventory or socket component — absent, null, or disabled. It is never
	// conflated with an empty inventory: one means "we cannot tell you", the
	// other means "you hold nothing".
	ErrInventoryUnavailable = errors.New("ownedrolls: profile inventory unavailable")

	// ErrPerksUnavailable means a weapon's perk pool could not be read — the
	// manifest is still downloading or mid-swap, or its query failed. Without
	// the pool no plug can be named, so the read has no answer; it is never
	// reported as a weapon with nothing in its perk columns.
	ErrPerksUnavailable = errors.New("ownedrolls: weapon perk pool unavailable")
)

// CredentialReader resolves a membership's current Bungie access token.
// Satisfied by *auth.TokenStore.
type CredentialReader interface {
	GetValidToken(membershipID string) (string, error)
}

// ProfileReader is the Bungie profile surface this package consumes.
// Satisfied by *bungie.Client.
type ProfileReader interface {
	GetProfile(ctx context.Context, membershipType int, membershipID, accessToken string, components []int) (*bungie.ProfileResponse, error)
}

// PerkPool is the Manifest surface this package consumes: which plugs are real
// perk columns on this weapon, and what are they called?
//
// Satisfied by *items.Service, whose columns already exclude trackers, empty
// sockets and catalysts, and already collapse a perk's base and enhanced
// variants onto one name.
type PerkPool interface {
	GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error)
}

// Service reads owned rolls for a membership.
type Service struct {
	profiles    ProfileReader
	perks       PerkPool
	credentials CredentialReader
}

func NewService(profiles ProfileReader, perks PerkPool, credentials CredentialReader) *Service {
	return &Service{profiles: profiles, perks: perks, credentials: credentials}
}

// Read returns every weapon roll the membership currently holds.
//
// An empty result is an answer: the components were readable and hold no
// weapons. A read that could not be made returns an error instead, so the two
// never render alike.
func (s *Service) Read(ctx context.Context, membershipType int, membershipID string) ([]OwnedRoll, error) {
	token, err := s.credentials.GetValidToken(membershipID)
	if err != nil || token == "" {
		return nil, ErrNoCredential
	}

	resp, err := s.profiles.GetProfile(ctx, membershipType, membershipID, token, Components)
	if err != nil {
		return nil, err
	}

	items, ok := instancedItems(resp)
	if !ok {
		return nil, ErrInventoryUnavailable
	}
	sockets, ok := componentData(resp.Response.ItemComponents.Sockets)
	if !ok {
		return nil, ErrInventoryUnavailable
	}

	rolls := make([]OwnedRoll, 0, len(items))
	pools := map[uint32]map[uint32]string{}
	for _, item := range items {
		state, held := (*sockets)[item.ItemInstanceID]
		if !held {
			continue // not an instanced item, or its sockets were not returned
		}
		pool, err := s.perkNames(pools, item.ItemHash)
		if err != nil {
			return nil, err
		}
		if len(pool) == 0 {
			continue // not a weapon with perk columns
		}
		perks := currentPerks(state.Sockets, pool)
		if len(perks) == 0 {
			continue
		}
		rolls = append(rolls, OwnedRoll{
			ItemHash:   item.ItemHash,
			InstanceID: item.ItemInstanceID,
			Perks:      perks,
		})
	}
	return rolls, nil
}

// perkNames indexes a weapon's real perk plugs by hash, memoised per read.
//
// Both variants of a perk map to the one name they share, so an enhanced plug
// in a socket resolves exactly as its base would.
func (s *Service) perkNames(cache map[uint32]map[uint32]string, itemHash uint32) (map[uint32]string, error) {
	if pool, done := cache[itemHash]; done {
		return pool, nil
	}
	cols, err := s.perks.GetWeaponPerks(itemHash)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPerksUnavailable, err)
	}
	pool := map[uint32]string{}
	for _, c := range cols {
		for _, p := range c.Plugs {
			if p.Base != 0 {
				pool[p.Base] = p.Name
			}
			if p.Enhanced != 0 {
				pool[p.Enhanced] = p.Name
			}
		}
	}
	cache[itemHash] = pool
	return pool, nil
}

// currentPerks resolves the plugs sitting in a weapon's sockets to perk names.
//
// Component 305 is positional over every socket the item has, so a socket whose
// plug is not in the weapon's perk pool is a mod, a cosmetic or the kill
// tracker, and is not part of the roll. A socket holding nothing has a null
// plug hash and is skipped rather than read as plug 0.
func currentPerks(states []bungie.ItemSocketState, pool map[uint32]string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, st := range states {
		if st.PlugHash == nil {
			continue
		}
		name, isPerk := pool[*st.PlugHash]
		if !isPerk {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// instancedItems gathers every item the membership holds, from the vault, each
// character's inventory, and each character's equipped slots.
//
// It reports false only when none of the three components was usable. One
// absent component — a player with no characters, say — still leaves a real
// answer from the others.
func instancedItems(resp *bungie.ProfileResponse) ([]bungie.DestinyItemComponent, bool) {
	var items []bungie.DestinyItemComponent
	any := false

	if vault, ok := componentData(resp.Response.ProfileInventory); ok {
		any = true
		items = append(items, vault.Items...)
	}
	if byCharacter, ok := componentData(resp.Response.CharacterInventories); ok {
		any = true
		for _, inv := range *byCharacter {
			items = append(items, inv.Items...)
		}
	}
	if byCharacter, ok := componentData(resp.Response.CharacterEquipment); ok {
		any = true
		for _, equipped := range *byCharacter {
			items = append(items, equipped.Items...)
		}
	}
	return items, any
}

// componentData reports whether a component carries a usable payload.
//
// Availability is data presence, not the privacy flag. An owner reading their
// own profile gets `privacy = 2` on most components and the data with it — the
// accepted owner capture recorded exactly that on five of six — so gating on
// privacy would refuse a working read. Disabled is a pointer because the live
// API omits it when the component is enabled rather than sending false.
func componentData[T any](env bungie.ComponentEnvelope[T]) (*T, bool) {
	if env.Data == nil {
		return nil, false
	}
	if env.Disabled != nil && *env.Disabled {
		return nil, false
	}
	return env.Data, true
}
