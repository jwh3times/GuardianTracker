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
	plugCatTracker = "v400.plugs.weapons.masterworks.trackers"
	plugCatEmpty   = "crafting.recipes.empty_socket"
	plugCatBarrels = "barrels"
	plugCatMags    = "magazines"
	plugCatFrames  = "frames"
	plugCatOrigins = "origins"
)

const weaponItemType = 3

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
	ItemType int `json:"itemType"`
	Sockets  struct {
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
		Name string `json:"name"`
	} `json:"displayProperties"`
	Plug struct {
		PlugCategoryIdentifier string `json:"plugCategoryIdentifier"`
	} `json:"plug"`
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

// classifyColumn assigns a column's role/label from its first non-excluded plug
// item, and reports skip for kill-tracker / empty / unresolvable columns.
func classifyColumn(isIntrinsic bool, hashes []uint32, items map[uint32]*plugItemDef, traitN *int) (role, label string, skip bool) {
	if isIntrinsic {
		return "intrinsic", "Intrinsic", false
	}
	for _, h := range hashes {
		it := items[h]
		if it == nil {
			continue
		}
		switch it.Plug.PlugCategoryIdentifier {
		case plugCatTracker, plugCatEmpty:
			return "", "", true
		case plugCatBarrels:
			return "barrel", "Barrel", false
		case plugCatMags:
			return "magazine", "Magazine", false
		case plugCatFrames:
			*traitN++
			return "trait", fmt.Sprintf("Trait %d", *traitN), false
		case plugCatOrigins:
			return "origin", "Origin", false
		}
	}
	return "", "", true // unknown / unresolved → skip
}

// resolvePerkNames maps plug-item hashes to display names, excluding placeholder/
// tracker plugs, deduping by name and preserving pool order.
func resolvePerkNames(hashes []uint32, items map[uint32]*plugItemDef) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, h := range hashes {
		it := items[h]
		if it == nil {
			continue
		}
		switch it.Plug.PlugCategoryIdentifier {
		case plugCatTracker, plugCatEmpty:
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
		args[i] = int32(h)
	}
	q := "SELECT id, json FROM DestinyPlugSetDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("getPlugSets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var dbID int32
		var blob string
		if err := rows.Scan(&dbID, &blob); err != nil {
			return nil, fmt.Errorf("getPlugSets scan: %w", err)
		}
		var def plugSetDef
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		out[uint32(dbID)] = &def
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
		args[i] = int32(h)
	}
	q := "SELECT id, json FROM DestinyInventoryItemDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("getPlugItems: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var dbID int32
		var blob string
		if err := rows.Scan(&dbID, &blob); err != nil {
			return nil, fmt.Errorf("getPlugItems scan: %w", err)
		}
		var def plugItemDef
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		out[uint32(dbID)] = &def
	}
	return out, rows.Err()
}
