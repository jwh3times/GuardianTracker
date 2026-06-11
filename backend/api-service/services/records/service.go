package records

import (
	"context"
	"fmt"
	"log"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	manifestrepo "guardian-tracker/api-service/services/manifest"
)

// Record state bit flags from the Bungie API.
const (
	RecordStateNone              = 0
	RecordStateRecordRedeemed    = 1  // bit 0 — triumph completed and claimed
	RecordStateRewardUnavailable = 2  // bit 1
	RecordStateObjectiveNotCompleted = 4  // bit 2
	RecordStateObscured          = 8  // bit 3 — not yet unlocked/visible
	RecordStateInvisible         = 16 // bit 4
	RecordStateEntitlementUnowned = 32 // bit 5
	RecordStateCanEquipTitle     = 64 // bit 6
)

// Catalyst represents one exotic weapon catalyst and its completion state.
type Catalyst struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Status string   `json:"status"` // "missing" | "in-progress" | "complete"
	Obj    *CatObj  `json:"obj"`    // non-nil only when status = "in-progress"
	Source string   `json:"source"`
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

// Service handles catalysts, crafting, and seals data.
type Service struct {
	bungie   *bungie.Client
	manifest *manifestrepo.Repository // may be nil if manifest not yet ready
	cache    cache.Cache
	ttl      time.Duration
}

// NewService creates a new records Service.
// manifest may be nil — all methods return empty slices gracefully when it is.
func NewService(b *bungie.Client, m *manifestrepo.Repository, c cache.Cache, ttl time.Duration) *Service {
	return &Service{bungie: b, manifest: m, cache: c, ttl: ttl}
}

// getProfileRecords fetches and caches the profile records component (900) for a user.
func (s *Service) getProfileRecords(ctx context.Context, membershipType int, membershipID, bungieToken string) (*bungie.RecordsProfileResponse, error) {
	cacheKey := fmt.Sprintf("records:%d:%s", membershipType, membershipID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		if r, ok := cached.(*bungie.RecordsProfileResponse); ok {
			return r, nil
		}
	}
	resp, err := s.bungie.GetRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, err
	}
	s.cache.Set(cacheKey, resp, s.ttl)
	return resp, nil
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

// GetCatalysts returns the exotic catalyst completion state for the authenticated user.
func (s *Service) GetCatalysts(ctx context.Context, membershipType int, membershipID, bungieToken string) ([]Catalyst, error) {
	if s.manifest == nil {
		return []Catalyst{}, nil
	}

	settings, err := s.getCoreSettings(ctx)
	if err != nil || settings.ExoticCatalystsRootNodeHash == 0 {
		log.Printf("records: catalyst root node unavailable: %v", err)
		return []Catalyst{}, nil
	}

	recordHashes, recordSources, err := s.walkNodeForRecords(settings.ExoticCatalystsRootNodeHash)
	if err != nil {
		log.Printf("records: walkNodeForRecords (catalysts): %v", err)
		return []Catalyst{}, nil
	}

	recordDefs, err := s.manifest.GetRecordDefinitions(recordHashes)
	if err != nil {
		log.Printf("records: GetRecordDefinitions (catalysts): %v", err)
		recordDefs = map[uint32]*manifestrepo.RecordDef{}
	}

	profileResp, err := s.getProfileRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, err
	}

	catalysts := make([]Catalyst, 0, len(recordHashes))
	for _, hash := range recordHashes {
		hashStr := fmt.Sprintf("%d", hash)
		record, hasRecord := profileResp.Response.ProfileRecords.Data.Records[hashStr]

		name := "Unknown Catalyst"
		if def, ok := recordDefs[hash]; ok && def.DisplayProperties.Name != "" {
			name = def.DisplayProperties.Name
		}
		source := recordSources[hash]

		cat := Catalyst{
			ID:     fmt.Sprintf("c-%d", hash),
			Name:   name,
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

	return catalysts, nil
}

// GetCrafting returns the weapon crafting pattern completion state for the authenticated user.
func (s *Service) GetCrafting(ctx context.Context, membershipType int, membershipID, bungieToken string) ([]CraftPattern, error) {
	if s.manifest == nil {
		return []CraftPattern{}, nil
	}

	settings, err := s.getCoreSettings(ctx)
	if err != nil || settings.CraftingRootNodeHash == 0 {
		log.Printf("records: crafting root node unavailable: %v", err)
		return []CraftPattern{}, nil
	}

	recordHashes, recordSources, err := s.walkNodeForRecords(settings.CraftingRootNodeHash)
	if err != nil {
		log.Printf("records: walkNodeForRecords (crafting): %v", err)
		return []CraftPattern{}, nil
	}

	recordDefs, err := s.manifest.GetRecordDefinitions(recordHashes)
	if err != nil {
		log.Printf("records: GetRecordDefinitions (crafting): %v", err)
		recordDefs = map[uint32]*manifestrepo.RecordDef{}
	}

	profileResp, err := s.getProfileRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, err
	}

	patterns := make([]CraftPattern, 0, len(recordHashes))
	for _, hash := range recordHashes {
		hashStr := fmt.Sprintf("%d", hash)
		record, hasRecord := profileResp.Response.ProfileRecords.Data.Records[hashStr]

		name := "Unknown Pattern"
		weaponType := "Weapon"
		if def, ok := recordDefs[hash]; ok {
			if def.DisplayProperties.Name != "" {
				name = def.DisplayProperties.Name
			}
		}
		source := recordSources[hash]

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

	return patterns, nil
}

// GetSeals returns the title seal completion state for the authenticated user.
func (s *Service) GetSeals(ctx context.Context, membershipType int, membershipID, bungieToken string) ([]Seal, error) {
	if s.manifest == nil {
		return []Seal{}, nil
	}

	settings, err := s.getCoreSettings(ctx)
	if err != nil {
		log.Printf("records: getCoreSettings (seals): %v", err)
		return []Seal{}, nil
	}

	profileResp, err := s.getProfileRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, err
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
				hashStr := fmt.Sprintf("%d", hash)
				record, hasRecord := profileResp.Response.ProfileRecords.Data.Records[hashStr]

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
				hashStr := fmt.Sprintf("%d", sealNode.CompletionRecordHash)
				if r, ok := profileResp.Response.ProfileRecords.Data.Records[hashStr]; ok {
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
	return seals, nil
}
