package weekly

import (
	"context"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
)

func weaponDef(name string, tier int) *bungie.InventoryItemDefinition {
	d := &bungie.InventoryItemDefinition{ItemType: bungie.ItemTypeWeapon}
	d.DisplayProperties = bungie.DisplayProperties{Name: name}
	d.Inventory.TierType = tier
	return d
}

func armorDef(name string, tier int) *bungie.InventoryItemDefinition {
	d := &bungie.InventoryItemDefinition{ItemType: bungie.ItemTypeArmor}
	d.DisplayProperties = bungie.DisplayProperties{Name: name}
	d.Inventory.TierType = tier
	return d
}

func modDef(name string) *bungie.InventoryItemDefinition {
	d := &bungie.InventoryItemDefinition{ItemType: bungie.ItemTypeMod}
	d.DisplayProperties = bungie.DisplayProperties{Name: name}
	d.Inventory.TierType = bungie.TierTypeCommon
	return d
}

func TestBuildVendorRotations(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, time.Minute)
	c.Set("live:vendoritems", map[uint32]string{
		100: "Banshee-44", // legendary weapon, missing
		200: "Banshee-44", // legendary weapon, owned
		300: "Banshee-44", // mod -> excluded
		400: "Ada-1",      // exotic armor, missing
	}, time.Minute)
	fm := &fakeManifest{items: map[uint32]*bungie.InventoryItemDefinition{
		100: weaponDef("Fatebringer", bungie.TierTypeLegendary),
		200: weaponDef("Spare Rations", bungie.TierTypeLegendary),
		300: modDef("Backup Mag"),
		400: armorDef("Celestial Nighthawk", bungie.TierTypeExotic),
	}}
	s := &Service{cache: c, manifest: fm}
	missing := map[uint32]struct{}{100: {}, 400: {}}

	got := s.buildVendorRotations(context.Background(), 3, "m", "token", missing, time.Now().UTC())

	if len(got) != 2 {
		t.Fatalf("vendors = %d, want 2 (Banshee, Ada): %+v", len(got), got)
	}
	banshee := got[0]
	if banshee.Name != "Banshee-44" || banshee.Role != "Gunsmith" {
		t.Errorf("vendor[0] = %q/%q, want Banshee-44/Gunsmith", banshee.Name, banshee.Role)
	}
	if len(banshee.Items) != 2 || banshee.Missing != 1 {
		t.Errorf("banshee items=%d missing=%d, want 2/1", len(banshee.Items), banshee.Missing)
	}
	if banshee.Items[0].Hash != "100" {
		t.Errorf("banshee not missing-first: %+v", banshee.Items)
	}
	ada := got[1]
	if ada.Name != "Ada-1" || len(ada.Items) != 1 || ada.Missing != 1 {
		t.Errorf("ada = %+v, want Ada-1 / 1 item / 1 missing", ada)
	}
	if ada.Items[0].Name != "Celestial Nighthawk" || ada.Items[0].Hash != "400" {
		t.Errorf("ada item = %+v", ada.Items[0])
	}
}

func TestBuildVendorRotations_Degraded(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, time.Minute)
	s := &Service{cache: c} // nil bungie client, empty cache, nil manifest

	// Empty token → getLiveVendorItems returns nil → no cards, no manifest call, no panic.
	got := s.buildVendorRotations(context.Background(), 3, "m", "", map[uint32]struct{}{}, time.Now().UTC())
	if len(got) != 0 {
		t.Errorf("degraded = %+v, want empty", got)
	}
}
