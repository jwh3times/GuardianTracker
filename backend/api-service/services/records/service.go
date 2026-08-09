package records

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/observability"
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

// Bungie files both exotic catalysts and weapon crafting patterns under a single
// "Patterns & Catalysts" presentation node (exoticCatalystsRootNodeHash). The
// per-record recordTypeName is the discriminator that separates the two; we walk
// that one node for both endpoints and partition by these values. (The separate
// craftingRootNodeHash points at the in-game "Shape" navigation node, which holds
// no records at all, so it can't drive the crafting list.)
const (
	recordTypeCatalyst = "Exotic Catalysts"
	recordTypePattern  = "Weapon Pattern"
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
	// Effect describes what the catalyst does once unlocked, resolved from the
	// linked weapon's catalyst-perk text (see resolveCatalystEffect); falls back
	// to the record's own description, then "" if nothing resolves.
	Effect string `json:"effect"`
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
	Icon     string           `json:"icon"` // weapon icon path on bungie.net (the pattern record's own icon)
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
	// Cur and Max are legacy/compat fields derived from the record's first
	// RAW objective (record.Objectives[0]), which may be explicitly hidden,
	// whereas Objectives below excludes hidden entries — so Cur/Max must
	// never be correlated with Objectives by index.
	Cur int `json:"cur"`
	Max int `json:"max"`
	// Objectives is the per-objective drill-down for the triumph's progress
	// bar; omitted entirely (not an empty array) when no objective survives
	// visibility filtering, so the response stays byte-identical for
	// triumphs with no objective data.
	Objectives []TriumphObjective `json:"objectives,omitempty"`
}

// TriumphObjective is one objective's progress within a triumph.
type TriumphObjective struct {
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
	GetExoticWeaponsByName() (map[string]manifestrepo.ExoticWeapon, error)
	GetCatalystLinks() ([]manifestrepo.CatalystLink, error)
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

// weaponTypesCacheKey is the cache key for the weapon name→type map. Evicted by
// OnVersionChanged — otherwise stale weapon-type labels would be served for up
// to an hour after a manifest update (B10).
const weaponTypesCacheKey = "manifest:weaponTypesByName"

// exoticWeaponsCacheKey is the cache key for the exotic weapon name→{type,icon}
// map (catalyst weapon-picture resolution). Evicted for the same reason as
// weaponTypesCacheKey.
const exoticWeaponsCacheKey = "manifest:exoticWeaponsByName"

// recordHashKey formats a record hash as the decimal string key Bungie uses to
// index the profile records map.
func recordHashKey(hash uint32) string {
	return strconv.FormatUint(uint64(hash), 10)
}

// weaponTypesByName returns the cached lowercased-weapon-name → weapon-type map
// used to enrich catalyst and crafting entries (B10). Failures return an empty
// map (entries fall back to a generic type) and are not cached.
func (s *Service) weaponTypesByName() map[string]string {
	if cached, ok := s.cache.Get(weaponTypesCacheKey); ok {
		if m, ok := cached.(map[string]string); ok {
			return m
		}
	}
	m, err := s.manifest.GetWeaponTypesByName()
	if err != nil || len(m) == 0 {
		return map[string]string{}
	}
	s.cache.Set(weaponTypesCacheKey, m, time.Hour)
	return m
}

// exoticWeaponsByName returns the cached lowercased exotic-weapon-name →
// {type, icon} map used to resolve a catalyst's weapon picture and type.
// Failures return an empty map (entries fall back to a type glyph) and are not
// cached.
func (s *Service) exoticWeaponsByName() map[string]manifestrepo.ExoticWeapon {
	if cached, ok := s.cache.Get(exoticWeaponsCacheKey); ok {
		if m, ok := cached.(map[string]manifestrepo.ExoticWeapon); ok {
			return m
		}
	}
	m, err := s.manifest.GetExoticWeaponsByName()
	if err != nil || len(m) == 0 {
		return map[string]manifestrepo.ExoticWeapon{}
	}
	s.cache.Set(exoticWeaponsCacheKey, m, time.Hour)
	return m
}

// resolveCatalystWeapon maps an exotic catalyst record to its weapon's type and
// icon. The catalyst record's own icon is a near-transparent generic glyph and
// its name is often an abbreviation of the weapon ("Whisper Catalyst" vs the
// weapon "Whisper of the Worm", or "Immovable Refit" for Vexcalibur), so we
// match against the exotic-weapon map in order of decreasing reliability:
//  1. exact name with the " Catalyst" suffix stripped,
//  2. a unique exotic-weapon name containing the stripped name,
//  3. the longest exotic-weapon name appearing in the catalyst's description
//     ("…while using <Weapon>." / "…to <Weapon> through shaping…").
//
// namesByLen must be the exotic map's keys sorted longest-first (built once by
// the caller). Returns the zero value when nothing matches; the UI then shows a
// type-glyph fallback rather than a wrong weapon.
func resolveCatalystWeapon(catalystName, description string, exotics map[string]manifestrepo.ExoticWeapon, namesByLen []string) manifestrepo.ExoticWeapon {
	base := strings.ToLower(strings.TrimSuffix(catalystName, " Catalyst"))
	if w, ok := exotics[base]; ok {
		return w
	}
	if base != "" {
		match, count := "", 0
		for name := range exotics {
			if strings.Contains(name, base) {
				match, count = name, count+1
			}
		}
		if count == 1 {
			return exotics[match]
		}
	}
	desc := strings.ToLower(description)
	for _, name := range namesByLen {
		if strings.Contains(desc, name) {
			return exotics[name]
		}
	}
	return manifestrepo.ExoticWeapon{}
}

// catalystLinksCacheKey is the cache key for the exotic-weapon catalyst linkage
// data (weapon name, unlock-objective hashes, resolved catalyst-perk text).
// Evicted for the same reason as weaponTypesCacheKey.
const catalystLinksCacheKey = "manifest:catalystLinks"

// catalystLinks returns the cached per-weapon catalyst linkage data. Failures
// return nil (every record then falls back to its own description) and are not
// cached.
func (s *Service) catalystLinks() []manifestrepo.CatalystLink {
	if cached, ok := s.cache.Get(catalystLinksCacheKey); ok {
		if l, ok := cached.([]manifestrepo.CatalystLink); ok {
			return l
		}
	}
	links, err := s.manifest.GetCatalystLinks()
	if err != nil || len(links) == 0 {
		return nil
	}
	s.cache.Set(catalystLinksCacheKey, links, time.Hour)
	return links
}

// catalystObjectiveIndex maps an unlock-objective hash to the single weapon
// link that (unambiguously, on the weapon side) references it. Hashes shared by
// more than one weapon's catalyst-socket pool are omitted — callers must also
// check the record-side count (an objective hash reused by more than one
// catalyst record is equally unusable even when a single weapon claims it).
func catalystObjectiveIndex(links []manifestrepo.CatalystLink) map[uint32]*manifestrepo.CatalystLink {
	byHash := map[uint32][]*manifestrepo.CatalystLink{}
	for i := range links {
		l := &links[i]
		seen := map[uint32]bool{}
		for _, oh := range l.ObjectiveHashes {
			if seen[oh] {
				continue
			}
			seen[oh] = true
			byHash[oh] = append(byHash[oh], l)
		}
	}
	out := map[uint32]*manifestrepo.CatalystLink{}
	for oh, ls := range byHash {
		if len(ls) == 1 {
			out[oh] = ls[0]
		}
	}
	return out
}

// catalystNameIndex maps a lowercased exotic-weapon name to its catalyst link.
func catalystNameIndex(links []manifestrepo.CatalystLink) map[string]*manifestrepo.CatalystLink {
	out := map[string]*manifestrepo.CatalystLink{}
	for i := range links {
		out[strings.ToLower(links[i].WeaponName)] = &links[i]
	}
	return out
}

// catalystPlugNameIndex maps a resolved catalyst-perk name (old-style
// "<Weapon> Catalyst" plugs) to its weapon's link.
func catalystPlugNameIndex(links []manifestrepo.CatalystLink) map[string]*manifestrepo.CatalystLink {
	out := map[string]*manifestrepo.CatalystLink{}
	for i := range links {
		for _, c := range links[i].Catalysts {
			out[c.Name] = &links[i]
		}
	}
	return out
}

// catalystRecordObjectiveCounts counts, among the given catalyst records, how
// many DISTINCT records reference each objective hash — an objective hash
// claimed by more than one record can't safely link either of them, even if
// it's unambiguous on the weapon side (verified real-manifest case: Wavesplitter
// and Cerberus+1's catalyst records happen to share a generic objective hash).
func catalystRecordObjectiveCounts(recordDefs map[uint32]*manifestrepo.RecordDef) map[uint32]int {
	out := map[uint32]int{}
	for _, def := range recordDefs {
		if def.RecordTypeName != recordTypeCatalyst {
			continue
		}
		seen := map[uint32]bool{}
		for _, oh := range def.ObjectiveHashes {
			if seen[oh] {
				continue
			}
			seen[oh] = true
			out[oh]++
		}
	}
	return out
}

// resolveCatalystEffect links a catalyst record to its weapon — via
// objective-hash overlap, then a stripped-name match, then a catalyst-plug-name
// match — and returns the linked weapon's resolved catalyst-perk text (joining
// multiple entries for multi-catalyst exotics). Falls back to the record's own
// description when nothing links or the linked weapon has no displayable text
// (verified real-manifest case: Duality Catalyst), and to "" if that is empty too.
func resolveCatalystEffect(
	def *manifestrepo.RecordDef,
	byObjHash map[uint32]*manifestrepo.CatalystLink,
	recordObjCounts map[uint32]int,
	byName, byPlugName map[string]*manifestrepo.CatalystLink,
) string {
	var link *manifestrepo.CatalystLink
	for _, oh := range def.ObjectiveHashes {
		if recordObjCounts[oh] > 1 {
			continue // claimed by more than one record — ambiguous, unusable
		}
		if l, ok := byObjHash[oh]; ok {
			link = l
			break
		}
	}
	if link == nil {
		stripped := strings.ToLower(strings.TrimSuffix(def.DisplayProperties.Name, " Catalyst"))
		if l, ok := byName[stripped]; ok {
			link = l
		}
	}
	if link == nil {
		if l, ok := byPlugName[def.DisplayProperties.Name]; ok {
			link = l
		}
	}
	if link != nil {
		var parts []string
		for _, c := range link.Catalysts {
			if d := strings.TrimSpace(c.Description); d != "" {
				parts = append(parts, d)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "; ")
		}
	}
	return strings.TrimSpace(def.DisplayProperties.Description)
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

// OnVersionChanged drops the three manifest-derived lookup tables so they
// rebuild from the new manifest. Implements bungie.ManifestObserver.
//
// The per-user `records:*` entries are deliberately left alone — they hold raw
// Bungie profile data, which a manifest swap does not invalidate.
func (s *Service) OnVersionChanged(version string) error {
	s.cache.Delete(weaponTypesCacheKey)
	s.cache.Delete(exoticWeaponsCacheKey)
	s.cache.Delete(catalystLinksCacheKey)
	return nil
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
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "catalyst record definitions lookup failed",
			slog.Int("membership_type", membershipType),
			observability.ID("membership", membershipID),
			observability.Err(err),
		)
		recordDefs = map[uint32]*manifestrepo.RecordDef{}
	}

	profileResp, fetchedAt, err := s.getProfileRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, time.Time{}, err
	}

	// Resolve each catalyst's weapon picture/type from the exotic-weapon map; the
	// catalyst record's own icon is just a generic glyph. namesByLen lets the
	// resolver fall back to scanning the description for a weapon name.
	exotics := s.exoticWeaponsByName()
	namesByLen := make([]string, 0, len(exotics))
	for name := range exotics {
		namesByLen = append(namesByLen, name)
	}
	sort.Slice(namesByLen, func(i, j int) bool { return len(namesByLen[i]) > len(namesByLen[j]) })

	// Hash-first linkage for the effect-text field: an exotic weapon's catalyst
	// text is derived from the manifest's catalyst-socket plugs (see
	// manifest.GetWeaponCatalysts), then linked to this record by objective-hash
	// overlap first (unambiguous on both the weapon and record side), falling
	// back to name-based matching.
	links := s.catalystLinks()
	byObjHash := catalystObjectiveIndex(links)
	byName := catalystNameIndex(links)
	byPlugName := catalystPlugNameIndex(links)
	recordObjCounts := catalystRecordObjectiveCounts(recordDefs)

	catalysts := make([]Catalyst, 0, len(recordHashes))
	for _, hash := range recordHashes {
		// The catalysts root is the combined "Patterns & Catalysts" node, so it
		// also contains weapon-pattern records — keep only true catalysts.
		def, ok := recordDefs[hash]
		if !ok || def.RecordTypeName != recordTypeCatalyst {
			continue
		}
		record, hasRecord := profileResp.Response.ProfileRecords.Data.Records[recordHashKey(hash)]

		name := "Unknown Catalyst"
		if def.DisplayProperties.Name != "" {
			name = def.DisplayProperties.Name
		}

		weapon := resolveCatalystWeapon(name, def.DisplayProperties.Description, exotics, namesByLen)

		cat := Catalyst{
			ID:     fmt.Sprintf("c-%d", hash),
			Name:   name,
			Type:   weapon.Type,
			Icon:   weapon.Icon,
			Effect: resolveCatalystEffect(def, byObjHash, recordObjCounts, byName, byPlugName),
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

		// Source line: while in progress, show what the catalyst needs (the
		// completion requirement); otherwise show where it drops (the obscured
		// description). Both come from the record; fall back to the record's
		// category grouping when the preferred text is missing.
		cat.Source = recordSources[hash]
		if cat.Status == "in-progress" {
			if req := strings.TrimSpace(def.DisplayProperties.Description); req != "" {
				cat.Source = req
			}
		} else if src := strings.TrimSpace(def.StateInfo.ObscuredDescription); src != "" {
			cat.Source = src
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
	// Crafting-pattern records live under the combined "Patterns & Catalysts"
	// node (exoticCatalystsRootNodeHash), not craftingRootNodeHash (a record-less
	// in-game nav node). We walk that node and keep only weapon-pattern records.
	if settings.ExoticCatalystsRootNodeHash == 0 {
		return nil, time.Time{}, manifestrepo.ErrNotReady
	}

	recordHashes, recordSources, err := s.walkNodeForRecords(settings.ExoticCatalystsRootNodeHash)
	if err != nil {
		return nil, time.Time{}, err
	}

	recordDefs, err := s.manifest.GetRecordDefinitions(recordHashes)
	if err != nil {
		observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "crafting record definitions lookup failed",
			slog.Int("membership_type", membershipType),
			observability.ID("membership", membershipID),
			observability.Err(err),
		)
		recordDefs = map[uint32]*manifestrepo.RecordDef{}
	}

	profileResp, fetchedAt, err := s.getProfileRecords(ctx, membershipType, membershipID, bungieToken)
	if err != nil {
		return nil, time.Time{}, err
	}

	weaponTypes := s.weaponTypesByName()

	patterns := make([]CraftPattern, 0, len(recordHashes))
	for _, hash := range recordHashes {
		// The walked root also contains exotic catalyst records — keep only the
		// weapon-pattern records.
		def, ok := recordDefs[hash]
		if !ok || def.RecordTypeName != recordTypePattern {
			continue
		}
		record, hasRecord := profileResp.Response.ProfileRecords.Data.Records[recordHashKey(hash)]

		name := "Unknown Pattern"
		if def.DisplayProperties.Name != "" {
			name = def.DisplayProperties.Name
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
			// The pattern record's own icon is the weapon's picture (unlike the
			// catalyst glyph), so it can be used directly.
			Icon: def.DisplayProperties.Icon,
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
		if err != nil {
			observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "seal root presentation node lookup failed",
				slog.Uint64("root_hash", uint64(rootHash)),
				slog.Int("membership_type", membershipType),
				observability.ID("membership", membershipID),
				observability.Err(err),
			)
			continue
		}
		if rootDefs[rootHash] == nil {
			observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "seal root presentation node missing",
				slog.Uint64("root_hash", uint64(rootHash)),
				slog.Int("membership_type", membershipType),
				observability.ID("membership", membershipID),
			)
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
			observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "seal presentation nodes lookup failed",
				slog.Int("seal_count", len(sealHashes)),
				slog.Int("membership_type", membershipType),
				observability.ID("membership", membershipID),
				observability.Err(err),
			)
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

				// Per-objective drill-down: include every objective that is
				// not explicitly hidden (nil Visible == absent == visible).
				// Redeemed state overrides stale objective payloads; blank
				// labels fall back to "Objective N", numbered over the
				// objectives that survive visibility filtering.
				var triumphObjectives []TriumphObjective
				if hasRecord {
					for _, obj := range record.Objectives {
						if obj.Visible != nil && !*obj.Visible {
							continue
						}
						objMax := obj.CompletionValue
						if objMax <= 0 {
							objMax = 1
						}
						objDone := obj.Complete
						objCur := obj.Progress
						if isComplete {
							objDone = true
							objCur = objMax
						} else if objCur > objMax {
							objCur = objMax
						}
						objLabel := obj.ProgressDescription
						if strings.TrimSpace(objLabel) == "" {
							objLabel = fmt.Sprintf("Objective %d", len(triumphObjectives)+1)
						}
						triumphObjectives = append(triumphObjectives, TriumphObjective{
							Label: objLabel,
							Done:  objDone,
							Cur:   objCur,
							Max:   objMax,
						})
					}
				}

				triumphs = append(triumphs, Triumph{
					Label:      label,
					Done:       isComplete,
					Cur:        cur,
					Max:        max,
					Objectives: triumphObjectives,
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
