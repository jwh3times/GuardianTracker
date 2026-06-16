package records

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	manifestrepo "guardian-tracker/api-service/services/manifest"
)

// Record state bit flags from the Bungie API.
const (
	RecordStateNone                  = 0
	RecordStateRecordRedeemed        = 1  // bit 0 — triumph completed and claimed
	RecordStateRewardUnavailable     = 2  // bit 1
	RecordStateObjectiveNotCompleted = 4  // bit 2
	RecordStateObscured              = 8  // bit 3 — not yet unlocked/visible
	RecordStateInvisible             = 16 // bit 4
	RecordStateEntitlementUnowned    = 32 // bit 5
	RecordStateCanEquipTitle         = 64 // bit 6
)

// Catalyst represents one exotic weapon catalyst and its completion state.
type Catalyst struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`   // weapon type, e.g. "Hand Cannon"; "" when unresolvable
	Icon   string  `json:"icon"`   // record icon path on bungie.net, may be ""
	Status string  `json:"status"` // "missing" | "in-progress" | "complete"
	Obj    *CatObj `json:"obj"`    // non-nil only when status = "in-progress"
	Source string  `json:"source"`
}

// CatObj holds progress info for an in-progress catalyst.
type CatObj struct {
	Label string `json:"label"`
	Cur   int    `json:"cur"`
	Max   int    `json:"max"`
}

// CraftPattern represents one craftable weapon pattern and its unlock state.
type CraftPattern struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Type     string           `json:"type"`
	Patterns CraftProgressObj `json:"patterns"`
	Note     string           `json:"note"`
	Source   string           `json:"source"`
}

// CraftProgressObj holds the deepsight/pattern completion counts.
type CraftProgressObj struct {
	Cur int `json:"cur"`
	Max int `json:"max"`
}

// Seal represents one title seal and its triumph completion state.
type Seal struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Pct      int       `json:"pct"`
	Gilded   int       `json:"gilded"`
	Left     string    `json:"left"`
	Triumphs []Triumph `json:"triumphs"`
}

// Triumph is one record/triumph within a seal.
type Triumph struct {
	Label string `json:"label"`
	Done  bool   `json:"done"`
	Cur   int    `json:"cur"`
	Max   int    `json:"max"`
}

// ManifestRepo is the subset of the manifest repository the records service uses.
// Satisfied by *manifest.Provider; calls error while the manifest is not ready,
// which degrades each endpoint to an empty result until the download completes.
type ManifestRepo interface {
	GetPresentationNodeDefinitions(hashes []uint32) (map[uint32]*manifestrepo.PresentationNodeDef, error)
	GetRecordDefinitions(hashes []uint32) (map[uint32]*manifestrepo.RecordDef, error)
	GetWeaponTypesByName() (map[string]string, error)
}

// Service handles catalysts, crafting, and seals data.
type Service struct {
	bungie   *bungie.Client
	manifest ManifestRepo // may be nil in degraded mode
	cache    cache.Cache
	ttl      time.Duration
}

// NewService creates a new records Service.
// manifest may be nil — all methods return empty slices gracefully when it is.
func NewService(b *bungie.Client, m ManifestRepo, c cache.Cache, ttl time.Duration) *Service {
	return &Service{bungie: b, manifest: m, cache: c, ttl: ttl}
}

// WeaponTypesCacheKey is the cache key for the weapon name→type map. Exported so
// main.go can evict it from the after-swap hook — otherwise stale weapon-type
// labels would be served for up to an hour after a manifest update (B10).
const WeaponTypesCacheKey = "manifest:weaponTypesByName"

// recordHashKey formats a record hash as the decimal string key Bungie uses to
// index the profile records map.
func recordHashKey(hash uint32) string {
	return strconv.FormatUint(uint64(hash), 10)
}

// weaponTypesByName returns the cached lowercased-weapon-name → weapon-type map
// used to enrich catalyst and crafting entries (B10). Failures return an empty
// map (entries fall back to a generic type) and are not cached.
func (s *Service) weaponTypesByName() map[string]string {
	if cached, ok := s.cache.Get(WeaponTypesCacheKey); ok {
		if m, ok := cached.(map[string]string); ok {
			return m
		}
	}
	m, err := s.manifest.GetWeaponTypesByName()
	if err != nil || len(m) == 0 {
		return map[string]string{}
	}
	s.cache.Set(WeaponTypesCacheKey, m, time.Hour)
	return m
}

// cachedRecords pairs the profile records response with its fetch time (B8).
type cachedRecords struct {
	resp      *bungie.RecordsProfileResponse
	fetchedAt time.Time
}

// recordsCacheKey is the single source of truth for the profile-records cache key.
func recordsCacheKey(membershipType int, membershipID string) string {
	return fmt.Sprintf("records:%d:%s", membershipType, membershipID)
}

// InvalidateCache drops the cached profile records for a user (e.g. on refresh).
func (s *Service) InvalidateCache(membershipType int, membershipID string) {
	s.cache.Delete(recordsCacheKey(membershipType, membershipID))
}

// getProfileRecords fetches and caches the profile records component (900) for a user.
// The returned time is when the data was actually fetched from Bungie.
func (s *Service) getProfileRecords(ctx context.Context, membershipType int, membershipID, bungieToken string) (*bungie.RecordsProfileResponse, time.Time, error) {
	cacheKey := recordsCacheKey(membershipType, membershipID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		if r, ok := cached.(*cachedRecords); ok {
			return r.resp, r.fetchedAt, nil
		}
	}
	resp, err := s.bungie.GetRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, time.Time{}, err
	}
	now := time.Now().UTC()
	s.cache.Set(cacheKey, &cachedRecords{resp: resp, fetchedAt: now}, s.ttl)
	return resp, now, nil
}

// getCoreSettings fetches and caches Bungie's Destiny 2 core settings (root node hashes).
// Cached for 24 hours since these rarely change.
func (s *Service) getCoreSettings(ctx context.Context) (*bungie.CoreSettings, error) {
	const cacheKey = "settings:core"
	if cached, ok := s.cache.Get(cacheKey); ok {
		if cs, ok := cached.(*bungie.CoreSettings); ok {
			return cs, nil
		}
	}
	settings, err := s.bungie.GetCommonSettings(ctx)
	if err != nil {
		return nil, err
	}
	s.cache.Set(cacheKey, settings, 24*time.Hour)
	return settings, nil
}

// walkNodeForRecords does a BFS traversal of presentation nodes starting from rootHash,
// collecting all record hashes found in the tree and the display name of their parent node.
func (s *Service) walkNodeForRecords(rootHash uint32) ([]uint32, map[uint32]string, error) {
	var allHashes []uint32
	sources := map[uint32]string{}

	queue := []uint32{rootHash}
	visited := map[uint32]bool{}

	for len(queue) > 0 {
		chunk := queue
		queue = nil

		nodeDefs, err := s.manifest.GetPresentationNodeDefinitions(chunk)
		if err != nil {
			return nil, nil, fmt.Errorf("walkNodeForRecords: %w", err)
		}

		for _, nodeHash := range chunk {
			if visited[nodeHash] {
				continue
			}
			visited[nodeHash] = true

			node := nodeDefs[nodeHash]
			if node == nil {
				continue
			}

			// Collect record hashes at this node
			for _, r := range node.Children.Records {
				allHashes = append(allHashes, r.RecordHash)
				sources[r.RecordHash] = node.DisplayProperties.Name
			}

			// Queue child presentation nodes
			for _, child := range node.Children.PresentationNodes {
				if !visited[child.PresentationNodeHash] {
					queue = append(queue, child.PresentationNodeHash)
				}
			}
		}
	}

	return allHashes, sources, nil
}

// GetCatalysts returns the exotic catalyst completion state for the authenticated
// user, plus the time the underlying Bungie data was fetched (B8).
func (s *Service) GetCatalysts(ctx context.Context, membershipType int, membershipID, bungieToken string) ([]Catalyst, time.Time, error) {
	if s.manifest == nil {
		return nil, time.Time{}, manifestrepo.ErrNotReady
	}

	settings, err := s.getCoreSettings(ctx)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("records: catalyst settings: %w", err)
	}
	if settings.ExoticCatalystsRootNodeHash == 0 {
		return nil, time.Time{}, manifestrepo.ErrNotReady
	}

	recordHashes, recordSources, err := s.walkNodeForRecords(settings.ExoticCatalystsRootNodeHash)
	if err != nil {
		return nil, time.Time{}, err
	}

	recordDefs, err := s.manifest.GetRecordDefinitions(recordHashes)
	if err != nil {
		log.Printf("records: GetRecordDefinitions (catalysts): %v", err)
		recordDefs = map[uint32]*manifestrepo.RecordDef{}
	}

	profileResp, fetchedAt, err := s.getProfileRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, time.Time{}, err
	}

	weaponTypes := s.weaponTypesByName()

	catalysts := make([]Catalyst, 0, len(recordHashes))
	for _, hash := range recordHashes {
		record, hasRecord := profileResp.Response.ProfileRecords.Data.Records[recordHashKey(hash)]

		name := "Unknown Catalyst"
		icon := ""
		if def, ok := recordDefs[hash]; ok {
			if def.DisplayProperties.Name != "" {
				name = def.DisplayProperties.Name
			}
			icon = def.DisplayProperties.Icon
		}
		source := recordSources[hash]

		// Catalyst records are conventionally named "<Weapon> Catalyst" — resolve
		// the weapon's type from the manifest; unresolvable names stay generic.
		weaponName := strings.ToLower(strings.TrimSuffix(name, " Catalyst"))

		cat := Catalyst{
			ID:     fmt.Sprintf("c-%d", hash),
			Name:   name,
			Type:   weaponTypes[weaponName],
			Icon:   icon,
			Source: source,
		}

		switch {
		case !hasRecord || record.State&RecordStateObscured != 0:
			cat.Status = "missing"
		case record.State&RecordStateRecordRedeemed != 0:
			cat.Status = "complete"
		default:
			cat.Status = "in-progress"
			for _, obj := range record.Objectives {
				if !obj.Complete && obj.CompletionValue > 0 {
					cat.Obj = &CatObj{
						Label: obj.ProgressDescription,
						Cur:   obj.Progress,
						Max:   obj.CompletionValue,
					}
					break
				}
			}
		}

		catalysts = append(catalysts, cat)
	}

	return catalysts, fetchedAt, nil
}

// GetCrafting returns the weapon crafting pattern completion state for the
// authenticated user, plus the time the underlying Bungie data was fetched (B8).
func (s *Service) GetCrafting(ctx context.Context, membershipType int, membershipID, bungieToken string) ([]CraftPattern, time.Time, error) {
	if s.manifest == nil {
		return nil, time.Time{}, manifestrepo.ErrNotReady
	}

	settings, err := s.getCoreSettings(ctx)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("records: crafting settings: %w", err)
	}
	if settings.CraftingRootNodeHash == 0 {
		return nil, time.Time{}, manifestrepo.ErrNotReady
	}

	recordHashes, recordSources, err := s.walkNodeForRecords(settings.CraftingRootNodeHash)
	if err != nil {
		return nil, time.Time{}, err
	}

	recordDefs, err := s.manifest.GetRecordDefinitions(recordHashes)
	if err != nil {
		log.Printf("records: GetRecordDefinitions (crafting): %v", err)
		recordDefs = map[uint32]*manifestrepo.RecordDef{}
	}

	profileResp, fetchedAt, err := s.getProfileRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, time.Time{}, err
	}

	weaponTypes := s.weaponTypesByName()

	patterns := make([]CraftPattern, 0, len(recordHashes))
	for _, hash := range recordHashes {
		record, hasRecord := profileResp.Response.ProfileRecords.Data.Records[recordHashKey(hash)]

		name := "Unknown Pattern"
		if def, ok := recordDefs[hash]; ok {
			if def.DisplayProperties.Name != "" {
				name = def.DisplayProperties.Name
			}
		}
		source := recordSources[hash]

		// Crafting records share the weapon's display name — resolve its type
		// from the manifest; unmatched names fall back to the generic label.
		weaponType := weaponTypes[strings.ToLower(name)]
		if weaponType == "" {
			weaponType = "Weapon"
		}

		cur := 0
		max := 1
		if hasRecord && len(record.Objectives) > 0 {
			// For crafting, objectives represent deepsight resonance completions
			for _, obj := range record.Objectives {
				if obj.CompletionValue > 0 {
					cur = obj.Progress
					max = obj.CompletionValue
					break
				}
			}
		}
		// Clamp progress
		if cur > max {
			cur = max
		}

		note := ""
		if hasRecord && record.State&RecordStateRecordRedeemed != 0 {
			note = "Pattern unlocked"
		}

		patterns = append(patterns, CraftPattern{
			ID:   fmt.Sprintf("p-%d", hash),
			Name: name,
			Type: weaponType,
			Patterns: CraftProgressObj{
				Cur: cur,
				Max: max,
			},
			Note:   note,
			Source: source,
		})
	}

	return patterns, fetchedAt, nil
}

// GetSeals returns the title seal completion state for the authenticated user,
// plus the time the underlying Bungie data was fetched (B8).
func (s *Service) GetSeals(ctx context.Context, membershipType int, membershipID, bungieToken string) ([]Seal, time.Time, error) {
	if s.manifest == nil {
		return nil, time.Time{}, manifestrepo.ErrNotReady
	}

	settings, err := s.getCoreSettings(ctx)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("records: seal settings: %w", err)
	}

	profileResp, fetchedAt, err := s.getProfileRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, time.Time{}, err
	}

	var seals []Seal
	for _, rootHash := range []uint32{settings.ActiveSealsRootNodeHash, settings.LegacySealsRootNodeHash} {
		if rootHash == 0 {
			continue
		}

		rootDefs, err := s.manifest.GetPresentationNodeDefinitions([]uint32{rootHash})
		if err != nil || rootDefs[rootHash] == nil {
			log.Printf("records: GetPresentationNodeDefinitions root %d: %v", rootHash, err)
			continue
		}
		root := rootDefs[rootHash]

		var sealHashes []uint32
		for _, child := range root.Children.PresentationNodes {
			sealHashes = append(sealHashes, child.PresentationNodeHash)
		}
		if len(sealHashes) == 0 {
			continue
		}

		sealDefs, err := s.manifest.GetPresentationNodeDefinitions(sealHashes)
		if err != nil {
			log.Printf("records: GetPresentationNodeDefinitions seals: %v", err)
			continue
		}

		for _, sealHash := range sealHashes {
			sealNode := sealDefs[sealHash]
			if sealNode == nil {
				continue
			}

			// Collect triumphs from direct record children and one level of child nodes
			var triumphHashes []uint32
			for _, r := range sealNode.Children.Records {
				triumphHashes = append(triumphHashes, r.RecordHash)
			}

			var childNodeHashes []uint32
			for _, child := range sealNode.Children.PresentationNodes {
				childNodeHashes = append(childNodeHashes, child.PresentationNodeHash)
			}
			if len(childNodeHashes) > 0 {
				childDefs, childErr := s.manifest.GetPresentationNodeDefinitions(childNodeHashes)
				if childErr == nil {
					for _, childDef := range childDefs {
						if childDef == nil {
							continue
						}
						for _, r := range childDef.Children.Records {
							triumphHashes = append(triumphHashes, r.RecordHash)
						}
					}
				}
			}

			recordDefs, _ := s.manifest.GetRecordDefinitions(triumphHashes)

			total := len(triumphHashes)
			done := 0
			var triumphs []Triumph

			for _, hash := range triumphHashes {
				record, hasRecord := profileResp.Response.ProfileRecords.Data.Records[recordHashKey(hash)]

				label := fmt.Sprintf("Record %d", hash)
				if def, ok := recordDefs[hash]; ok && def.DisplayProperties.Name != "" {
					label = def.DisplayProperties.Name
				}

				isComplete := hasRecord && record.State&RecordStateRecordRedeemed != 0
				cur := 0
				max := 1
				if hasRecord && len(record.Objectives) > 0 {
					cur = record.Objectives[0].Progress
					max = record.Objectives[0].CompletionValue
				}
				if max == 0 {
					max = 1
				}
				if isComplete {
					done++
					cur = max
				}

				triumphs = append(triumphs, Triumph{
					Label: label,
					Done:  isComplete,
					Cur:   cur,
					Max:   max,
				})
			}

			pct := 0
			if total > 0 {
				pct = (done * 100) / total
			}

			left := total - done
			var leftStr string
			switch left {
			case 0:
				leftStr = "Seal complete"
			case 1:
				leftStr = "1 triumph left"
			default:
				leftStr = fmt.Sprintf("%d triumphs left", left)
			}

			// Count gilded completions via the seal's completion record
			gilded := 0
			if sealNode.CompletionRecordHash != 0 {
				if r, ok := profileResp.Response.ProfileRecords.Data.Records[recordHashKey(sealNode.CompletionRecordHash)]; ok {
					for _, io := range r.IntervalObjectives {
						if io.Complete {
							gilded++
						}
					}
				}
			}

			seals = append(seals, Seal{
				ID:       fmt.Sprintf("s-%d", sealHash),
				Name:     sealNode.DisplayProperties.Name,
				Pct:      pct,
				Gilded:   gilded,
				Left:     leftStr,
				Triumphs: triumphs,
			})
		}
	}

	if seals == nil {
		seals = []Seal{}
	}
	return seals, fetchedAt, nil
}
