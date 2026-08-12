package weekly

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/efficiency"
	"guardian-tracker/api-service/services/manifest"
)

// These tests cover GetWeekly's composition — the ~120 lines that call the
// well-tested leaf helpers. Before the MissingItemReader seam, none of this was
// reachable: the authenticated branch dereferenced a concrete
// *collections.Service, which needs a real manifest service and a real SQLite
// manifest to build, so the only GetWeekly test passed nil and took the
// no-token path.

const (
	testMembershipType = 3
	testMembershipID   = "member-1"
	testCharacterID    = "char-hunter"
)

// A Friday after 17:00 UTC — Xûr is in town, so the Xûr branches are live.
var testWeeklyNow = time.Date(2026, 6, 12, 18, 0, 0, 0, time.UTC)

// fakeMissingItems is the stand-in the seam exists to make possible. errOnly
// cases return (nil, err) to exercise the degrade path.
type fakeMissingItems struct {
	hashes map[uint32]struct{}
	err    error
	calls  int
}

func (f *fakeMissingItems) GetMissingItemHashes(_ context.Context, _ int, _, _ string) (map[uint32]struct{}, error) {
	f.calls++
	return f.hashes, f.err
}

type fakeWishlist struct {
	userID  int64
	items   []WishlistItem
	userErr error
	listErr error
}

func (f fakeWishlist) GetUserID(context.Context, string) (int64, error) {
	return f.userID, f.userErr
}

func (f fakeWishlist) List(context.Context, int64) ([]WishlistItem, error) {
	return f.items, f.listErr
}

// weeklyFixture seeds every cache entry GetWeekly's authenticated path reads, so
// the composition runs against a nil Bungie client with no HTTP server: the five
// inputs are stated as values instead of as JSON blobs, and the leaf fetchers
// (which have their own tests) stay out of the way.
//
// Every entry is written through the service's own key constructor. A key format
// change then breaks these tests loudly, rather than quietly seeding an entry
// nothing reads and leaving the test asserting against the fetch path.
type weeklyFixture struct {
	pub              *publicWeeklyCache
	roster           *characterRoster
	dailyVendors     []dailyVendorItem
	characterVendors *bungie.CharacterVendorsResponse
	liveVendorItems  map[uint32]string
	missing          MissingItemReader
	wishlist         WishlistReader
	engine           *efficiency.Engine
	manifest         ManifestRepo
}

func (f weeklyFixture) service(t *testing.T) *Service {
	t.Helper()

	c := cache.NewMemoryCache(time.Minute, 0)
	missing := f.missing
	if missing == nil {
		missing = &fakeMissingItems{hashes: map[uint32]struct{}{}}
	}
	s := NewServiceWithClock(
		nil, // bungie — every read below is served from the seeded cache
		f.manifest, missing, f.wishlist, c, f.engine,
		fakeVersioner{"v-test"},
		func() time.Time { return testWeeklyNow },
	)

	pub := f.pub
	if pub == nil {
		pub = &publicWeeklyCache{}
	}
	c.Set(publicWeeklyCacheKey, pub, time.Minute)

	roster := f.roster
	if roster == nil {
		roster = &characterRoster{
			Characters: map[string]bungie.CharacterComponent{
				testCharacterID: {CharacterID: testCharacterID, ClassType: 1},
			},
			PrimaryID: testCharacterID,
		}
	}
	c.Set(rosterCacheKey(testMembershipType, testMembershipID), *roster, time.Minute)

	// Seeded even when empty: an unseeded key falls through to the fetch path,
	// which would reach a nil Bungie client instead of asserting composition.
	c.Set(s.dailyVendorsCacheKey(testMembershipType, testMembershipID, testCharacterID), f.dailyVendors, time.Minute)
	c.Set(liveVendorItemsCacheKey(testMembershipType, testMembershipID, testCharacterID), f.liveVendorItems, time.Minute)
	if f.characterVendors != nil {
		c.Set(characterVendorsCacheKey(testMembershipType, testMembershipID, testCharacterID), f.characterVendors, time.Minute)
	}
	return s
}

func (f weeklyFixture) get(t *testing.T) *Weekly {
	t.Helper()
	res, err := f.service(t).GetWeekly(
		context.Background(), testMembershipType, testMembershipID, "bungie-token", testCharacterID)
	if err != nil {
		t.Fatalf("GetWeekly: %v", err)
	}
	if res == nil {
		t.Fatal("GetWeekly returned a nil payload")
	}
	return res
}

func xurPub(items ...xurItemEnriched) *publicWeeklyCache {
	return &publicWeeklyCache{
		XurPresent:  true,
		XurLeavesAt: testWeeklyNow.Add(48 * time.Hour),
		XurItems:    items,
	}
}

// The missing set is per-user and arrives from the seam; intersecting it with
// the shared public Xûr inventory is the join that makes the payload user
// specific. Nothing asserted it end to end before.
func TestGetWeekly_StampsMissingOnXurItems(t *testing.T) {
	res := weeklyFixture{
		pub: xurPub(
			xurItemEnriched{Hash: 100, Name: "Gjallarhorn"},
			xurItemEnriched{Hash: 200, Name: "Hard Light"},
		),
		missing: &fakeMissingItems{hashes: map[uint32]struct{}{100: {}}},
	}.get(t)

	if res.Xur == nil {
		t.Fatal("Xur block absent while Xûr is present")
	}
	if len(res.Xur.Items) != 2 {
		t.Fatalf("Xur items = %d, want 2", len(res.Xur.Items))
	}
	if !res.Xur.Items[0].Missing {
		t.Error("Gjallarhorn is in the missing set but was not stamped Missing")
	}
	if res.Xur.Items[1].Missing {
		t.Error("Hard Light is not in the missing set but was stamped Missing")
	}
}

// Location comes from the second fan-out goroutine and is resolved through the
// manifest. It reaches the payload only if both halves of the fan-out land.
func TestGetWeekly_XurLocationReachesPayload(t *testing.T) {
	vendors := &bungie.CharacterVendorsResponse{}
	vendors.Response.Vendors.Data = map[string]bungie.VendorComponent{
		"2190858386": {VendorHash: bungie.XurVendorHash, VendorLocationIndex: 0, Enabled: true},
	}

	res := weeklyFixture{
		pub:              xurPub(xurItemEnriched{Hash: 100, Name: "Gjallarhorn"}),
		characterVendors: vendors,
		manifest: &fakeManifest{
			destinationHash: xurTowerDestinationHash,
			destinationName: "The Last City",
		},
	}.get(t)

	if res.Xur == nil {
		t.Fatal("Xur block absent while Xûr is present")
	}
	if res.Xur.Location != "The Tower" {
		t.Errorf("Xur.Location = %q, want The Tower", res.Xur.Location)
	}
}

// In DailyActions the precedence is the opposite way round from
// buildRecommended: an item that is missing keeps the "missing" badge even when
// it is also wishlisted, and only a wishlisted-but-owned item reads "avail-now".
// Both rules are real; neither was asserted through GetWeekly.
func TestGetWeekly_DailyActionBadgePrecedence(t *testing.T) {
	res := weeklyFixture{
		pub: xurPub(
			xurItemEnriched{Hash: 100, Name: "Gjallarhorn", Type: "Rocket Launcher"},
			xurItemEnriched{Hash: 200, Name: "Hard Light", Type: "Auto Rifle"},
		),
		// 100 is both missing and wishlisted; 200 is wishlisted only.
		missing:  &fakeMissingItems{hashes: map[uint32]struct{}{100: {}}},
		wishlist: fakeWishlist{userID: 7, items: []WishlistItem{{ItemHash: 100}, {ItemHash: 200}}},
	}.get(t)

	byID := map[string]DailyAction{}
	for _, a := range res.DailyActions {
		byID[a.ID] = a
	}
	if got := byID["daily-xur-0"].Badge; got != "missing" {
		t.Errorf("missing+wishlisted badge = %q, want missing", got)
	}
	if got := byID["daily-xur-1"].Badge; got != "avail-now" {
		t.Errorf("wishlisted-only badge = %q, want avail-now", got)
	}
}

// With a built index and a non-empty missing set the engine wins; the legacy
// Xûr-only heuristic is the fallback. Diff also pins the sources.Difficulty
// call that replaced collections.ClassifyDifficulty.
func TestGetWeekly_PrefersEngineRanking(t *testing.T) {
	eng := efficiency.NewEngine(
		fakeBucketSource{rows: []manifest.CollectibleWithItem{
			raidRow(100), raidRow(101), raidRow(102),
		}},
		fakeVersioner{"v-test"},
	)
	eng.BuildIndex()

	res := weeklyFixture{
		pub: &publicWeeklyCache{
			MilestoneHashes: []uint32{10},
			MilestoneNames:  map[uint32]string{10: "Vault of Glass"},
		},
		missing: &fakeMissingItems{hashes: map[uint32]struct{}{100: {}, 102: {}}},
		engine:  eng,
	}.get(t)

	if len(res.Recommended) == 0 {
		t.Fatal("no recommended actions")
	}
	if res.Recommended[0].ID != "eff-7" {
		t.Errorf("Recommended[0].ID = %q, want the engine action eff-7 (fell back to buildRecommended)",
			res.Recommended[0].ID)
	}
	if res.Recommended[0].Diff != "Challenging" {
		t.Errorf("Diff = %q, want Challenging from the raid source string", res.Recommended[0].Diff)
	}
}

// Degraded means "names and labels are placeholders". It is decided while
// building the public payload and must survive assembly to reach the client.
func TestGetWeekly_PropagatesDegraded(t *testing.T) {
	res := weeklyFixture{pub: &publicWeeklyCache{Degraded: true}}.get(t)
	if !res.Degraded {
		t.Error("Degraded did not propagate from the public payload")
	}

	fresh := weeklyFixture{pub: &publicWeeklyCache{}}.get(t)
	if fresh.Degraded {
		t.Error("Degraded set without a degraded public payload")
	}
}

// FetchedAt drives the frontend's freshness chip. A zero value would render as
// year 1, so assembly falls back to now.
func TestGetWeekly_FetchedAtFallsBackToNow(t *testing.T) {
	res := weeklyFixture{pub: &publicWeeklyCache{}}.get(t)
	if !res.FetchedAt.Equal(testWeeklyNow) {
		t.Errorf("FetchedAt = %v, want the fallback to now (%v)", res.FetchedAt, testWeeklyNow)
	}

	fetched := testWeeklyNow.Add(-30 * time.Minute)
	res = weeklyFixture{pub: &publicWeeklyCache{FetchedAt: fetched}}.get(t)
	if !res.FetchedAt.Equal(fetched) {
		t.Errorf("FetchedAt = %v, want the payload's own timestamp (%v)", res.FetchedAt, fetched)
	}
}

// A collections failure must degrade to an empty missing set rather than fail
// the request: the rest of the payload (schedule, milestones, Xûr inventory) is
// still useful. Nothing but the Missing stamps should be lost.
//
// Emptiness is asserted through the engine rather than by checking that one
// known hash went unstamped. "Hash 100 is not marked missing" stays true for a
// fallback set that is wrong but non-empty; the engine reads the whole set and
// ranks nothing only when it is genuinely empty.
func TestGetWeekly_MissingItemErrorDegradesToEmptySet(t *testing.T) {
	eng := efficiency.NewEngine(
		fakeBucketSource{rows: []manifest.CollectibleWithItem{raidRow(100), raidRow(101)}},
		fakeVersioner{"v-test"},
	)
	eng.BuildIndex()

	reader := &fakeMissingItems{err: errors.New("bungie profile unavailable")}
	res := weeklyFixture{
		pub:     xurPub(xurItemEnriched{Hash: 100, Name: "Gjallarhorn"}),
		missing: reader,
		engine:  eng,
	}.get(t)

	if reader.calls != 1 {
		t.Errorf("missing-item reader calls = %d, want 1", reader.calls)
	}
	if res.Xur == nil || len(res.Xur.Items) != 1 {
		t.Fatal("a missing-items failure should not cost the Xûr inventory")
	}
	if res.Xur.Items[0].Missing {
		t.Error("item stamped Missing despite the missing set being unavailable")
	}
	// An empty missing set makes efficiency.Rank return nothing, so assembly
	// falls back to the legacy heuristic. Any non-empty fallback set would rank
	// the seeded raid bucket and surface eff-7 instead.
	if len(res.Recommended) == 0 || res.Recommended[0].ID != "r-milestones" {
		t.Errorf("Recommended[0] = %+v, want the r-milestones fallback — the degraded missing set is not empty",
			res.Recommended)
	}
	if res.ResetLabel == "" {
		t.Error("schedule lost to a missing-items failure")
	}
}
