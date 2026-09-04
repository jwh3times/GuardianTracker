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
type PerkColumn struct {
	Role  string   `json:"role"`  // intrinsic | barrel | magazine | trait | origin
	Label string   `json:"label"` // "Intrinsic", "Barrel", "Trait 1", …
	Perks []string `json:"perks"` // possible perk display names, deduped, in pool order
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
		perks := resolvePerkNames(cands[i].hashes, plugItems)
		if len(perks) == 0 {
			continue
		}
		cols = append(cols, PerkColumn{Role: role, Label: label, Perks: perks})
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

// resolvePerkNames maps plug-item hashes to display names, excluding junk plugs
// (see isJunkPCI), deduping by name and preserving pool order.
func resolvePerkNames(hashes []uint32, items map[uint32]*plugItemDef) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, h := range hashes {
		it := items[h]
		if it == nil || isJunkPCI(it.Plug.PlugCategoryIdentifier) {
			continue
		}
		name := it.DisplayProperties.Name
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
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
	if len(hashes) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(hashes))
	args := make([]any, len(hashes))
	for i, h := range hashes {
		placeholders[i] = "?"
		args[i] = hashToDBKey(h)
	}
	q := "SELECT id, json FROM DestinyPlugSetDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("getPlugSets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var dbID int64
		var blob string
		if err := rows.Scan(&dbID, &blob); err != nil {
			return nil, fmt.Errorf("getPlugSets scan: %w", err)
		}
		var def plugSetDef
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		out[dbKeyToHash(dbID)] = &def
	}
	return out, rows.Err()
}

// getPlugItemsLocked batch-fetches plug-item defs (name + category). Assumes r.mu is held.
func (r *Repository) getPlugItemsLocked(hashes []uint32) (map[uint32]*plugItemDef, error) {
	out := map[uint32]*plugItemDef{}
	if len(hashes) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(hashes))
	args := make([]any, len(hashes))
	for i, h := range hashes {
		placeholders[i] = "?"
		args[i] = hashToDBKey(h)
	}
	q := "SELECT id, json FROM DestinyInventoryItemDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("getPlugItems: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var dbID int64
		var blob string
		if err := rows.Scan(&dbID, &blob); err != nil {
			return nil, fmt.Errorf("getPlugItems scan: %w", err)
		}
		var def plugItemDef
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		out[dbKeyToHash(dbID)] = &def
	}
	return out, rows.Err()
}
