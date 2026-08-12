package weekly

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
)

// The rule these three tests pin was a prose comment at each site and asserted
// nowhere: an empty vendor or roster response is a transient answer, not a fact
// about the day, and caching one under a key that lives until the next daily
// reset serves every user an empty week until then.
//
// Each test proves it the only way that distinguishes "not cached" from "cached
// something harmless": change what the upstream reports, call again, and require
// the new answer. Asserting the cache slot is empty would pass just as happily
// for a slot holding the wrong thing.

// vendorsSelling builds an authenticated vendor response in which one
// allowlisted vendor is selling the given item hashes.
func vendorsSelling(vendorKey string, hashes ...uint32) *bungie.CharacterVendorsResponse {
	resp := &bungie.CharacterVendorsResponse{}
	sales := map[string]bungie.VendorSaleItem{}
	for i, h := range hashes {
		sales[string(rune('0'+i))] = bungie.VendorSaleItem{ItemHash: h}
	}
	resp.Response.Sales.Data = map[string]bungie.VendorSales{vendorKey: {SaleItems: sales}}
	return resp
}

const bansheeVendorKey = "672118013"

func TestGetLiveVendorItems_DoesNotCacheAnEmptyResult(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	c := cache.NewMemoryCache(time.Minute, 0)
	s := &Service{cache: c}

	// A real response in which no allowlisted vendor is selling anything.
	c.Set(characterVendorsCacheKey(3, "member-1", "char-1"), &bungie.CharacterVendorsResponse{}, time.Minute)
	if got := s.getLiveVendorItems(ctx, 3, "member-1", "char-1", "token", now); len(got) != 0 {
		t.Fatalf("first call = %+v, want empty", got)
	}

	// Banshee restocks. The empty answer must not still be standing in for it.
	c.Set(characterVendorsCacheKey(3, "member-1", "char-1"), vendorsSelling(bansheeVendorKey, 4242), time.Minute)
	got := s.getLiveVendorItems(ctx, 3, "member-1", "char-1", "token", now)
	if got[4242] != "Banshee-44" {
		t.Errorf("second call = %+v, want the restocked item — the empty result was cached until the next daily reset", got)
	}
}

func TestGetDailyVendorItems_DoesNotCacheAnEmptyResult(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	c := cache.NewMemoryCache(time.Minute, 0)
	// enrichDailyVendorItems resolves item names through the manifest, so the
	// service needs one to produce a non-empty result at all.
	mod := &bungie.InventoryItemDefinition{Hash: 4242, ItemType: bungie.ItemTypeMod}
	mod.DisplayProperties.Name = "Targeting Adjuster"
	s := &Service{cache: c, manifest: &fakeManifest{items: map[uint32]*bungie.InventoryItemDefinition{4242: mod}}}

	c.Set(characterVendorsCacheKey(3, "member-1", "char-1"), &bungie.CharacterVendorsResponse{}, time.Minute)
	if got := s.getDailyVendorItems(ctx, 3, "member-1", "char-1", "token", now); len(got) != 0 {
		t.Fatalf("first call = %+v, want empty", got)
	}

	c.Set(characterVendorsCacheKey(3, "member-1", "char-1"), vendorsSelling(bansheeVendorKey, 4242), time.Minute)
	got := s.getDailyVendorItems(ctx, 3, "member-1", "char-1", "token", now)
	if len(got) != 1 || got[0].ItemHash != 4242 {
		t.Errorf("second call = %+v, want the restocked mod — the empty result was cached until the next daily reset", got)
	}
}

// resolveCharacter's empty-roster rule matters for a different reason: a cached
// empty roster resolves every request to the fallback ("", -1) for its whole
// TTL, so a user whose roster came back empty once loses character scoping for
// five minutes.
//
// This one needs a real HTTP round trip rather than a seeded cache: the rule
// only applies when the fetch *succeeds* and returns nothing, which a nil client
// (an error) never reaches.
func TestResolveCharacter_DoesNotCacheAnEmptyRoster(t *testing.T) {
	ctx := context.Background()
	characters := `{}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"ErrorCode":1,"Response":{"characters":{"data":%s}}}`, characters)
	}))
	defer srv.Close()

	s := &Service{cache: cache.NewMemoryCache(time.Minute, 0), bungie: bungie.NewClient("k", srv.URL, 100, 100)}

	if id, class := s.resolveCharacter(ctx, 3, "member-1", "token", ""); id != "" || class != -1 {
		t.Fatalf("first call = (%q, %d), want the unresolved fallback", id, class)
	}

	// Bungie now reports the roster it should have all along.
	characters = `{"char-1":{"characterId":"char-1","classType":2,"dateLastPlayed":"2026-06-10T12:00:00Z"}}`
	id, class := s.resolveCharacter(ctx, 3, "member-1", "token", "")
	if id != "char-1" || class != 2 {
		t.Errorf("second call = (%q, %d), want char-1 — the empty roster was cached over it", id, class)
	}

	// And the real roster is cached: a third call does not refetch.
	characters = `{}`
	if id, _ := s.resolveCharacter(ctx, 3, "member-1", "token", ""); id != "char-1" {
		t.Errorf("third call = %q, want the cached char-1", id)
	}
}
