package bungie

// BungieResponse is the standard wrapper for all Bungie API responses
type BungieResponse struct {
	Response        interface{} `json:"Response"`
	ErrorCode       int         `json:"ErrorCode"`
	ThrottleSeconds int         `json:"ThrottleSeconds"`
	ErrorStatus     string      `json:"ErrorStatus"`
	Message         string      `json:"Message"`
}

// ManifestResponse contains manifest metadata
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

// ProfileResponse contains user profile data with collectibles (component 800)
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

// CharactersResponse contains a user's characters (component 200)
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

// CharacterComponent represents a single Destiny 2 character (component 200)
type CharacterComponent struct {
	CharacterID          string `json:"characterId"`
	MembershipID         string `json:"membershipId"`
	MembershipType       int    `json:"membershipType"`
	ClassType            int    `json:"classType"`
	RaceType             int    `json:"raceType"`
	GenderType           int    `json:"genderType"`
	Light                int    `json:"light"`
	EmblemPath           string `json:"emblemPath"`
	EmblemBackgroundPath string `json:"emblemBackgroundPath"`
	DateLastPlayed       string `json:"dateLastPlayed"`
	BaseCharacterLevel   int    `json:"baseCharacterLevel"`
	TitleRecordHash      uint32 `json:"titleRecordHash"`
}

// CollectibleComponent represents a single collectible's state
type CollectibleComponent struct {
	State int `json:"state"`
}

// CollectibleState bit flags
const (
	CollectibleStateNone                       = 0
	CollectibleStateNotAcquired                = 1
	CollectibleStateObscured                   = 2
	CollectibleStateInvisible                  = 4
	CollectibleStateCannotAffordMaterialReqs   = 8
	CollectibleStateInventorySpaceUnavailable  = 16
	CollectibleStateUniquenessViolation        = 32
	CollectibleStatePurchaseDisabled           = 64
)

// IsCollected returns true if the collectible has been acquired
func (c CollectibleComponent) IsCollected() bool {
	return c.State&CollectibleStateNotAcquired == 0
}

// InventoryItemDefinition from manifest
type InventoryItemDefinition struct {
	Hash              uint32            `json:"hash"`
	DisplayProperties DisplayProperties `json:"displayProperties"`
	ItemType          int               `json:"itemType"`
	ItemSubType       int               `json:"itemSubType"`
	ClassType         int               `json:"classType"`
	Inventory         InventoryInfo     `json:"inventory"`
	CollectibleHash   uint32            `json:"collectibleHash"`
	EquippingBlock    EquippingBlock    `json:"equippingBlock"`
}

// CollectibleDefinition from manifest
type CollectibleDefinition struct {
	Hash              uint32            `json:"hash"`
	DisplayProperties DisplayProperties `json:"displayProperties"`
	SourceString      string            `json:"sourceString"`
	SourceHash        uint32            `json:"sourceHash"`
	ItemHash          uint32            `json:"itemHash"`
	ParentNodeHashes  []uint32          `json:"parentNodeHashes"`
}

// DisplayProperties contains item visual information
type DisplayProperties struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Icon           string `json:"icon"`
	HasIcon        bool   `json:"hasIcon"`
	HighResIcon    string `json:"highResIcon"`
	IconSequences  []struct {
		Frames []string `json:"frames"`
	} `json:"iconSequences"`
}

// InventoryInfo contains inventory/tier information
type InventoryInfo struct {
	TierType          int    `json:"tierType"`
	TierTypeName      string `json:"tierTypeName"`
	TierTypeHash      uint32 `json:"tierTypeHash"`
	BucketTypeHash    uint32 `json:"bucketTypeHash"`
	RecoveryBucketTypeHash uint32 `json:"recoveryBucketTypeHash"`
}

// EquippingBlock contains equipment slot information
type EquippingBlock struct {
	EquipmentSlotTypeHash uint32 `json:"equipmentSlotTypeHash"`
	UniqueLabel           string `json:"uniqueLabel"`
	AmmoType              int    `json:"ammoType"`
}

// Item type constants from Bungie
const (
	ItemTypeNone              = 0
	ItemTypeCurrency          = 1
	ItemTypeArmor             = 2
	ItemTypeWeapon            = 3
	ItemTypeMessage           = 7
	ItemTypeEngram            = 8
	ItemTypeConsumable        = 9
	ItemTypeMod               = 19
	ItemTypeSubclass          = 16
	ItemTypeShip              = 21
	ItemTypeVehicle           = 22
	ItemTypeEmblem            = 14
	ItemTypeEmote             = 20
	ItemTypeGhost             = 24
	ItemTypeFinisher          = 31
)

// Tier type constants
const (
	TierTypeUnknown   = 0
	TierTypeBasic     = 1
	TierTypeCommon    = 2
	TierTypeUncommon  = 3
	TierTypeRare      = 4
	TierTypeSuperior  = 5  // Legendary
	TierTypeExotic    = 6
)

// Class type constants
const (
	ClassTypeTitan   = 0
	ClassTypeHunter  = 1
	ClassTypeWarlock = 2
	ClassTypeUnknown = 3
)

// Race type constants
const (
	RaceTypeHuman   = 0
	RaceTypeAwoken  = 1
	RaceTypeExo     = 2
	RaceTypeUnknown = 3
)

// GetClassName returns the human-readable class name
func GetClassName(classType int) string {
	switch classType {
	case ClassTypeTitan:
		return "Titan"
	case ClassTypeHunter:
		return "Hunter"
	case ClassTypeWarlock:
		return "Warlock"
	default:
		return "Guardian"
	}
}

// GetRaceName returns the human-readable race name
func GetRaceName(raceType int) string {
	switch raceType {
	case RaceTypeHuman:
		return "Human"
	case RaceTypeAwoken:
		return "Awoken"
	case RaceTypeExo:
		return "Exo"
	default:
		return "Unknown"
	}
}

// Weapon sub-type constants
const (
	WeaponSubTypeAutoRifle       = 6
	WeaponSubTypeShotgun         = 7
	WeaponSubTypeMachineGun      = 8
	WeaponSubTypeHandCannon      = 9
	WeaponSubTypeRocketLauncher  = 10
	WeaponSubTypeFusionRifle     = 11
	WeaponSubTypeSniperRifle     = 12
	WeaponSubTypePulseRifle      = 13
	WeaponSubTypeScoutRifle      = 14
	WeaponSubTypeSidearm         = 17
	WeaponSubTypeSword           = 18
	WeaponSubTypeLinearFusionRifle = 22
	WeaponSubTypeGrenadeLauncher = 23
	WeaponSubTypeSubmachineGun   = 24
	WeaponSubTypeTraceRifle      = 25
	WeaponSubTypeBow             = 31
	WeaponSubTypeGlaive          = 33
)

// Equipment slot hashes for categorization
const (
	EquipmentSlotKinetic  uint32 = 1498876634
	EquipmentSlotEnergy   uint32 = 2465295065
	EquipmentSlotPower    uint32 = 953998645
	EquipmentSlotHelmet   uint32 = 3448274439
	EquipmentSlotGauntlet uint32 = 3551918588
	EquipmentSlotChest    uint32 = 14239492
	EquipmentSlotLeg      uint32 = 20886954
	EquipmentSlotClass    uint32 = 1585787867
)

// GetTierName returns human-readable tier name
func GetTierName(tierType int) string {
	switch tierType {
	case TierTypeExotic:
		return "Exotic"
	case TierTypeSuperior:
		return "Legendary"
	case TierTypeRare:
		return "Rare"
	case TierTypeUncommon:
		return "Uncommon"
	case TierTypeCommon:
		return "Common"
	default:
		return "Unknown"
	}
}

// GetWeaponTypeName returns human-readable weapon type name
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
	case WeaponSubTypeLinearFusionRifle:
		return "Linear Fusion Rifle"
	case WeaponSubTypeGrenadeLauncher:
		return "Grenade Launcher"
	case WeaponSubTypeSubmachineGun:
		return "Submachine Gun"
	case WeaponSubTypeTraceRifle:
		return "Trace Rifle"
	case WeaponSubTypeBow:
		return "Bow"
	case WeaponSubTypeGlaive:
		return "Glaive"
	default:
		return "Unknown Weapon"
	}
}

// GetArmorTypeName returns human-readable armor type based on equipment slot
func GetArmorTypeName(equipSlotHash uint32) string {
	switch equipSlotHash {
	case EquipmentSlotHelmet:
		return "Helmet"
	case EquipmentSlotGauntlet:
		return "Gauntlets"
	case EquipmentSlotChest:
		return "Chest Armor"
	case EquipmentSlotLeg:
		return "Leg Armor"
	case EquipmentSlotClass:
		return "Class Armor"
	default:
		return "Armor"
	}
}
