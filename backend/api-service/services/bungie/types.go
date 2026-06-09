package bungie

// BungieResponse is the standard wrapper for all Bungie API responses.
type BungieResponse struct {
	Response        interface{} `json:"Response"`
	ErrorCode       int         `json:"ErrorCode"`
	ThrottleSeconds int         `json:"ThrottleSeconds"`
	ErrorStatus     string      `json:"ErrorStatus"`
	Message         string      `json:"Message"`
}

// ManifestResponse contains manifest metadata.
type ManifestResponse struct {
	Response struct {
		Version                 string `json:"version"`
		MobileWorldContentPaths struct {
			En string `json:"en"`
		} `json:"mobileWorldContentPaths"`
	} `json:"Response"`
	ErrorCode   int    `json:"ErrorCode"`
	ErrorStatus string `json:"ErrorStatus"`
	Message     string `json:"Message"`
}

// ProfileResponse contains user profile data with collectibles (component 800).
type ProfileResponse struct {
	Response struct {
		ProfileCollectibles struct {
			Data struct {
				Collectibles map[string]CollectibleComponent `json:"collectibles"`
			} `json:"data"`
			Privacy int `json:"privacy"`
		} `json:"profileCollectibles"`
		CharacterCollectibles struct {
			Data map[string]struct {
				Collectibles map[string]CollectibleComponent `json:"collectibles"`
			} `json:"data"`
		} `json:"characterCollectibles"`
	} `json:"Response"`
	ErrorCode   int    `json:"ErrorCode"`
	ErrorStatus string `json:"ErrorStatus"`
	Message     string `json:"Message"`
}

// CharactersResponse contains a user's characters (component 200).
type CharactersResponse struct {
	Response struct {
		Characters struct {
			Data map[string]CharacterComponent `json:"data"`
		} `json:"characters"`
	} `json:"Response"`
	ErrorCode   int    `json:"ErrorCode"`
	ErrorStatus string `json:"ErrorStatus"`
	Message     string `json:"Message"`
}

// CollectibleComponent is the per-item collectible state from the API.
type CollectibleComponent struct {
	State int `json:"state"`
}

// IsCollected returns true if the item has been collected (state bit 0 is NOT set).
func (c CollectibleComponent) IsCollected() bool {
	return c.State&1 == 0
}

// CharacterComponent holds the character data from component 200.
type CharacterComponent struct {
	CharacterID          string `json:"characterId"`
	ClassType            int    `json:"classType"`
	RaceType             int    `json:"raceType"`
	Light                int    `json:"light"`
	EmblemPath           string `json:"emblemPath"`
	EmblemBackgroundPath string `json:"emblemBackgroundPath"`
	DateLastPlayed       string `json:"dateLastPlayed"`
}

// CollectibleDefinition is a Bungie manifest collectible entry.
type CollectibleDefinition struct {
	Hash              uint32            `json:"hash"`
	DisplayProperties DisplayProperties `json:"displayProperties"`
	SourceString      string            `json:"sourceString"`
	ItemHash          uint32            `json:"itemHash"`
}

// InventoryItemDefinition is a Bungie manifest item entry.
type InventoryItemDefinition struct {
	Hash              uint32            `json:"hash"`
	DisplayProperties DisplayProperties `json:"displayProperties"`
	ItemType          int               `json:"itemType"`
	ItemSubType       int               `json:"itemSubType"`
	Inventory         struct {
		TierType int `json:"tierType"`
	} `json:"inventory"`
	EquippingBlock struct {
		EquipmentSlotTypeHash uint32 `json:"equipmentSlotTypeHash"`
	} `json:"equippingBlock"`
}

// DisplayProperties holds the name, description, and icon for a manifest entity.
type DisplayProperties struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// Item type constants.
const (
	ItemTypeWeapon = 3
	ItemTypeArmor  = 2
)

// Tier type constants.
const (
	TierTypeExotic    = 6
	TierTypeLegendary = 5
	TierTypeRare      = 4
	TierTypeUncommon  = 3
	TierTypeCommon    = 2
)

// GetTierName maps a tier type integer to its rarity name.
func GetTierName(tierType int) string {
	switch tierType {
	case TierTypeExotic:
		return "Exotic"
	case TierTypeLegendary:
		return "Legendary"
	case TierTypeRare:
		return "Rare"
	case TierTypeUncommon:
		return "Uncommon"
	default:
		return "Common"
	}
}

// Weapon sub-type constants.
const (
	WeaponSubTypeAutoRifle        = 6
	WeaponSubTypeShotgun          = 7
	WeaponSubTypeMachineGun       = 8
	WeaponSubTypeHandCannon       = 9
	WeaponSubTypeRocketLauncher   = 10
	WeaponSubTypeFusionRifle      = 11
	WeaponSubTypeSniperRifle      = 12
	WeaponSubTypePulseRifle       = 13
	WeaponSubTypeScoutRifle       = 14
	WeaponSubTypeSidearm          = 17
	WeaponSubTypeSword            = 18
	WeaponSubTypeGrenadeLauncher  = 23
	WeaponSubTypeSubmachineGun    = 22
	WeaponSubTypeLinearFusion     = 24
	WeaponSubTypeTraceRifle       = 25
	WeaponSubTypeBow              = 31
	WeaponSubTypeGlaive           = 33
)

// GetWeaponTypeName returns a human-readable name for a weapon sub-type.
func GetWeaponTypeName(subType int) string {
	switch subType {
	case WeaponSubTypeAutoRifle:
		return "Auto Rifle"
	case WeaponSubTypeShotgun:
		return "Shotgun"
	case WeaponSubTypeMachineGun:
		return "Machine Gun"
	case WeaponSubTypeHandCannon:
		return "Hand Cannon"
	case WeaponSubTypeRocketLauncher:
		return "Rocket Launcher"
	case WeaponSubTypeFusionRifle:
		return "Fusion Rifle"
	case WeaponSubTypeSniperRifle:
		return "Sniper Rifle"
	case WeaponSubTypePulseRifle:
		return "Pulse Rifle"
	case WeaponSubTypeScoutRifle:
		return "Scout Rifle"
	case WeaponSubTypeSidearm:
		return "Sidearm"
	case WeaponSubTypeSword:
		return "Sword"
	case WeaponSubTypeGrenadeLauncher:
		return "Grenade Launcher"
	case WeaponSubTypeSubmachineGun:
		return "Submachine Gun"
	case WeaponSubTypeLinearFusion:
		return "Linear Fusion Rifle"
	case WeaponSubTypeTraceRifle:
		return "Trace Rifle"
	case WeaponSubTypeBow:
		return "Bow"
	case WeaponSubTypeGlaive:
		return "Glaive"
	default:
		return "Weapon"
	}
}

// Armor slot hash constants.
const (
	SlotHelmet   uint32 = 3448274439
	SlotGauntlets uint32 = 3551918588
	SlotChest    uint32 = 14239492
	SlotLegs     uint32 = 20886954
	SlotClassItem uint32 = 1585787867
)

// GetArmorTypeName returns a human-readable name for an armor slot.
func GetArmorTypeName(slotHash uint32) string {
	switch slotHash {
	case SlotHelmet:
		return "Helmet"
	case SlotGauntlets:
		return "Gauntlets"
	case SlotChest:
		return "Chest Armor"
	case SlotLegs:
		return "Leg Armor"
	case SlotClassItem:
		return "Class Item"
	default:
		return "Armor"
	}
}

// Character class constants.
const (
	ClassTitan   = 0
	ClassHunter  = 1
	ClassWarlock = 2
)

// GetClassName maps a class type integer to its name.
func GetClassName(classType int) string {
	switch classType {
	case ClassTitan:
		return "Titan"
	case ClassHunter:
		return "Hunter"
	case ClassWarlock:
		return "Warlock"
	default:
		return "Unknown"
	}
}

// Race type constants.
const (
	RaceHuman   = 0
	RaceAwoken  = 1
	RaceExo     = 2
)

// GetRaceName maps a race type integer to its name.
func GetRaceName(raceType int) string {
	switch raceType {
	case RaceHuman:
		return "Human"
	case RaceAwoken:
		return "Awoken"
	case RaceExo:
		return "Exo"
	default:
		return "Unknown"
	}
}
