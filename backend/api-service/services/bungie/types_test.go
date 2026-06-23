package bungie

import (
	"encoding/json"
	"testing"
)

func TestIsCollected(t *testing.T) {
	// State bit 0 (NotAcquired) set → not collected; clear → collected.
	if (CollectibleComponent{State: 0}).IsCollected() != true {
		t.Error("state 0 should be collected")
	}
	if (CollectibleComponent{State: 1}).IsCollected() != false {
		t.Error("state 1 (NotAcquired) should not be collected")
	}
	// Other state bits set but bit 0 clear → still collected.
	if (CollectibleComponent{State: 4}).IsCollected() != true {
		t.Error("state 4 should be collected (bit 0 clear)")
	}
	if (CollectibleComponent{State: 5}).IsCollected() != false {
		t.Error("state 5 should not be collected (bit 0 set)")
	}
}

func TestItemTypeName(t *testing.T) {
	cases := []struct {
		itemType, subType int
		want              string
	}{
		{ItemTypeWeapon, WeaponSubTypeHandCannon, "Hand Cannon"},
		{ItemTypeArmor, 0, "Armor"},
		{ItemTypeMod, 0, "Mod"},
		{14, 0, "Emblem"},
		{21, 0, "Ship"},
		{22, 0, "Sparrow"},
		{23, 0, "Emote"},
		{24, 0, "Ghost"},
		{999, 0, "Item"},
	}
	for _, tc := range cases {
		if got := ItemTypeName(tc.itemType, tc.subType); got != tc.want {
			t.Errorf("ItemTypeName(%d,%d) = %q, want %q", tc.itemType, tc.subType, got, tc.want)
		}
	}
}

func TestGetTierName(t *testing.T) {
	cases := map[int]string{
		TierTypeExotic:    "Exotic",
		TierTypeLegendary: "Legendary",
		TierTypeRare:      "Rare",
		TierTypeUncommon:  "Uncommon",
		TierTypeCommon:    "Common",
		0:                 "Common", // default
	}
	for in, want := range cases {
		if got := GetTierName(in); got != want {
			t.Errorf("GetTierName(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestGetWeaponTypeName(t *testing.T) {
	cases := map[int]string{
		WeaponSubTypeAutoRifle:       "Auto Rifle",
		WeaponSubTypeShotgun:         "Shotgun",
		WeaponSubTypeMachineGun:      "Machine Gun",
		WeaponSubTypeHandCannon:      "Hand Cannon",
		WeaponSubTypeRocketLauncher:  "Rocket Launcher",
		WeaponSubTypeFusionRifle:     "Fusion Rifle",
		WeaponSubTypeSniperRifle:     "Sniper Rifle",
		WeaponSubTypePulseRifle:      "Pulse Rifle",
		WeaponSubTypeScoutRifle:      "Scout Rifle",
		WeaponSubTypeSidearm:         "Sidearm",
		WeaponSubTypeSword:           "Sword",
		WeaponSubTypeGrenadeLauncher: "Grenade Launcher",
		WeaponSubTypeSubmachineGun:   "Submachine Gun",
		WeaponSubTypeLinearFusion:    "Linear Fusion Rifle",
		WeaponSubTypeTraceRifle:      "Trace Rifle",
		WeaponSubTypeBow:             "Bow",
		WeaponSubTypeGlaive:          "Glaive",
		0:                            "Weapon", // default
	}
	for in, want := range cases {
		if got := GetWeaponTypeName(in); got != want {
			t.Errorf("GetWeaponTypeName(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestGetArmorTypeName(t *testing.T) {
	cases := map[uint32]string{
		SlotHelmet:    "Helmet",
		SlotGauntlets: "Gauntlets",
		SlotChest:     "Chest Armor",
		SlotLegs:      "Leg Armor",
		SlotClassItem: "Class Item",
		0:             "Armor", // default
	}
	for in, want := range cases {
		if got := GetArmorTypeName(in); got != want {
			t.Errorf("GetArmorTypeName(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestGetClassName(t *testing.T) {
	cases := map[int]string{
		ClassTitan:   "Titan",
		ClassHunter:  "Hunter",
		ClassWarlock: "Warlock",
		99:           "Unknown",
	}
	for in, want := range cases {
		if got := GetClassName(in); got != want {
			t.Errorf("GetClassName(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestGetRaceName(t *testing.T) {
	cases := map[int]string{
		RaceHuman:  "Human",
		RaceAwoken: "Awoken",
		RaceExo:    "Exo",
		99:         "Unknown",
	}
	for in, want := range cases {
		if got := GetRaceName(in); got != want {
			t.Errorf("GetRaceName(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestCollectibleDefinitionParsesSourceHash(t *testing.T) {
	blob := `{"hash":123,"itemHash":456,"sourceHash":2065138144,"sourceString":"Source: \"Vault of Glass\" Raid"}`
	var def CollectibleDefinition
	if err := json.Unmarshal([]byte(blob), &def); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if def.SourceHash != 2065138144 {
		t.Errorf("SourceHash = %d, want 2065138144", def.SourceHash)
	}
}
