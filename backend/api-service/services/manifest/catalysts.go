package manifest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"guardian-tracker/api-service/services/bungie"
)

// catWeaponMods is the WEAPON MODS socket category. Most catalyst-bearing
// exotics (129/145, verified) carry their catalyst socket here; the remaining
// 16 (newer exotics) carry it inside catWeaponPerks instead — see
// findCatalystSocketLocked.
const catWeaponMods uint32 = 2685412949

// Catalyst-socket plug-category-identifiers.
const (
	// plugCatEmptyMasterworkExact is the pci of the not-yet-unlocked catalyst
	// placeholder plug across every catalyst-bearing exotic (verified 16/16 of
	// the newer weapon-perks-category exotics; older weapon-mods-category
	// exotics use a per-weapon versioned pci instead — see isQualifyingCatalystPCI).
	catalystEmptyPlugHash uint32 = 1498917124
)

// catalystGenericPrefixes are description prefixes that never carry the real
// catalyst effect — either the generic masterwork-upgrade blurb shown before a
// catalyst is socketed, or the generic "an exotic catalyst can be inserted"
// empty-socket text. When a plug's own description starts with one of these, the
// resolver falls through to the plug's perks[] (already tried first) or the
// same-named-row fallback (old-style v300 catalysts).
var catalystGenericPrefixes = []string{
	"Upgrades this weapon to a Masterwork",
	"An Exotic catalyst can be inserted",
	"When upgraded to a Masterwork",
}

// WeaponCatalyst is one exotic weapon's catalyst-perk entry. Most exotics have
// exactly one; a handful of newer exotics (Wish-Keeper, Revision Zero,
// Vexcalibur, Choir of One, …) legitimately have 3-4.
type WeaponCatalyst struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// sandboxPerkDef is a minimal DestinySandboxPerkDefinition.
type sandboxPerkDef struct {
	DisplayProperties struct {
		Description string `json:"description"`
	} `json:"displayProperties"`
	IsDisplayable bool `json:"isDisplayable"`
}

// isTrackerPCI reports whether a pci is a cosmetic kill/damage tracker — never a
// real perk or catalyst plug.
func isTrackerPCI(pci string) bool {
	return strings.Contains(pci, plugCatTracker)
}

// isQualifyingCatalystPCI reports whether a plug-category-identifier marks the
// socket it belongs to as the exotic-catalyst socket: the real catalyst-content
// pci ("catalysts", used by every newer-style exotic), the shared empty-catalyst
// pci (v400.empty.exotic.masterwork), or any per-weapon versioned masterwork pci
// (older v300-style exotics, e.g. "v300_new_hand_cannon1_masterwork") — but never
// a kill tracker, which also happens to contain "masterworks" as a substring.
func isQualifyingCatalystPCI(pci string) bool {
	if isTrackerPCI(pci) {
		return false
	}
	return pci == plugCatCatalysts || pci == plugCatEmptyMasterwork || strings.Contains(pci, "masterwork")
}

// isEmptyCatalystPlug reports whether a plug is the not-yet-unlocked catalyst
// placeholder rather than real catalyst content — by hash, by pci, or (the
// v300-style "Upgrade Masterwork" plug, which reuses the real content plug's own
// pci) by name.
func isEmptyCatalystPlug(hash uint32, it *plugItemDef) bool {
	if hash == catalystEmptyPlugHash || it == nil {
		return true
	}
	if it.Plug.PlugCategoryIdentifier == plugCatEmptyMasterwork {
		return true
	}
	switch it.DisplayProperties.Name {
	case "Empty Catalyst Socket", "Upgrade Masterwork":
		return true
	}
	return false
}

// isGenericCatalystText reports whether a description is one of the generic
// masterwork/empty-socket blurbs rather than a real per-perk effect description.
func isGenericCatalystText(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	for _, p := range catalystGenericPrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// GetWeaponCatalysts returns the catalyst-perk pool for an exotic weapon, or nil
// for non-weapons, non-exotics, or exotics without a catalyst socket (e.g. Legend
// of Acrius). Pure manifest data — no user state.
func (r *Repository) GetWeaponCatalysts(itemHash uint32) ([]WeaponCatalyst, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var blob string
	err := r.db.QueryRow("SELECT json FROM DestinyInventoryItemDefinition WHERE id = ?", hashToDBKey(itemHash)).Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetWeaponCatalysts query: %w", err)
	}
	var def weaponDef
	if err := json.Unmarshal([]byte(blob), &def); err != nil {
		return nil, fmt.Errorf("GetWeaponCatalysts parse: %w", err)
	}
	if def.ItemType != weaponItemType || def.Inventory.TierType != bungie.TierTypeExotic {
		return nil, nil
	}

	idx, found, err := r.findCatalystSocketLocked(&def)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	hashes, err := r.catalystCandidateHashesLocked(def.Sockets.SocketEntries[idx])
	if err != nil {
		return nil, err
	}
	items, err := r.getPlugItemsLocked(hashes)
	if err != nil {
		return nil, err
	}

	return r.resolveCatalystsLocked(hashes, items)
}

// resolveCatalystsLocked resolves a catalyst socket's candidate plug hashes into
// the ordered, filtered WeaponCatalyst list. Assumes r.mu is already held.
func (r *Repository) resolveCatalystsLocked(hashes []uint32, items map[uint32]*plugItemDef) ([]WeaponCatalyst, error) {
	var out []WeaponCatalyst
	for _, h := range hashes {
		it := items[h]
		if it == nil || isEmptyCatalystPlug(h, it) || isTrackerPCI(it.Plug.PlugCategoryIdentifier) {
			continue
		}
		name := it.DisplayProperties.Name
		if name == "" {
			continue
		}
		desc, err := r.resolveCatalystEffectLocked(it)
		if err != nil {
			return nil, err
		}
		out = append(out, WeaponCatalyst{Name: name, Description: desc})
	}
	return out, nil
}

// findCatalystSocketLocked returns the socket-entry index of a weapon's catalyst
// socket, searching the WEAPON MODS category first (129/145 exotics, verified)
// then WEAPON PERKS (the remaining 16 newer exotics). Assumes r.mu is held.
func (r *Repository) findCatalystSocketLocked(def *weaponDef) (int, bool, error) {
	for _, wantCat := range []uint32{catWeaponMods, catWeaponPerks} {
		for _, sc := range def.Sockets.SocketCategories {
			if sc.SocketCategoryHash != wantCat {
				continue
			}
			for _, idx := range sc.SocketIndexes {
				if idx < 0 || idx >= len(def.Sockets.SocketEntries) {
					continue
				}
				hashes, err := r.catalystCandidateHashesLocked(def.Sockets.SocketEntries[idx])
				if err != nil {
					return 0, false, err
				}
				if len(hashes) == 0 {
					continue
				}
				items, err := r.getPlugItemsLocked(hashes)
				if err != nil {
					return 0, false, err
				}
				for _, h := range hashes {
					if it := items[h]; it != nil && isQualifyingCatalystPCI(it.Plug.PlugCategoryIdentifier) {
						return idx, true, nil
					}
				}
			}
		}
	}
	return 0, false, nil
}

// catalystCandidateHashesLocked resolves one socket entry's candidate plug-item
// hashes, in the same priority order GetWeaponPerks uses (randomized plug set >
// explicit reusable plug items > reusable plug set > single initial item).
// Assumes r.mu is held.
func (r *Repository) catalystCandidateHashesLocked(e socketEntryDef) ([]uint32, error) {
	var plugSetHash uint32
	switch {
	case e.RandomizedPlugSetHash != 0:
		plugSetHash = e.RandomizedPlugSetHash
	case len(e.ReusablePlugItems) > 0:
		hs := make([]uint32, 0, len(e.ReusablePlugItems))
		for _, p := range e.ReusablePlugItems {
			hs = append(hs, p.PlugItemHash)
		}
		return hs, nil
	case e.ReusablePlugSetHash != 0:
		plugSetHash = e.ReusablePlugSetHash
	case e.SingleInitialItemHash != 0:
		return []uint32{e.SingleInitialItemHash}, nil
	default:
		return nil, nil
	}
	plugSets, err := r.getPlugSetsLocked([]uint32{plugSetHash})
	if err != nil {
		return nil, err
	}
	return plugSetItemHashes(plugSets[plugSetHash]), nil
}

// resolveCatalystEffectLocked derives a catalyst plug's displayed effect text:
//  1. the first displayable DestinySandboxPerkDefinition among the plug's perks[];
//  2. else the plug's own description, if not a generic masterwork/empty blurb;
//  3. else (old v300-style catalysts, whose socket plug carries only generic
//     text) the same-named row elsewhere in the table — preferring an itemType19
//     (Mod) row with a displayable perk, as real catalyst payloads live there.
//
// Returns "" if none of the three yield text (verified: only Duality Catalyst).
// Assumes r.mu is held.
func (r *Repository) resolveCatalystEffectLocked(it *plugItemDef) (string, error) {
	txt, err := r.firstDisplayablePerkTextLocked(it)
	if err != nil {
		return "", err
	}
	if txt != "" {
		return txt, nil
	}
	if !isGenericCatalystText(it.DisplayProperties.Description) {
		return it.DisplayProperties.Description, nil
	}
	return r.sameNamedCatalystFallbackLocked(it.DisplayProperties.Name)
}

// firstDisplayablePerkTextLocked returns the first displayable sandbox-perk
// description among a plug's perks[], or "" if it has none. Assumes r.mu is held.
func (r *Repository) firstDisplayablePerkTextLocked(it *plugItemDef) (string, error) {
	if it == nil || len(it.Perks) == 0 {
		return "", nil
	}
	hashes := make([]uint32, 0, len(it.Perks))
	for _, p := range it.Perks {
		hashes = append(hashes, p.PerkHash)
	}
	perks, err := r.getSandboxPerksLocked(hashes)
	if err != nil {
		return "", err
	}
	for _, p := range it.Perks {
		sp := perks[p.PerkHash]
		if sp == nil || !sp.IsDisplayable {
			continue
		}
		if desc := strings.TrimSpace(sp.DisplayProperties.Description); desc != "" {
			return desc, nil
		}
	}
	return "", nil
}

// sameNamedCatalystFallbackLocked looks up every DestinyInventoryItemDefinition
// row sharing the given display name and returns the first displayable
// sandbox-perk text found, preferring an itemType 19 (Mod) row — old-style v300
// catalysts store their real effect on such a row, separate from the socket's
// own (generic-text) content plug. Returns "" if nothing matches. Assumes r.mu
// is held.
func (r *Repository) sameNamedCatalystFallbackLocked(name string) (string, error) {
	if name == "" {
		return "", nil
	}
	rows, err := r.db.Query(
		"SELECT json FROM DestinyInventoryItemDefinition WHERE json_extract(json, '$.displayProperties.name') = ?",
		name,
	)
	if err != nil {
		return "", fmt.Errorf("sameNamedCatalystFallback: %w", err)
	}
	defer rows.Close()

	var preferred, any string
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return "", fmt.Errorf("sameNamedCatalystFallback scan: %w", err)
		}
		var cand plugItemDef
		if err := json.Unmarshal([]byte(blob), &cand); err != nil {
			continue
		}
		txt, err := r.firstDisplayablePerkTextLocked(&cand)
		if err != nil {
			return "", err
		}
		if txt == "" {
			continue
		}
		if cand.ItemType == bungie.ItemTypeMod && preferred == "" {
			preferred = txt
		}
		if any == "" {
			any = txt
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("sameNamedCatalystFallback rows: %w", err)
	}
	if preferred != "" {
		return preferred, nil
	}
	return any, nil
}

// getSandboxPerksLocked batch-fetches sandbox perk defs (name/description +
// isDisplayable) by hash. Assumes r.mu is held.
func (r *Repository) getSandboxPerksLocked(hashes []uint32) (map[uint32]*sandboxPerkDef, error) {
	out := map[uint32]*sandboxPerkDef{}
	if len(hashes) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(hashes))
	args := make([]any, len(hashes))
	for i, h := range hashes {
		placeholders[i] = "?"
		args[i] = int32(h)
	}
	q := "SELECT id, json FROM DestinySandboxPerkDefinition WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("getSandboxPerks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var dbID int32
		var blob string
		if err := rows.Scan(&dbID, &blob); err != nil {
			return nil, fmt.Errorf("getSandboxPerks scan: %w", err)
		}
		var def sandboxPerkDef
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}
		out[uint32(dbID)] = &def
	}
	return out, rows.Err()
}
