package weekly

import (
	"context"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
)

func TestExtractVendorItems(t *testing.T) {
	resp := &bungie.CharacterVendorsResponse{}
	resp.Response.Sales.Data = map[string]bungie.VendorSales{
		"672118013": {SaleItems: map[string]bungie.VendorSaleItem{ // Banshee-44 (allowlisted)
			"0": {ItemHash: 100},
			"1": {ItemHash: 200},
			"2": {ItemHash: 0}, // zero hash skipped
		}},
		"3603221665": {SaleItems: map[string]bungie.VendorSaleItem{ // Shaxx (allowlisted)
			"0": {ItemHash: 300},
		}},
		"999999999": {SaleItems: map[string]bungie.VendorSaleItem{ // not allowlisted
			"0": {ItemHash: 400},
		}},
	}

	got := extractVendorItems(resp, liveVendorAllowlist)

	if got[100] != "Banshee-44" || got[200] != "Banshee-44" {
		t.Errorf("Banshee items = %+v, want both labeled Banshee-44", got)
	}
	if got[300] != "Lord Shaxx" {
		t.Errorf("Shaxx item = %q, want Lord Shaxx", got[300])
	}
	if _, ok := got[400]; ok {
		t.Errorf("item from non-allowlisted vendor must be excluded")
	}
	if _, ok := got[0]; ok {
		t.Errorf("zero item hash must be skipped")
	}
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestExtractVendorItems_NilSafe(t *testing.T) {
	if got := extractVendorItems(nil, liveVendorAllowlist); len(got) != 0 {
		t.Errorf("nil response = %+v, want empty", got)
	}
	// Empty-but-non-nil response: the range loop runs zero times — also empty, no panic.
	empty := &bungie.CharacterVendorsResponse{}
	empty.Response.Sales.Data = map[string]bungie.VendorSales{}
	if got := extractVendorItems(empty, liveVendorAllowlist); len(got) != 0 {
		t.Errorf("empty response = %+v, want empty", got)
	}
}

func TestLiveVendorItemHashes_MergesXurAndVendors(t *testing.T) {
	friday := time.Date(2026, 6, 12, 18, 0, 0, 0, time.UTC) // Xûr present

	c := cache.NewMemoryCache(time.Minute, time.Minute)
	c.Set("weekly:public", &publicWeeklyCache{
		XurPresent: true,
		XurItems:   []xurItemEnriched{{Hash: 111}, {Hash: 222}},
	}, time.Minute)
	c.Set("live:vendoritems:3:member-1:char-1", map[uint32]string{333: "Banshee-44"}, time.Minute)
	s := &Service{cache: c}

	got := s.liveVendorItemHashesAtCharacter(context.Background(), 3, "member-1", "char-1", "token", friday)

	if got[111] != "Xûr" || got[222] != "Xûr" {
		t.Errorf("Xûr items = %+v, want both labeled Xûr", got)
	}
	if got[333] != "Banshee-44" {
		t.Errorf("vendor item = %q, want Banshee-44", got[333])
	}
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestLiveVendorItemHashes_XurWinsTie(t *testing.T) {
	friday := time.Date(2026, 6, 12, 18, 0, 0, 0, time.UTC) // Xûr present

	c := cache.NewMemoryCache(time.Minute, time.Minute)
	// Hash 555 is sold by both a vendor and Xûr — Xûr must win the label.
	c.Set("weekly:public", &publicWeeklyCache{
		XurPresent: true,
		XurItems:   []xurItemEnriched{{Hash: 555}},
	}, time.Minute)
	c.Set("live:vendoritems:3:member-1:char-1", map[uint32]string{555: "Banshee-44"}, time.Minute)
	s := &Service{cache: c}

	got := s.liveVendorItemHashesAtCharacter(context.Background(), 3, "member-1", "char-1", "token", friday)

	if got[555] != "Xûr" {
		t.Errorf("shared hash label = %q, want Xûr (Xûr wins ties)", got[555])
	}
}

func TestLiveVendorItemHashes_Degraded(t *testing.T) {
	wednesday := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC) // Xûr absent

	c := cache.NewMemoryCache(time.Minute, time.Minute)
	s := &Service{cache: c} // nil bungie client, empty caches

	// Empty token → vendor fetch skipped; Xûr absent on a Wednesday → empty.
	// Must not panic despite the nil bungie client.
	got := s.liveVendorItemHashesAt(context.Background(), 3, "member-1", "", wednesday)
	if len(got) != 0 {
		t.Errorf("degraded path = %+v, want empty", got)
	}
}

func TestLiveVendorItemsCacheIsolatedByCharacter(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	c := cache.NewMemoryCache(time.Minute, time.Minute)
	hunterResponse := &bungie.CharacterVendorsResponse{}
	warlockResponse := &bungie.CharacterVendorsResponse{}
	c.Set("vendors:character:3:member-1:char-hunter", hunterResponse, time.Minute)
	c.Set("vendors:character:3:member-1:char-warlock", warlockResponse, time.Minute)
	c.Set("live:vendoritems:3:member-1:char-hunter", map[uint32]string{101: "Banshee-44"}, time.Minute)
	c.Set("live:vendoritems:3:member-1:char-warlock", map[uint32]string{202: "Banshee-44"}, time.Minute)
	c.Set("daily:vendors:3:member-1:char-hunter", []dailyVendorItem{{ItemHash: 101}}, time.Minute)
	c.Set("daily:vendors:3:member-1:char-warlock", []dailyVendorItem{{ItemHash: 202}}, time.Minute)
	s := &Service{cache: c}

	if got := s.getCharacterVendors(context.Background(), 3, "member-1", "char-hunter", "token"); got != hunterResponse {
		t.Error("hunter 400/402 response crossed character cache boundary")
	}
	if got := s.getCharacterVendors(context.Background(), 3, "member-1", "char-warlock", "token"); got != warlockResponse {
		t.Error("warlock 400/402 response crossed character cache boundary")
	}
	hunter := s.getLiveVendorItems(context.Background(), 3, "member-1", "char-hunter", "token", now)
	warlock := s.getLiveVendorItems(context.Background(), 3, "member-1", "char-warlock", "token", now)
	if _, ok := hunter[101]; !ok || len(hunter) != 1 {
		t.Errorf("hunter cache = %+v, want only item 101", hunter)
	}
	if _, ok := warlock[202]; !ok || len(warlock) != 1 {
		t.Errorf("warlock cache = %+v, want only item 202", warlock)
	}
	hunterDaily := s.getDailyVendorItems(context.Background(), 3, "member-1", "char-hunter", "token", now)
	warlockDaily := s.getDailyVendorItems(context.Background(), 3, "member-1", "char-warlock", "token", now)
	if len(hunterDaily) != 1 || hunterDaily[0].ItemHash != 101 {
		t.Errorf("hunter daily cache = %+v, want only item 101", hunterDaily)
	}
	if len(warlockDaily) != 1 || warlockDaily[0].ItemHash != 202 {
		t.Errorf("warlock daily cache = %+v, want only item 202", warlockDaily)
	}
}
