// Package e2efixture provides the hermetic Bungie API and manifest used by
// browser tests. It is intentionally imported only by the fake-Bungie command
// and its tests; production code never serves these fixtures.
package e2efixture

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	ManifestVersion     = "e2e-fixture-1"
	ManifestContentName = "world_sql_content_e2e.content"

	MembershipID   = "4611686018400000001"
	MembershipType = 3
	DisplayName    = "E2E Guardian"
	CharacterID    = "2305843009300000001"

	XurVendorHash uint32 = 2190858386
)

const (
	fatebringerHash  uint32 = 100
	midnightCoupHash uint32 = 101
	gjallarhornHash  uint32 = 200
	testHelmetHash   uint32 = 300
	testShipHash     uint32 = 400
	testShaderHash   uint32 = 401
	boldEndingsHash  uint32 = 500
	sunshotHash      uint32 = 600
	strangeCoinHash  uint32 = 700
	testingModHash   uint32 = 701

	combinedRecordsRootHash uint32 = 9000
	catalystNodeHash        uint32 = 9001
	patternNodeHash         uint32 = 9002
	sunshotCatalystHash     uint32 = 9100
	boldEndingsPatternHash  uint32 = 9101
	activeSealsRootHash     uint32 = 9200
	dredgenSealHash         uint32 = 9300
	completedTriumphHash    uint32 = 9301
	inProgressTriumphHash   uint32 = 9302
	sealCompletionHash      uint32 = 9400

	weeklyMilestoneHash  uint32 = 9500
	weeklyActivityHash   uint32 = 9501
	weeklyModifierHash   uint32 = 9502
	towerDestinationHash uint32 = 9600
)

var manifestTables = []string{
	"DestinyInventoryItemDefinition",
	"DestinyCollectibleDefinition",
	"DestinyPresentationNodeDefinition",
	"DestinyPlugSetDefinition",
	"DestinyRecordDefinition",
	"DestinyVendorDefinition",
	"DestinyDestinationDefinition",
	"DestinyMilestoneDefinition",
	"DestinyActivityDefinition",
	"DestinyActivityModifierDefinition",
	"DestinySandboxPerkDefinition",
}

type manifestRow struct {
	hash uint32
	json any
}

// ManifestTables returns the exact set of production-queryable tables in the
// generated archive. The returned slice is a copy so tests cannot mutate it.
func ManifestTables() []string {
	return append([]string(nil), manifestTables...)
}

// GenerateManifestZip creates a tiny Bungie-style mobile world content archive.
// The SQLite file is built at runtime and the returned zip contains one .content
// entry, matching ManifestService's production extraction path.
func GenerateManifestZip() ([]byte, error) {
	tempDir, err := os.MkdirTemp("", "guardian-e2e-manifest-")
	if err != nil {
		return nil, fmt.Errorf("create manifest temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, ManifestContentName)
	if err := writeManifestDB(dbPath); err != nil {
		return nil, err
	}
	dbBytes, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("read generated manifest: %w", err)
	}

	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	header := &zip.FileHeader{
		Name:     ManifestContentName,
		Method:   zip.Deflate,
		Modified: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
	entry, err := zw.CreateHeader(header)
	if err != nil {
		return nil, fmt.Errorf("create manifest zip entry: %w", err)
	}
	if _, err := entry.Write(dbBytes); err != nil {
		return nil, fmt.Errorf("write manifest zip entry: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close manifest zip: %w", err)
	}
	return out.Bytes(), nil
}

func writeManifestDB(path string) error {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return fmt.Errorf("open generated manifest: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = db.Close()
		}
	}()

	rows := fixtureRows()
	tables := ManifestTables()
	sort.Strings(tables)
	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, json TEXT NOT NULL)", table)); err != nil {
			return fmt.Errorf("create %s: %w", table, err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin manifest fixture transaction: %w", err)
	}
	for _, table := range tables {
		for _, row := range rows[table] {
			blob, err := json.Marshal(row.json)
			if err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("marshal %s fixture %d: %w", table, row.hash, err)
			}
			query := fmt.Sprintf("INSERT INTO %s (id, json) VALUES (?, ?)", table)
			if _, err := tx.Exec(query, signedHash(row.hash), string(blob)); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert %s fixture %d: %w", table, row.hash, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit manifest fixtures: %w", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("close generated manifest: %w", err)
	}
	closed = true
	return nil
}

func signedHash(hash uint32) int64 {
	if hash > math.MaxInt32 {
		return int64(hash) - (1 << 32)
	}
	return int64(hash)
}

func fixtureRows() map[string][]manifestRow {
	items := []manifestRow{
		{fatebringerHash, weapon(fatebringerHash, "Fatebringer", "A legendary hand cannon from the Vault of Glass.", "/e2e/fatebringer.png", 9, 5)},
		{midnightCoupHash, weapon(midnightCoupHash, "Midnight Coup", "A trusted hand cannon from the Leviathan.", "/e2e/midnight-coup.png", 9, 5)},
		{gjallarhornHash, weapon(gjallarhornHash, "Gjallarhorn", "An exotic rocket launcher with an enduring legacy.", "/e2e/gjallarhorn.png", 10, 6)},
		{testHelmetHash, armor(testHelmetHash, "Helm of Tests", "A deterministic Warlock helmet.", "/e2e/helmet.png", 2)},
		{testShipHash, simpleItem(testShipHash, "Test Ship", "A ship for crossing the test harness.", "/e2e/ship.png", 21, 0, 5)},
		{testShaderHash, simpleItem(testShaderHash, "Test Shader", "Applies deterministic colors.", "/e2e/shader.png", 19, 20, 5)},
		{boldEndingsHash, weapon(boldEndingsHash, "Bold Endings", "A craftable hand cannon pattern fixture.", "/e2e/bold-endings.png", 9, 5)},
		{sunshotHash, weapon(sunshotHash, "Sunshot", "An exotic hand cannon that fires explosive rounds.", "/e2e/sunshot.png", 9, 6)},
		{strangeCoinHash, simpleItem(strangeCoinHash, "Strange Coin", "Currency accepted by Xûr.", "/e2e/strange-coin.png", 10, 0, 5)},
		{testingModHash, simpleItem(testingModHash, "Testing Targeting", "A rotating test mod.", "/e2e/testing-mod.png", 19, 0, 5)},
		{11000, plug(11000, "Arrowhead Brake", "Lightly vented barrel.", "barrels", nil, nil)},
		{11001, plug(11001, "Corkscrew Rifling", "Balanced barrel.", "barrels", nil, nil)},
		{11002, plug(11002, "Explosive Payload", "Projectiles create an area-of-effect detonation.", "frames", nil, nil)},
		{11003, plug(11003, "Firefly", "Precision kills increase reload speed and explode the target.", "frames", nil, nil)},
		{11004, plug(11004, "Sunshot Catalyst", "Upgrades this weapon to a Masterwork.", "catalysts", []uint32{7101}, []uint32{13000})},
	}

	// Fatebringer has two production-shaped perk sockets; Sunshot has a catalyst
	// socket whose objective hash links its record to the effect text below.
	items[0].json.(map[string]any)["sockets"] = map[string]any{
		"socketEntries": []any{
			map[string]any{"randomizedPlugSetHash": uint32(12000)},
			map[string]any{"randomizedPlugSetHash": uint32(12001)},
		},
		"socketCategories": []any{
			map[string]any{"socketCategoryHash": uint32(4241085061), "socketIndexes": []int{0, 1}},
		},
	}
	items[7].json.(map[string]any)["sockets"] = map[string]any{
		"socketEntries": []any{map[string]any{"reusablePlugSetHash": uint32(12003)}},
		"socketCategories": []any{
			map[string]any{"socketCategoryHash": uint32(4241085061), "socketIndexes": []int{0}},
		},
	}

	return map[string][]manifestRow{
		"DestinyInventoryItemDefinition": items,
		"DestinyCollectibleDefinition": {
			{1000, collectible(1000, fatebringerHash, "Fatebringer", "Complete the Vault of Glass raid.")},
			{1001, collectible(1001, midnightCoupHash, "Midnight Coup", "Earned from Onslaught.")},
			{2000, collectible(2000, gjallarhornHash, "Gjallarhorn", "Available from Xûr.")},
			{3000, collectible(3000, testHelmetHash, "Helm of Tests", "Earned from a test engram.")},
			{4000, collectible(4000, testShipHash, "Test Ship", "Earned while exploring.")},
			{4001, collectible(4001, testShaderHash, "Test Shader", "Available from a Tower vendor.")},
			{5000, collectible(5000, boldEndingsHash, "Bold Endings", "Extract Deepsight Resonance.")},
		},
		"DestinyPresentationNodeDefinition": {
			{1, node(1, "Items", []uint32{10, 20, 40}, nil, nil, 0)},
			{10, node(10, "Weapons", []uint32{11, 12, 13}, nil, nil, 0)},
			{11, node(11, "Hand Cannons", nil, []uint32{1000, 1001}, nil, 0)},
			{12, node(12, "Rocket Launchers", nil, []uint32{2000}, nil, 0)},
			{13, node(13, "Craftable Weapons", nil, []uint32{5000}, nil, 0)},
			{20, node(20, "Armor", []uint32{21}, nil, nil, 0)},
			{21, node(21, "Helmets", nil, []uint32{3000}, nil, 0)},
			{40, node(40, "Cosmetics", []uint32{41, 42}, nil, nil, 0)},
			{41, node(41, "Ships", nil, []uint32{4000}, nil, 0)},
			{42, node(42, "Shaders", nil, []uint32{4001}, nil, 0)},
			{combinedRecordsRootHash, node(combinedRecordsRootHash, "Patterns & Catalysts", []uint32{catalystNodeHash, patternNodeHash}, nil, nil, 0)},
			{catalystNodeHash, node(catalystNodeHash, "Exotic Catalysts", nil, nil, []uint32{sunshotCatalystHash}, 0)},
			{patternNodeHash, node(patternNodeHash, "Weapon Patterns", nil, nil, []uint32{boldEndingsPatternHash}, 0)},
			{activeSealsRootHash, node(activeSealsRootHash, "Active Seals", []uint32{dredgenSealHash}, nil, nil, 0)},
			{dredgenSealHash, node(dredgenSealHash, "Dredgen", nil, nil, []uint32{completedTriumphHash, inProgressTriumphHash}, sealCompletionHash)},
		},
		"DestinyPlugSetDefinition": {
			{12000, plugSet(12000, 11000, 11001)},
			{12001, plugSet(12001, 11002, 11003)},
			{12003, plugSet(12003, 11004)},
		},
		"DestinyRecordDefinition": {
			{sunshotCatalystHash, record(sunshotCatalystHash, "Sunshot Catalyst", "Defeat enemies using Sunshot.", "Exotic Catalysts", "Found in strikes and the Crucible.", []uint32{7101})},
			{boldEndingsPatternHash, record(boldEndingsPatternHash, "Bold Endings", "Extract this weapon's pattern.", "Weapon Pattern", "Weapon Patterns", nil)},
			{completedTriumphHash, record(completedTriumphHash, "Prestige Worldwide", "Complete the introductory Gambit triumph.", "Triumph", "Dredgen", nil)},
			{inProgressTriumphHash, record(inProgressTriumphHash, "Tested Resolve", "Complete each deterministic Gambit objective.", "Triumph", "Dredgen", nil)},
			{sealCompletionHash, record(sealCompletionHash, "Dredgen Completion", "Complete the Dredgen seal.", "Triumph", "Dredgen", nil)},
		},
		"DestinyVendorDefinition": {
			{XurVendorHash, map[string]any{"hash": XurVendorHash, "locations": []any{map[string]any{"destinationHash": towerDestinationHash}}}},
		},
		"DestinyDestinationDefinition": {
			{towerDestinationHash, map[string]any{"hash": towerDestinationHash, "displayProperties": display("The Tower", "Humanity's Last City.", "/e2e/tower.png")}},
		},
		"DestinyMilestoneDefinition": {
			{weeklyMilestoneHash, map[string]any{
				"hash": weeklyMilestoneHash, "displayProperties": display("Vault of Glass", "Complete the featured raid.", "/e2e/vog.png"), "milestoneType": 3,
				"rewards": map[string]any{"featured": map[string]any{"rewardEntries": map[string]any{"pinnacle": map[string]any{"items": []any{map[string]any{"itemHash": fatebringerHash, "quantity": 1}}}}}},
			}},
		},
		"DestinyActivityDefinition": {
			{weeklyActivityHash, map[string]any{"hash": weeklyActivityHash, "displayProperties": display("Vault of Glass: Test Run", "A deterministic raid activity.", "/e2e/activity.png"), "activityTypeHash": uint32(2043403989)}},
		},
		"DestinyActivityModifierDefinition": {
			{weeklyModifierHash, map[string]any{"hash": weeklyModifierHash, "displayProperties": display("E2E Singe", "Incoming and outgoing test damage is increased.", "/e2e/modifier.png")}},
		},
		"DestinySandboxPerkDefinition": {
			{13000, map[string]any{"hash": uint32(13000), "displayProperties": display("Sun Blast", "Increases range and stability.", "/e2e/sun-blast.png"), "isDisplayable": true}},
		},
	}
}

func display(name, description, icon string) map[string]any {
	return map[string]any{"name": name, "description": description, "icon": icon}
}

func simpleItem(hash uint32, name, description, icon string, itemType, subType, tier int) map[string]any {
	return map[string]any{
		"hash": hash, "displayProperties": display(name, description, icon),
		"itemType": itemType, "itemSubType": subType,
		"inventory":      map[string]any{"tierType": tier},
		"equippingBlock": map[string]any{"equipmentSlotTypeHash": uint32(0)},
	}
}

func weapon(hash uint32, name, description, icon string, subType, tier int) map[string]any {
	return simpleItem(hash, name, description, icon, 3, subType, tier)
}

func armor(hash uint32, name, description, icon string, classType int) map[string]any {
	item := simpleItem(hash, name, description, icon, 2, 0, 5)
	item["classType"] = classType
	item["equippingBlock"] = map[string]any{"equipmentSlotTypeHash": uint32(3448274439)}
	return item
}

func collectible(hash, itemHash uint32, name, source string) map[string]any {
	return map[string]any{
		"hash": hash, "itemHash": itemHash, "displayProperties": display(name, source, fmt.Sprintf("/e2e/collectible-%d.png", hash)),
		"sourceString": source,
	}
}

func node(hash uint32, name string, childNodes, collectibles, records []uint32, completion uint32) map[string]any {
	presentationChildren := make([]any, 0, len(childNodes))
	for _, child := range childNodes {
		presentationChildren = append(presentationChildren, map[string]any{"presentationNodeHash": child})
	}
	collectibleChildren := make([]any, 0, len(collectibles))
	for _, child := range collectibles {
		collectibleChildren = append(collectibleChildren, map[string]any{"collectibleHash": child})
	}
	recordChildren := make([]any, 0, len(records))
	for _, child := range records {
		recordChildren = append(recordChildren, map[string]any{"recordHash": child})
	}
	return map[string]any{
		"hash": hash, "displayProperties": display(name, name+" E2E fixture.", fmt.Sprintf("/e2e/node-%d.png", hash)),
		"children":             map[string]any{"presentationNodes": presentationChildren, "collectibles": collectibleChildren, "records": recordChildren},
		"completionRecordHash": completion,
	}
}

func plug(hash uint32, name, description, category string, objectives, perks []uint32) map[string]any {
	perkRows := make([]any, 0, len(perks))
	for _, perkHash := range perks {
		perkRows = append(perkRows, map[string]any{"perkHash": perkHash})
	}
	return map[string]any{
		"hash": hash, "displayProperties": display(name, description, fmt.Sprintf("/e2e/plug-%d.png", hash)), "itemType": 19,
		"inventory": map[string]any{"tierType": 5}, "plug": map[string]any{"plugCategoryIdentifier": category},
		"objectives": map[string]any{"objectiveHashes": objectives}, "perks": perkRows,
	}
}

func plugSet(hash uint32, plugHashes ...uint32) map[string]any {
	items := make([]any, 0, len(plugHashes))
	for _, plugHash := range plugHashes {
		items = append(items, map[string]any{"plugItemHash": plugHash, "currentlyCanRoll": true})
	}
	return map[string]any{"hash": hash, "reusablePlugItems": items}
}

func record(hash uint32, name, description, recordType, source string, objectives []uint32) map[string]any {
	return map[string]any{
		"hash": hash, "displayProperties": display(name, description, fmt.Sprintf("/e2e/record-%d.png", hash)),
		"recordTypeName": recordType, "stateInfo": map[string]any{"obscuredDescription": source},
		"objectiveHashes": objectives, "intervalObjectives": []any{}, "forTitleGilding": false,
	}
}
