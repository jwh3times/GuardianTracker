package manifest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Verified weapon socket-category hashes (see the design spec's "Verified
// manifest facts"). Only these two carry displayable perks.
const (
	catIntrinsicTraits uint32 = 3956125808
	catWeaponPerks     uint32 = 4241085061
)

// plugCategoryIdentifier values we special-case.
const (
	plugCatTracker         = "v400.plugs.weapons.masterworks.trackers"
	plugCatEmpty           = "crafting.recipes.empty_socket"
	plugCatFrames          = "frames"
	plugCatCatalysts       = "catalysts"
	plugCatEmptyMasterwork = "v400.empty.exotic.masterwork"
)

const weaponItemType = 3

// isJunkPCI reports whether a plug-category-identifier is never a real weapon
// perk within the weapon-perks socket category and must be skipped — both when
// classifying a column's role/label and when resolving its perk names. This is a
// BLACKLIST (skip only known-junk plugs), not the allowlist the old code used:
// every other pci is a real, displayable perk column and must be kept.
//
//   - masterworks.trackers / crafting.recipes.empty_socket: cosmetic kill
//     trackers and the "no perk rolled yet" placeholder — never real perks.
//   - "catalysts" / v400.empty.exotic.masterwork: the exotic-catalyst socket that
//     16/145 catalyst-bearing exotics carry inside this same socket category.
//     Catalysts are surfaced separately via GetWeaponCatalysts, not perkColumns.
func isJunkPCI(pci string) bool {
	if pci == plugCatCatalysts {
		return true
	}
	return strings.Contains(pci, plugCatTracker) ||
		strings.Contains(pci, plugCatEmpty) ||
		strings.Contains(pci, plugCatEmptyMasterwork)
}

// pciLabels maps a known plug-category-identifier to its column display label.
// classifyPCI also matches versioned suffix variants (e.g. "v950.new.sword0.blades"
// matches "blades") so new weapon-version pcis don't need a table update.
var pciLabels = map[string]string{
	plugCatFrames:  "", // handled specially — numbered "Trait N"
	"barrels":      "Barrel",
	"magazines":    "Magazine",
	"origins":      "Origin",
	"scopes":       "Scope",
	"tubes":        "Launcher Barrel",
	"magazines_gl": "Magazine",
	"batteries":    "Battery",
	"stocks":       "Stock",
	"blades":       "Blade",
	"guards":       "Guard",
	"arrows":       "Arrow",
	"bowstrings":   "Bowstring",
	"hafts":        "Haft",
	"grips":        "Grip",
	"rails":        "Rail",
	"bolts":        "Bolt",
	"launchers":    "Launcher",
}

// pciRoles mirrors pciLabels with the column's stable "role" identifier.
var pciRoles = map[string]string{
	plugCatFrames:  "trait",
	"barrels":      "barrel",
	"magazines":    "magazine",
	"origins":      "origin",
	"scopes":       "scope",
	"tubes":        "barrel",
	"magazines_gl": "magazine",
	"batteries":    "battery",
	"stocks":       "stock",
	"blades":       "blade",
	"guards":       "guard",
	"arrows":       "arrow",
	"bowstrings":   "bowstring",
	"hafts":        "haft",
	"grips":        "grip",
	"rails":        "rail",
	"bolts":        "bolt",
	"launchers":    "barrel",
}

// basePCI resolves a plug-category-identifier to a known base category, matching
// either exactly or by ".<base>" suffix (versioned variants, e.g. sword pcis like
// "v950.new.sword0.blades" or "v950.new.sword0.guards").
func basePCI(pci string) (string, bool) {
	if _, ok := pciRoles[pci]; ok {
		return pci, true
	}
	for base := range pciRoles {
		if strings.HasSuffix(pci, "."+base) {
			return base, true
		}
	}
	return "", false
}

// classifyPCI assigns a column's role/label from a known (non-junk) plug's
// plugCategoryIdentifier, falling back to a generic "Perks" label for any pci we
// don't recognize rather than dropping the column.
func classifyPCI(pci string, traitN *int) (role, label string) {
	base, ok := basePCI(pci)
	if !ok {
		return "perk", "Perks"
	}
	if base == plugCatFrames {
		*traitN++
		return "trait", fmt.Sprintf("Trait %d", *traitN)
	}
	return pciRoles[base], pciLabels[base]
}

// PerkColumn is one socket column of a weapon's possible-perk pool, in display order.
// Perks and Plugs are index-aligned: Plugs[i] carries the hashes behind Perks[i].
type PerkColumn struct {
	// SocketIndex is the column's position in profile component 305. It is
	// internal matching context, not part of the public possible-perks shape.
	SocketIndex int        `json:"-"`
	Role        string     `json:"role"`  // intrinsic | barrel | magazine | trait | origin
	Label       string     `json:"label"` // "Intrinsic", "Barrel", "Trait 1", …
	Perks       []string   `json:"perks"` // possible perk display names, deduped, in pool order
	Plugs       []PerkPlug `json:"plugs"` // the plug hashes behind each name, same order
}

// PerkPlug is one distinct perk of a column, carrying the plug hashes that the
// column's name-level dedupe collapses.
//
// The Manifest publishes a perk's base and enhanced variants as two separate
// item definitions and links them with no field in either direction: they share
// a display name and a plugCategoryHash, and differ only in tier and in
// itemTypeDisplayName ("Barrel" vs "Enhanced Barrel"). The pairing therefore has
// to be derived, and it is only sound inside one socket's own plug pool — the
// display name is not unique manifest-wide, where "Outlaw" resolves to three
// base Trait hashes and "Psychohack" to ten base Origin Trait hashes.
type PerkPlug struct {
	Name string `json:"name"`
	// Base is the unenhanced variant, or 0 when the pool holds only an enhanced one.
	Base uint32 `json:"base"`
	// Enhanced is the enhanced variant, or 0 when the perk has none in this pool.
	Enhanced uint32 `json:"enhanced"`
	// Ambiguous reports that this name did not resolve to a single plug per
	// variant within this pool, so Base/Enhanced are a first-seen sample rather
	// than an answer. Consumers must not treat an ambiguous plug as identifying.
	Ambiguous bool `json:"ambiguous"`
}

// --- parse structs (the shared bungie.InventoryItemDefinition lacks these) ---

type socketEntryDef struct {
	SingleInitialItemHash uint32 `json:"singleInitialItemHash"`
	ReusablePlugSetHash   uint32 `json:"reusablePlugSetHash"`
	RandomizedPlugSetHash uint32 `json:"randomizedPlugSetHash"`
	ReusablePlugItems     []struct {
		PlugItemHash uint32 `json:"plugItemHash"`
	} `json:"reusablePlugItems"`
}

type socketCategoryDef struct {
	SocketCategoryHash uint32 `json:"socketCategoryHash"`
	SocketIndexes      []int  `json:"socketIndexes"`
}

type weaponDef struct {
	Hash              uint32 `json:"hash"`
	DisplayProperties struct {
		Name string `json:"name"`
	} `json:"displayProperties"`
	ItemType  int `json:"itemType"`
	Inventory struct {
		TierType int `json:"tierType"`
	} `json:"inventory"`
	Sockets struct {
		SocketEntries    []socketEntryDef    `json:"socketEntries"`
		SocketCategories []socketCategoryDef `json:"socketCategories"`
	} `json:"sockets"`
}

type plugSetDef struct {
	ReusablePlugItems []struct {
		PlugItemHash     uint32 `json:"plugItemHash"`
		CurrentlyCanRoll *bool  `json:"currentlyCanRoll"`
	} `json:"reusablePlugItems"`
}

type plugItemDef struct {
	DisplayProperties struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"displayProperties"`
	ItemType int `json:"itemType"`
	// ItemTypeDisplayName distinguishes a perk's enhanced variant from its base
	// ("Enhanced Barrel" vs "Barrel"); nothing else in the definition does.
	ItemTypeDisplayName string `json:"itemTypeDisplayName"`
	// Objectives.ObjectiveHashes carries a catalyst plug's unlock-progress
	// objectives (used by the records service to link an exotic-catalyst record
	// to its weapon via objective-hash overlap). Nested under "objectives" here —
	// unlike DestinyRecordDefinition, where the same field sits at the top level.
	Objectives struct {
		ObjectiveHashes []uint32 `json:"objectiveHashes"`
	} `json:"objectives"`
	Plug struct {
		PlugCategoryIdentifier string `json:"plugCategoryIdentifier"`
	} `json:"plug"`
	// Perks feeds catalyst effect-text resolution: the first displayable
	// DestinySandboxPerkDefinition among these wins over the plug's own
	// (often generic) description.
	Perks []struct {
		PerkHash uint32 `json:"perkHash"`
	} `json:"perks"`
}

// GetWeaponHashes returns the hash of every weapon definition (itemType 3), in
// no particular order. It exists so a caller can walk every weapon's perk pool;
// against the manifest verified for #363 that is 2208 weapons, all with perk
// columns.
func (r *Repository) GetWeaponHashes() ([]uint32, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rows, err := r.db.Query(
		"SELECT id FROM DestinyInventoryItemDefinition WHERE json_extract(json, '$.itemType') = ?",
		weaponItemType,
	)
	if err != nil {
		return nil, fmt.Errorf("GetWeaponHashes: %w", err)
	}
	defer rows.Close()

	var out []uint32
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("GetWeaponHashes scan: %w", err)
		}
		// Row ids are the hash reinterpreted as a signed 32-bit integer.
		out = append(out, uint32(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetWeaponHashes: %w", err)
	}
	return out, nil
}

// GetWeaponPerks returns the ordered possible-perk columns for a weapon, or nil
// for non-weapons / unknown hashes. Pure manifest data — no user state.
func (r *Repository) GetWeaponPerks(itemHash uint32) ([]PerkColumn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var blob string
	err := r.db.QueryRow("SELECT json FROM DestinyInventoryItemDefinition WHERE id = ?", hashToDBKey(itemHash)).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetWeaponPerks query: %w", err)
	}
	var def weaponDef
	if err := json.Unmarshal([]byte(blob), &def); err != nil {
		return nil, fmt.Errorf("GetWeaponPerks parse: %w", err)
	}
	if def.ItemType != weaponItemType {
		return nil, nil
	}

	// Selected sockets, in display order: intrinsic category first, then weapon perks.
	type sel struct {
		entry       socketEntryDef
		isIntrinsic bool
		socketIndex int
	}
	var selected []sel
	for _, want := range []uint32{catIntrinsicTraits, catWeaponPerks} {
		for _, cat := range def.Sockets.SocketCategories {
			if cat.SocketCategoryHash != want {
				continue
			}
			for _, idx := range cat.SocketIndexes {
				if idx < 0 || idx >= len(def.Sockets.SocketEntries) {
					continue
				}
				selected = append(selected, sel{
					entry:       def.Sockets.SocketEntries[idx],
					isIntrinsic: want == catIntrinsicTraits,
					socketIndex: idx,
				})
			}
		}
	}
	if len(selected) == 0 {
		return nil, nil // not a weapon with perk sockets
	}

	// Gather every plug-set hash referenced by the selected sockets.
	plugSetHashes := map[uint32]struct{}{}
	for _, s := range selected {
		if s.entry.RandomizedPlugSetHash != 0 {
			plugSetHashes[s.entry.RandomizedPlugSetHash] = struct{}{}
		} else if s.entry.ReusablePlugSetHash != 0 {
			plugSetHashes[s.entry.ReusablePlugSetHash] = struct{}{}
		}
	}
	plugSets, err := r.getPlugSetsLocked(keys(plugSetHashes))
	if err != nil {
		return nil, err
	}

	// Resolve the candidate plug-item hashes for each socket into ordered lists,
	// then collect the union to batch-fetch their names + categories.
	type cand struct {
		hashes []uint32
	}
	cands := make([]cand, len(selected))
	itemHashes := map[uint32]struct{}{}
	for i, s := range selected {
		var hs []uint32
		switch {
		case s.entry.RandomizedPlugSetHash != 0:
			hs = plugSetItemHashes(plugSets[s.entry.RandomizedPlugSetHash])
		case len(s.entry.ReusablePlugItems) > 0:
			for _, p := range s.entry.ReusablePlugItems {
				hs = append(hs, p.PlugItemHash)
			}
		case s.entry.ReusablePlugSetHash != 0:
			hs = plugSetItemHashes(plugSets[s.entry.ReusablePlugSetHash])
		case s.entry.SingleInitialItemHash != 0:
			hs = []uint32{s.entry.SingleInitialItemHash}
		}
		cands[i] = cand{hashes: hs}
		for _, h := range hs {
			itemHashes[h] = struct{}{}
		}
	}
	plugItems, err := r.getPlugItemsLocked(keys(itemHashes))
	if err != nil {
		return nil, err
	}

	// Assemble columns.
	var cols []PerkColumn
	traitN := 0
	for i, s := range selected {
		role, label, skip := classifyColumn(s.isIntrinsic, cands[i].hashes, plugItems, &traitN)
		if skip {
			continue
		}
		perks, plugs := resolvePerks(cands[i].hashes, plugItems)
		if len(perks) == 0 {
			continue
		}
		cols = append(cols, PerkColumn{SocketIndex: s.socketIndex, Role: role, Label: label, Perks: perks, Plugs: plugs})
	}
	return cols, nil
}

// classifyColumn assigns a column's role/label from its first non-junk plug item
// (continuing past junk plugs rather than stopping the whole column at the first
// one — see isJunkPCI), and reports skip only when every plug in the pool is
// junk or unresolvable.
func classifyColumn(isIntrinsic bool, hashes []uint32, items map[uint32]*plugItemDef, traitN *int) (role, label string, skip bool) {
	if isIntrinsic {
		return "intrinsic", "Intrinsic", false
	}
	for _, h := range hashes {
		it := items[h]
		if it == nil || isJunkPCI(it.Plug.PlugCategoryIdentifier) {
			continue
		}
		role, label := classifyPCI(it.Plug.PlugCategoryIdentifier, traitN)
		return role, label, false
	}
	return "", "", true // every plug junk/unresolved → skip
}

// resolvePerks maps a socket pool's plug-item hashes to its distinct perks,
// excluding junk plugs (see isJunkPCI), deduping by display name and preserving
// pool order. The returned slices are index-aligned.
//
// The dedupe is what collapses a perk's base and enhanced variants into one
// entry, since the two share a display name; resolvePerks keeps the hashes that
// collapse rather than discarding them. See PerkPlug for why the pairing is only
// derivable within a single pool.
func resolvePerks(hashes []uint32, items map[uint32]*plugItemDef) ([]string, []PerkPlug) {
	var order []string
	acc := map[string]*perkVariants{}
	for _, h := range hashes {
		it := items[h]
		if it == nil || isJunkPCI(it.Plug.PlugCategoryIdentifier) {
			continue
		}
		name := it.DisplayProperties.Name
		if name == "" {
			continue
		}
		v := acc[name]
		if v == nil {
			v = &perkVariants{}
			acc[name] = v
			order = append(order, name)
		}
		v.add(h, it.ItemTypeDisplayName)
	}
	if len(order) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(order))
	plugs := make([]PerkPlug, 0, len(order))
	for _, name := range order {
		names = append(names, name)
		plugs = append(plugs, acc[name].plug(name))
	}
	return names, plugs
}

// enhancedTDNPrefix is the Manifest's own convention for naming a perk's
// enhanced variant: its itemTypeDisplayName is exactly this prefix plus the
// base's. Verified across every base/enhanced pair that resolves 1:1, with no
// mismatch; it is the only signal that survives, since tier varies
// (Common→Uncommon, Legendary→Legendary) and the two variants routinely share no
// sandbox perk hash at all.
const enhancedTDNPrefix = "Enhanced "

// perkVariants collects one display name's plugs from a single socket pool,
// split by variant. It dedupes repeated hashes because a plug set may list the
// same plug item more than once, which is not the same as two distinct plugs
// sharing a name.
type perkVariants struct {
	base, enhanced       []uint32
	baseTDN, enhancedTDN string
}

func (v *perkVariants) add(hash uint32, tdn string) {
	if strings.HasPrefix(tdn, enhancedTDNPrefix) {
		if !containsHash(v.enhanced, hash) {
			if len(v.enhanced) == 0 {
				v.enhancedTDN = tdn
			}
			v.enhanced = append(v.enhanced, hash)
		}
		return
	}
	if !containsHash(v.base, hash) {
		if len(v.base) == 0 {
			v.baseTDN = tdn
		}
		v.base = append(v.base, hash)
	}
}

// plug reduces the collected variants to one PerkPlug, pairing base to enhanced
// only on the Manifest's naming convention. Anything that does not satisfy it —
// more than one distinct hash in a variant, or a pair whose display types do not
// line up — is reported Ambiguous rather than resolved to whichever plug the
// pool happened to list first.
func (v *perkVariants) plug(name string) PerkPlug {
	p := PerkPlug{Name: name, Ambiguous: len(v.base) > 1 || len(v.enhanced) > 1}
	if len(v.base) > 0 {
		p.Base = v.base[0]
	}
	switch {
	case len(v.enhanced) == 0:
		// No enhanced variant in this pool; Enhanced stays 0.
	case len(v.base) == 0:
		// Enhanced-only pool: nothing to pair against, so the base stays 0
		// rather than being invented.
		p.Enhanced = v.enhanced[0]
	case v.enhancedTDN == enhancedTDNPrefix+v.baseTDN:
		p.Enhanced = v.enhanced[0]
	default:
		// Both variants present but the convention does not hold. Unobserved in
		// the current Manifest; surface it instead of guessing a pairing.
		p.Ambiguous = true
	}
	return p
}

func containsHash(hs []uint32, h uint32) bool {
	for _, x := range hs {
		if x == h {
			return true
		}
	}
	return false
}

// plugSetItemHashes returns the rollable plug-item hashes of a plug set in order
// (currentlyCanRoll == false excluded; absent treated as rollable).
func plugSetItemHashes(ps *plugSetDef) []uint32 {
	if ps == nil {
		return nil
	}
	var out []uint32
	for _, p := range ps.ReusablePlugItems {
		if p.CurrentlyCanRoll != nil && !*p.CurrentlyCanRoll {
			continue
		}
		out = append(out, p.PlugItemHash)
	}
	return out
}

func keys(m map[uint32]struct{}) []uint32 {
	out := make([]uint32, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// getPlugSetsLocked batch-fetches plug sets by hash. Assumes r.mu is held.
func (r *Repository) getPlugSetsLocked(hashes []uint32) (map[uint32]*plugSetDef, error) {
	out := map[uint32]*plugSetDef{}
	err := queryDefsChunked(r.db, hashes, byRowID("DestinyPlugSetDefinition", "getPlugSets"),
		func(id uint32, def *plugSetDef) { out[id] = def })
	if err != nil {
		return nil, err
	}
	return out, nil
}

// getPlugItemsLocked batch-fetches plug-item defs (name + category). Assumes r.mu is held.
func (r *Repository) getPlugItemsLocked(hashes []uint32) (map[uint32]*plugItemDef, error) {
	out := map[uint32]*plugItemDef{}
	err := queryDefsChunked(r.db, hashes, byRowID("DestinyInventoryItemDefinition", "getPlugItems"),
		func(id uint32, def *plugItemDef) { out[id] = def })
	if err != nil {
		return nil, err
	}
	return out, nil
}
