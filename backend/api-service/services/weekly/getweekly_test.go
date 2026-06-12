package weekly

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
)

// richManifest is a configurable ManifestRepo for the public-weekly fetch paths.
type richManifest struct {
	items      map[uint32]*bungie.InventoryItemDefinition
	milestones map[uint32]*bungie.MilestoneDefinition
	activities map[uint32]*bungie.ActivityDefinition
	modifiers  map[uint32]*bungie.ActivityModifierDefinition
}

func (f *richManifest) GetItemsByHashes([]uint32) (map[uint32]*bungie.InventoryItemDefinition, error) {
	return f.items, nil
}
func (f *richManifest) GetMilestoneDefinitions([]uint32) (map[uint32]*bungie.MilestoneDefinition, error) {
	return f.milestones, nil
}
func (f *richManifest) GetActivityDefinitions([]uint32) (map[uint32]*bungie.ActivityDefinition, error) {
	return f.activities, nil
}
func (f *richManifest) GetActivityModifierDefinitions([]uint32) (map[uint32]*bungie.ActivityModifierDefinition, error) {
	return f.modifiers, nil
}

func weeklyService(t *testing.T, body string, m ManifestRepo) *Service {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewService(bungie.NewClient("k", srv.URL, 100, 100), m, nil, nil,
		cache.NewMemoryCache(time.Minute, time.Minute))
}

func TestFetchXurInventory_EnrichesFromManifest(t *testing.T) {
	gjally := &bungie.InventoryItemDefinition{Hash: 100, ItemType: 3, ItemSubType: 10}
	gjally.DisplayProperties.Name = "Gjallarhorn"
	gjally.Inventory.TierType = 6 // Exotic
	mani := &richManifest{items: map[uint32]*bungie.InventoryItemDefinition{100: gjally}}

	svc := weeklyService(t, `{"ErrorCode":1,"Response":{"sales":{"data":{
		"2190858386":{"saleItems":{"0":{"itemHash":100,"costs":[{"itemHash":3,"quantity":29}]}}}}}}}`, mani)

	pub := &publicWeeklyCache{}
	svc.fetchXurInventory(context.Background(), pub)
	if len(pub.XurItems) != 1 {
		t.Fatalf("XurItems = %+v, want 1", pub.XurItems)
	}
	it := pub.XurItems[0]
	if it.Name != "Gjallarhorn" || it.Rarity != "exotic" || it.Type != "Rocket Launcher" {
		t.Errorf("enriched item = %+v", it)
	}
	if it.Cost != "29 Strange Coin" {
		t.Errorf("cost = %q", it.Cost)
	}
	if pub.Degraded {
		t.Error("should not be degraded when manifest resolves items")
	}
}

func TestFetchXurInventory_DegradedWithoutManifest(t *testing.T) {
	svc := weeklyService(t, `{"ErrorCode":1,"Response":{"sales":{"data":{
		"2190858386":{"saleItems":{"0":{"itemHash":100}}}}}}}`, nil)

	pub := &publicWeeklyCache{}
	svc.fetchXurInventory(context.Background(), pub)
	if !pub.Degraded {
		t.Error("expected degraded when manifest is unavailable")
	}
	if len(pub.XurItems) != 1 || pub.XurItems[0].Name != "Unknown Item" {
		t.Errorf("degraded items = %+v", pub.XurItems)
	}
}

func TestFetchMilestones_EnrichesWeeklyActivities(t *testing.T) {
	msDef := &bungie.MilestoneDefinition{Hash: 4001, MilestoneType: bungie.MilestoneTypeWeekly}
	msDef.DisplayProperties.Name = "Vanguard Ops"
	actDef := &bungie.ActivityDefinition{Hash: 50}
	actDef.DisplayProperties.Name = "Nightfall: The Ordeal"
	modDef := &bungie.ActivityModifierDefinition{Hash: 7}
	modDef.DisplayProperties.Name = "Champion Foes"
	mani := &richManifest{
		milestones: map[uint32]*bungie.MilestoneDefinition{4001: msDef},
		activities: map[uint32]*bungie.ActivityDefinition{50: actDef},
		modifiers:  map[uint32]*bungie.ActivityModifierDefinition{7: modDef},
	}

	svc := weeklyService(t, `{"ErrorCode":1,"Response":{"4001":{"milestoneHash":4001,
		"activities":[{"activityHash":50,"modifierHashes":[7]}]}}}`, mani)

	pub := &publicWeeklyCache{}
	svc.fetchMilestones(context.Background(), pub)
	if pub.MilestoneNames[4001] != "Vanguard Ops" {
		t.Errorf("milestone name = %q", pub.MilestoneNames[4001])
	}
	if len(pub.WeeklyActivities) != 1 {
		t.Fatalf("WeeklyActivities = %+v, want 1", pub.WeeklyActivities)
	}
	if got := pub.WeeklyActivities[0].Modifiers; len(got) != 1 || got[0] != "Champion Foes" {
		t.Errorf("modifiers = %v", got)
	}
}

func TestFetchMilestones_DegradedWithoutManifest(t *testing.T) {
	svc := weeklyService(t, `{"ErrorCode":1,"Response":{"4001":{"milestoneHash":4001}}}`, nil)
	pub := &publicWeeklyCache{}
	svc.fetchMilestones(context.Background(), pub)
	if !pub.Degraded {
		t.Error("expected degraded without a manifest")
	}
	if pub.MilestoneNames[4001] == "" {
		t.Error("expected a fallback milestone label")
	}
}

func TestNextXurArrival(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			"wednesday → coming friday",
			time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC), // Wed
			time.Date(2026, 6, 12, 17, 0, 0, 0, time.UTC), // Fri
		},
		{
			"friday before 17:00 → same day",
			time.Date(2026, 6, 12, 9, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 12, 17, 0, 0, 0, time.UTC),
		},
		{
			"friday after 17:00 → next friday",
			time.Date(2026, 6, 12, 18, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 19, 17, 0, 0, 0, time.UTC),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NextXurArrival(tc.now); !got.Equal(tc.want) {
				t.Errorf("NextXurArrival(%v) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}

// TestGetWeekly_PublicPath exercises the no-token assembly path: the public
// Bungie calls fail (httptest returns Bungie error codes), so the service must
// degrade gracefully and still return a fully-formed (partial) weekly payload.
func TestGetWeekly_PublicPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ErrorCode 5 → every public fetch errors and is handled, not fatal.
		w.Write([]byte(`{"ErrorCode":5,"ErrorStatus":"SystemDisabled"}`))
	}))
	defer srv.Close()

	svc := NewService(
		bungie.NewClient("k", srv.URL, 100, 100),
		&fakeManifest{items: map[uint32]*bungie.InventoryItemDefinition{}},
		nil, // collections — unused when bungieToken is empty
		nil, // wishlist — nil-checked
		cache.NewMemoryCache(time.Minute, time.Minute),
	)

	res, err := svc.GetWeekly(context.Background(), 3, "4611686018467260757", "")
	if err != nil {
		t.Fatalf("GetWeekly: %v", err)
	}
	if res == nil {
		t.Fatal("GetWeekly returned nil payload")
	}
	if res.ResetLabel == "" {
		t.Error("ResetLabel should be populated even on the degraded path")
	}
	if res.ResetAt.IsZero() {
		t.Error("ResetAt should be set")
	}
	// buildRecommended always returns at least the fallback action.
	if len(res.Recommended) == 0 {
		t.Error("expected at least one recommended action (fallback)")
	}
}
