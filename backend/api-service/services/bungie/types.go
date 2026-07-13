package bungie

// BungieResponse is the standard wrapper for all Bungie API responses.
type BungieResponse struct {
	Response        any    `json:"Response"`
	ErrorCode       int    `json:"ErrorCode"`
	ThrottleSeconds int    `json:"ThrottleSeconds"`
	ErrorStatus     string `json:"ErrorStatus"`
	Message         string `json:"Message"`
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
	SourceHash        uint32            `json:"sourceHash"`
	ItemHash          uint32            `json:"itemHash"`
}

// InventoryItemDefinition is a Bungie manifest item entry.
type InventoryItemDefinition struct {
	Hash              uint32            `json:"hash"`
	DisplayProperties DisplayProperties `json:"displayProperties"`
	ItemType          int               `json:"itemType"`
	ItemSubType       int               `json:"itemSubType"`
	// ClassType is absent on class-agnostic items. Keep it as a pointer so an
	// omitted value cannot be mistaken for Titan (class type 0).
	ClassType *int `json:"classType,omitempty"`
	Inventory struct {
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
	ItemTypeWeapon   = 3
	ItemTypeArmor    = 2
	ItemTypeMod      = 19
	ItemTypeFinisher = 29
)

// Item sub-type constants used for classification.
const (
	ItemSubTypeShader   = 20
	ItemSubTypeOrnament = 21
)

// ItemTypeName returns a human-readable item type string, using weapon sub-type for specificity.
func ItemTypeName(itemType, subType int) string {
	switch itemType {
	case ItemTypeWeapon:
		return GetWeaponTypeName(subType)
	case ItemTypeArmor:
		return "Armor"
	case ItemTypeMod:
		switch subType {
		case ItemSubTypeShader:
			return "Shader"
		case ItemSubTypeOrnament:
			return "Ornament"
		}
		return "Mod"
	case 14:
		return "Emblem"
	case 21:
		return "Ship"
	case 22:
		return "Sparrow"
	case 23:
		return "Emote"
	case 24:
		return "Ghost"
	case ItemTypeFinisher:
		return "Finisher"
	default:
		return "Item"
	}
}

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
	WeaponSubTypeGrenadeLauncher = 23
	WeaponSubTypeSubmachineGun   = 22
	WeaponSubTypeLinearFusion    = 24
	WeaponSubTypeTraceRifle      = 25
	WeaponSubTypeBow             = 31
	WeaponSubTypeGlaive          = 33
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
	SlotHelmet    uint32 = 3448274439
	SlotGauntlets uint32 = 3551918588
	SlotChest     uint32 = 14239492
	SlotLegs      uint32 = 20886954
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
	RaceHuman  = 0
	RaceAwoken = 1
	RaceExo    = 2
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

// --- Weekly / Milestones / Vendors ---

// Vendor hash constants for well-known Bungie vendors.
const (
	XurVendorHash       uint32 = 2190858386
	Ada1VendorHash      uint32 = 350061650
	Banshee44VendorHash uint32 = 672118013
	// Ritual vendors. Verified against DestinyVendorDefinition in the real
	// manifest: 3603221665=Lord Shaxx, 69482069=Commander Zavala, 248695599=The Drifter.
	ShaxxVendorHash   uint32 = 3603221665
	ZavalaVendorHash  uint32 = 69482069
	DrifterVendorHash uint32 = 248695599
)

// Milestone type constants matching DestinyMilestoneType enum.
const (
	MilestoneTypeUnknown  = 0
	MilestoneTypeOneTime  = 1
	MilestoneTypeTutorial = 2
	MilestoneTypeWeekly   = 3
	MilestoneTypeDaily    = 4
	MilestoneTypeSpecial  = 5
)

// PublicMilestonesResponse wraps the Bungie milestone map.
type PublicMilestonesResponse struct {
	Response    map[string]PublicMilestone `json:"Response"`
	ErrorCode   int                        `json:"ErrorCode"`
	ErrorStatus string                     `json:"ErrorStatus"`
	Message     string                     `json:"Message"`
}

// PublicMilestone is one entry from the Bungie Milestones API.
type PublicMilestone struct {
	MilestoneHash   uint32              `json:"milestoneHash"`
	AvailableQuests []MilestoneQuest    `json:"availableQuests"`
	Activities      []MilestoneActivity `json:"activities"`
}

// MilestoneQuest is one quest associated with a milestone.
type MilestoneQuest struct {
	QuestItemHash uint32 `json:"questItemHash"`
}

// MilestoneActivity is one activity associated with a milestone.
type MilestoneActivity struct {
	ActivityHash   uint32   `json:"activityHash"`
	ModifierHashes []uint32 `json:"modifierHashes"`
}

// ActivityModifierDefinition is a DestinyActivityModifierDefinition entry from the manifest.
type ActivityModifierDefinition struct {
	Hash              uint32            `json:"hash"`
	DisplayProperties DisplayProperties `json:"displayProperties"`
}

// ActivityDefinition is a minimal DestinyActivityDefinition entry from the manifest.
type ActivityDefinition struct {
	Hash              uint32            `json:"hash"`
	DisplayProperties DisplayProperties `json:"displayProperties"`
	ActivityTypeHash  uint32            `json:"activityTypeHash"`
}

// CharacterVendorsResponse wraps the per-character vendor API response.
type CharacterVendorsResponse struct {
	Response struct {
		Vendors struct {
			Data map[string]VendorComponent `json:"data"`
		} `json:"vendors"`
		Sales struct {
			Data map[string]VendorSales `json:"data"`
		} `json:"sales"`
	} `json:"Response"`
	ErrorCode   int    `json:"ErrorCode"`
	ErrorStatus string `json:"ErrorStatus"`
	Message     string `json:"Message"`
}

// VendorComponent contains the live, character-scoped summary for one vendor.
type VendorComponent struct {
	VendorHash          uint32 `json:"vendorHash"`
	VendorLocationIndex int    `json:"vendorLocationIndex"`
	Enabled             bool   `json:"enabled"`
}

// PublicVendorsResponse wraps the multi-vendor Bungie response.
type PublicVendorsResponse struct {
	Response struct {
		Sales struct {
			Data map[string]VendorSales `json:"data"` // key = vendor hash string
		} `json:"sales"`
		VendorGroups struct {
			Data struct{} `json:"data"`
		} `json:"vendorGroups"`
	} `json:"Response"`
	ErrorCode   int    `json:"ErrorCode"`
	ErrorStatus string `json:"ErrorStatus"`
	Message     string `json:"Message"`
}

// VendorSales holds the sale items for one vendor.
type VendorSales struct {
	SaleItems map[string]VendorSaleItem `json:"saleItems"` // key = vendorItemIndex string
}

// VendorSaleItem is one sale item from a vendor.
type VendorSaleItem struct {
	ItemHash uint32           `json:"itemHash"`
	Costs    []VendorItemCost `json:"costs"`
}

// VendorItemCost is the currency cost for one item.
type VendorItemCost struct {
	ItemHash uint32 `json:"itemHash"`
	Quantity int    `json:"quantity"`
}

// ItemQuantity is Bungie's generic item-hash and quantity pair.
type ItemQuantity struct {
	ItemHash uint32 `json:"itemHash"`
	Quantity int    `json:"quantity"`
}

// RecordsProfileResponse contains profile records (component 900).
type RecordsProfileResponse struct {
	Response struct {
		ProfileRecords struct {
			Data struct {
				Records map[string]RecordComponent `json:"records"`
			} `json:"data"`
		} `json:"profileRecords"`
		CharacterRecords struct {
			Data map[string]struct {
				Records map[string]RecordComponent `json:"records"`
			} `json:"data"`
		} `json:"characterRecords"`
	} `json:"Response"`
	ErrorCode   int    `json:"ErrorCode"`
	ErrorStatus string `json:"ErrorStatus"`
	Message     string `json:"Message"`
}

// RecordComponent is the state of a single record/triumph from the API.
type RecordComponent struct {
	State              int               `json:"state"`
	Objectives         []RecordObjective `json:"objectives"`
	IntervalObjectives []RecordObjective `json:"intervalObjectives"`
}

// RecordObjective is one objective within a record.
type RecordObjective struct {
	ObjectiveHash   uint32 `json:"objectiveHash"`
	Progress        int    `json:"progress"`
	CompletionValue int    `json:"completionValue"`
	Complete        bool   `json:"complete"`
	// Visible is a *bool because Bungie omits the field entirely when an
	// objective is visible; only an explicit false means hidden. A plain bool
	// would decode an absent field to false, which is the exact inverse of
	// the real semantics. nil = absent = visible.
	Visible             *bool  `json:"visible"`
	ProgressDescription string `json:"progressDescription"`
}

// CoreSettingsResponse wraps /Platform/Settings/
type CoreSettingsResponse struct {
	Response struct {
		Destiny2CoreSettings struct {
			ActiveSealsRootNodeHash     uint32 `json:"activeSealsRootNodeHash"`
			LegacySealsRootNodeHash     uint32 `json:"legacySealsRootNodeHash"`
			ExoticCatalystsRootNodeHash uint32 `json:"exoticCatalystsRootNodeHash"`
			CraftingRootNodeHash        uint32 `json:"craftingRootNodeHash"`
		} `json:"destiny2CoreSettings"`
	} `json:"Response"`
	ErrorCode   int    `json:"ErrorCode"`
	ErrorStatus string `json:"ErrorStatus"`
	Message     string `json:"Message"`
}

// CoreSettings holds the root presentation node hashes for key game systems.
type CoreSettings struct {
	ActiveSealsRootNodeHash     uint32
	LegacySealsRootNodeHash     uint32
	ExoticCatalystsRootNodeHash uint32
	CraftingRootNodeHash        uint32
}

// MilestoneDefinition is a DestinyMilestoneDefinition entry from the manifest.
type MilestoneDefinition struct {
	Hash              uint32                                       `json:"hash"`
	DisplayProperties DisplayProperties                            `json:"displayProperties"`
	MilestoneType     int                                          `json:"milestoneType"`
	Rewards           map[string]MilestoneRewardCategoryDefinition `json:"rewards"`
}

// MilestoneRewardCategoryDefinition groups reward entries within a milestone.
type MilestoneRewardCategoryDefinition struct {
	RewardEntries map[string]MilestoneRewardEntryDefinition `json:"rewardEntries"`
}

// MilestoneRewardEntryDefinition describes the items represented by one reward entry.
type MilestoneRewardEntryDefinition struct {
	Items []ItemQuantity `json:"items"`
}

// VendorDefinition is the location-bearing subset of DestinyVendorDefinition.
type VendorDefinition struct {
	Hash      uint32                     `json:"hash"`
	Locations []VendorLocationDefinition `json:"locations"`
}

// VendorLocationDefinition points from a vendor location slot to a destination.
type VendorLocationDefinition struct {
	DestinationHash uint32 `json:"destinationHash"`
}

// DestinationDefinition is the display subset of DestinyDestinationDefinition.
type DestinationDefinition struct {
	Hash              uint32            `json:"hash"`
	DisplayProperties DisplayProperties `json:"displayProperties"`
}
