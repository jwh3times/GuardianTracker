package weekly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/efficiency"
	"guardian-tracker/api-service/services/manifest"
	"guardian-tracker/api-service/services/recommendations"
	"guardian-tracker/api-service/services/sources"
)

// fakeManifest satisfies weekly.ManifestRepo.
type fakeManifest struct {
	items              map[uint32]*bungie.InventoryItemDefinition
	destinationHash    uint32
	destinationName    string
	resolveLocationErr error
}

func (f *fakeManifest) GetItemsByHashes(hashes []uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	out := map[uint32]*bungie.InventoryItemDefinition{}
	for _, h := range hashes {
		if def, ok := f.items[h]; ok {
			out[h] = def
		}
	}
	return out, nil
}
func (f *fakeManifest) ResolveVendorLocation(uint32, int) (uint32, string, error) {
	return f.destinationHash, f.destinationName, f.resolveLocationErr
}
func (f *fakeManifest) GetMilestoneDefinitions([]uint32) (map[uint32]*bungie.MilestoneDefinition, error) {
	return map[uint32]*bungie.MilestoneDefinition{}, nil
}
func (f *fakeManifest) GetActivityDefinitions([]uint32) (map[uint32]*bungie.ActivityDefinition, error) {
	return map[uint32]*bungie.ActivityDefinition{}, nil
}
func (f *fakeManifest) GetActivityModifierDefinitions([]uint32) (map[uint32]*bungie.ActivityModifierDefinition, error) {
	return map[uint32]*bungie.ActivityModifierDefinition{}, nil
}

func TestNextDailyReset(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			"before reset same day",
			time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 10, 17, 0, 0, 0, time.UTC),
		},
		{
			"exactly at reset rolls to tomorrow",
			time.Date(2026, 6, 10, 17, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 11, 17, 0, 0, 0, time.UTC),
		},
		{
			"after reset rolls to tomorrow",
			time.Date(2026, 6, 10, 20, 30, 0, 0, time.UTC),
			time.Date(2026, 6, 11, 17, 0, 0, 0, time.UTC),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NextDailyReset(tc.now); !got.Equal(tc.want) {
				t.Errorf("NextDailyReset(%v) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}

func TestToDuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want Duration
	}{
		{0, Duration{}},
		{-5 * time.Minute, Duration{}}, // negative clamps to zero
		{59 * time.Minute, Duration{M: 59}},
		{90 * time.Minute, Duration{H: 1, M: 30}},
		{25 * time.Hour, Duration{D: 1, H: 1}},
		{(3*24 + 2) * time.Hour, Duration{D: 3, H: 2}},
	}
	for _, tc := range cases {
		if got := toDuration(tc.in); got != tc.want {
			t.Errorf("toDuration(%v) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestXurLocationLabel(t *testing.T) {
	if got := xurLocationLabel(xurTowerDestinationHash, "The Last City"); got != "The Tower" {
		t.Errorf("Tower destination label = %q, want The Tower", got)
	}
	if got := xurLocationLabel(123, "Nessus"); got != "Nessus" {
		t.Errorf("unknown destination label = %q, want Nessus", got)
	}
}

func TestGetXurLocation_UsesCharacterVendorCache(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"vendors":{"data":{
			"2190858386":{"vendorHash":2190858386,"vendorLocationIndex":0,"enabled":true}
		}},"sales":{"data":{}}}}`)
	}))
	defer srv.Close()

	c := cache.NewMemoryCache(time.Minute, 0)
	s := NewService(
		bungie.NewClient("k", srv.URL, 100, 100),
		&fakeManifest{destinationHash: xurTowerDestinationHash, destinationName: "The Last City"},
		nil,
		nil,
		c,
		nil,
		recommendations.NewPlanner(nil),
		fakeVersioner{"v-test"},
	)

	for range 2 {
		if got := s.getXurLocation(context.Background(), 3, "member-1", "char-1", "token"); got != "The Tower" {
			t.Fatalf("getXurLocation = %q, want The Tower", got)
		}
	}
	if requests != 1 {
		t.Errorf("character vendor requests = %d, want 1", requests)
	}
}

func TestGetXurLocation_OmitsUnknownLocation(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
	resp := &bungie.CharacterVendorsResponse{}
	resp.Response.Vendors.Data = map[string]bungie.VendorComponent{
		"2190858386": {VendorHash: bungie.XurVendorHash, VendorLocationIndex: -1, Enabled: true},
	}
	c.Set(characterVendorsCacheKey(3, "member-1", "char-1"), resp, time.Minute)
	s := &Service{cache: c, manifest: &fakeManifest{destinationHash: xurTowerDestinationHash}}

	if got := s.getXurLocation(context.Background(), 3, "member-1", "char-1", "token"); got != "" {
		t.Errorf("getXurLocation = %q, want omitted location", got)
	}
}

// The milestone-name vocabulary is owned and tested by services/sources; this
// only pins the delegation.
func TestCategoryFromMilestoneName_DelegatesToSources(t *testing.T) {
	for _, name := range []string{
		"Nightfall: The Ordeal",
		"Weekly Raid Challenge",
		"Featured Dungeon",
		"Something Else Entirely",
	} {
		if got, want := categoryFromMilestoneName(name), sources.MilestoneCategory(name); got != want {
			t.Errorf("categoryFromMilestoneName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestRecommendedActionDifficultyWireUsesCanonicalTiers(t *testing.T) {
	for _, tier := range []sources.DifficultyTier{sources.Easy, sources.Moderate, sources.Challenging, sources.Unrated} {
		blob, err := json.Marshal(RecommendedAction{Diff: tier})
		if err != nil {
			t.Fatalf("Marshal(%q): %v", tier, err)
		}
		want := `"diff":"` + string(tier) + `"`
		if !strings.Contains(string(blob), want) {
			t.Errorf("wire for %q = %s, want %s", tier, blob, want)
		}
	}
}

func TestNewServiceWithClockUsesInjectedUTCClock(t *testing.T) {
	fixed := time.Date(2026, 7, 11, 18, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	s := NewServiceWithClock(nil, nil, nil, nil, cache.NewNoOpCache(), nil, recommendations.NewPlanner(nil), fakeVersioner{"v-test"}, func() time.Time { return fixed })
	if got, want := s.nowUTC(), fixed.UTC(); !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("nowUTC() = %v, want %v", got, want)
	}
	if !XurPresent(s.nowUTC()) {
		t.Fatal("injected Saturday clock should put Xur in the present window")
	}
}

func TestNewServiceRequiresAcquisitionRecommender(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewServiceWithClock accepted a nil AcquisitionRecommender")
		}
	}()
	NewServiceWithClock(nil, nil, nil, nil, cache.NewNoOpCache(), nil, nil, fakeVersioner{"v-test"}, time.Now)
}

func TestBuildTodayActions_CapAtSixAndXurBadges(t *testing.T) {
	s := &Service{}
	now := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC) // Saturday — Xûr present
	pub := &publicWeeklyCache{
		XurPresent:  true,
		XurLeavesAt: time.Date(2026, 6, 16, 17, 0, 0, 0, time.UTC),
		DailyMilestones: []dailyMilestoneItem{
			{Hash: 1, Name: "Daily One"},
			{Hash: 2, Name: "Daily Two"},
		},
		XurItems: []xurItemEnriched{
			{Hash: 10, Name: "Missing Item", Type: "Hand Cannon"},
			{Hash: 11, Name: "Wishlist Item", Type: "Auto Rifle"},
			{Hash: 12, Name: "Owned Item", Type: "Scout Rifle"},
		},
		WeeklyActivities: []weeklyActivityItem{
			{Hash: 20, Name: "Nightfall", Category: "Nightfall", Modifiers: []string{"Champions"}},
			{Hash: 21, Name: "Raid", Category: "Featured Raid"},
			{Hash: 22, Name: "Dungeon", Category: "Featured Dungeon"},
		},
	}
	vendors := []dailyVendorItem{{VendorName: "Banshee-44", ItemName: "Targeting Mod", ItemType: "Weapon Mod"}}
	missing := map[uint32]struct{}{10: {}}
	wishlist := map[uint32]struct{}{11: {}}

	actions := s.buildTodayActions(pub, vendors, missing, wishlist, now, Duration{H: 7}, Duration{D: 3})

	if len(actions) != 6 {
		t.Fatalf("len(actions) = %d, want cap of 6", len(actions))
	}
	var xurBadges []string
	for _, a := range actions {
		if a.Category == "xur" {
			xurBadges = append(xurBadges, a.Badge)
		}
	}
	if len(xurBadges) != 2 {
		t.Fatalf("xur actions = %d, want 2 (owned item excluded)", len(xurBadges))
	}
	if xurBadges[0] != "missing" || xurBadges[1] != "avail-now" {
		t.Errorf("xur badges = %v, want [missing avail-now]", xurBadges)
	}
}

// TestEnrichDailyVendorItems_DropsUnknownItems is the B15 regression: items
// without a manifest definition (or that are not mods) must be dropped, never
// surfaced as "Unknown Mod" rows.
func TestEnrichDailyVendorItems_DropsUnknownItems(t *testing.T) {
	mod := &bungie.InventoryItemDefinition{Hash: 100, ItemType: bungie.ItemTypeMod}
	mod.DisplayProperties.Name = "Targeting Adjuster"
	core := &bungie.InventoryItemDefinition{Hash: 200, ItemType: 0} // not a mod
	core.DisplayProperties.Name = "Enhancement Core"

	s := &Service{manifest: &fakeManifest{items: map[uint32]*bungie.InventoryItemDefinition{
		100: mod,
		200: core,
	}}}

	resp := &bungie.CharacterVendorsResponse{}
	resp.Response.Sales.Data = map[string]bungie.VendorSales{
		"672118013": { // Banshee-44
			SaleItems: map[string]bungie.VendorSaleItem{
				"0": {ItemHash: 100}, // real mod — kept
				"1": {ItemHash: 200}, // enhancement core — dropped (not a mod)
				"2": {ItemHash: 999}, // no manifest def — dropped (B15)
			},
		},
	}

	items := s.enrichDailyVendorItems(resp)
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1; got %+v", len(items), items)
	}
	if items[0].ItemName != "Targeting Adjuster" {
		t.Errorf("ItemName = %q", items[0].ItemName)
	}
}

func TestXurItemHashesAt(t *testing.T) {
	friday := time.Date(2026, 6, 12, 18, 0, 0, 0, time.UTC)    // Xûr present
	wednesday := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC) // Xûr absent

	c := cache.NewMemoryCache(time.Minute, 0)
	c.Set(publicWeeklyCacheKey, &publicWeeklyCache{
		XurPresent: true,
		XurItems:   []xurItemEnriched{{Hash: 111}, {Hash: 222}},
	}, time.Minute)
	s := &Service{cache: c}

	got := s.xurItemHashesAt(context.Background(), friday)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if _, ok := got[111]; !ok {
		t.Error("hash 111 missing")
	}

	// Outside Xûr's window the lookup short-circuits to empty.
	if got := s.xurItemHashesAt(context.Background(), wednesday); len(got) != 0 {
		t.Errorf("expected empty set on a Wednesday, got %d entries", len(got))
	}
}

// --- helpers for TestBuildMilestones_SetsMissingForRaid ---

type fakeBucketSource struct {
	rows []manifest.CollectibleWithItem
}

func (f fakeBucketSource) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	return f.rows, nil
}

type fakeVersioner struct{ v string }

func (f fakeVersioner) Version() string { return f.v }

func raidRow(itemHash uint32) manifest.CollectibleWithItem {
	item := &bungie.InventoryItemDefinition{Hash: itemHash}
	item.Inventory.TierType = bungie.TierTypeLegendary
	return manifest.CollectibleWithItem{
		Collectible: bungie.CollectibleDefinition{
			Hash: itemHash, ItemHash: itemHash, SourceHash: 7,
			SourceString: "Vault of Glass Raid",
		},
		Item: item,
	}
}

func TestBuildMilestones_SetsMissingForRaid(t *testing.T) {
	eng := efficiency.NewEngine(
		fakeBucketSource{rows: []manifest.CollectibleWithItem{
			raidRow(100), raidRow(101), raidRow(102),
		}},
		fakeVersioner{"v1"},
	)
	eng.BuildIndex()

	pub := &publicWeeklyCache{
		MilestoneHashes:  []uint32{10, 11},
		MilestoneNames:   map[uint32]string{10: "Vault of Glass", 11: "Clan Rewards"},
		MilestoneRewards: map[uint32]string{10: "Pinnacle Gear", 11: "XP"},
	}
	missing := map[uint32]struct{}{100: {}, 102: {}}

	ms := buildMilestones(pub, eng, missing)
	byName := map[string]Milestone{}
	for _, m := range ms {
		byName[m.Name] = m
	}
	if got := byName["Vault of Glass"].Missing; got == nil || *got != 2 {
		t.Errorf("Vault of Glass Missing = %v, want 2", got)
	}
	if byName["Clan Rewards"].Missing != nil {
		t.Error("non-raid milestone Missing should be nil")
	}

	// nil engine → all nil, no panic.
	for _, m := range buildMilestones(pub, nil, missing) {
		if m.Missing != nil {
			t.Errorf("nil engine should leave Missing nil, got %v", m.Missing)
		}
	}
}
