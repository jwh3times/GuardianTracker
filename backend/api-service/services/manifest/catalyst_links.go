package manifest

import (
	"encoding/json"
	"fmt"

	"guardian-tracker/api-service/services/bungie"
)

// CatalystLink pairs an exotic weapon with its resolved catalyst-perk entries
// and the union of objective hashes referenced by EVERY plug in its catalyst
// socket's candidate pool — including the not-yet-unlocked placeholder plug,
// which (e.g. old v300-style catalysts) often carries the real unlock
// objectives even though its own text is excluded from Catalysts. The records
// service uses ObjectiveHashes to link an exotic-catalyst record to its weapon
// via objective-hash overlap.
type CatalystLink struct {
	WeaponName      string
	ObjectiveHashes []uint32
	Catalysts       []WeaponCatalyst
}

// GetCatalystLinks returns one CatalystLink per exotic weapon with a detected
// catalyst socket (Legend of Acrius-shaped exotics without one are omitted). One
// full manifest scan — callers should cache the result per manifest version,
// same pattern as GetWeaponTypesByName / GetExoticWeaponsByName.
func (r *Repository) GetCatalystLinks() ([]CatalystLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.Query(
		"SELECT json FROM DestinyInventoryItemDefinition WHERE json_extract(json, '$.itemType') = ? AND json_extract(json, '$.inventory.tierType') = ?",
		weaponItemType, bungie.TierTypeExotic,
	)
	if err != nil {
		return nil, fmt.Errorf("GetCatalystLinks query: %w", err)
	}
	defer rows.Close()

	var out []CatalystLink
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, fmt.Errorf("GetCatalystLinks scan: %w", err)
		}
		var def weaponDef
		if err := json.Unmarshal([]byte(blob), &def); err != nil {
			continue
		}

		idx, found, err := r.findCatalystSocketLocked(&def)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}

		hashes, err := r.catalystCandidateHashesLocked(def.Sockets.SocketEntries[idx])
		if err != nil {
			return nil, err
		}
		items, err := r.getPlugItemsLocked(hashes)
		if err != nil {
			return nil, err
		}

		catalysts, err := r.resolveCatalystsLocked(hashes, items)
		if err != nil {
			return nil, err
		}

		var objHashes []uint32
		for _, h := range hashes {
			if it := items[h]; it != nil {
				objHashes = append(objHashes, it.Objectives.ObjectiveHashes...)
			}
		}

		out = append(out, CatalystLink{
			WeaponName:      def.DisplayProperties.Name,
			ObjectiveHashes: objHashes,
			Catalysts:       catalysts,
		})
	}
	return out, rows.Err()
}
