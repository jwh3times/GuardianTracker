package collections

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/manifest"
)

// fakeVendors stands in for Weekly's live-availability capability. calls counts
// invocations so a test can prove a read never asked for availability at all,
// which is a different claim from asking and getting nothing back.
type fakeVendors struct {
	live  map[uint32]string
	calls int
}

func (f *fakeVendors) LiveVendorItemHashes(context.Context, int, string, string) map[uint32]string {
	f.calls++
	return f.live
}

// fakeParticipant stands in for one refresh participant, recording exactly
// which membership it was told about.
type fakeParticipant struct {
	invalidated []Membership
}

func (f *fakeParticipant) InvalidateCache(membershipType int, membershipID string) {
	f.invalidated = append(f.invalidated, Membership{MembershipType: membershipType, MembershipID: membershipID})
}

// serviceFixture is the two-category tree used across these tests: Weapons(10)
// → Hand Cannons(11) holds items 100 (owned) and 101 (missing); Armor(20) holds
// item 200 (missing).
func serviceFixture() (map[uint32]*manifest.PresentationNodeDef, []items.AcquisitionFacts) {
	nodes := map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{10, 20}, nil),
		10: node(10, "Weapons", []uint32{11}, nil),
		11: node(11, "Hand Cannons", nil, []uint32{1000, 1001}),
		20: node(20, "Armor", nil, []uint32{2000}),
	}
	// Ascending item-hash order, as the Items catalog publishes it.
	catalog := []items.AcquisitionFacts{
		weapon(100, "Fatebringer", 1000),
		weapon(101, "The Palindrome", 1001),
		armor(200, "Helm of Saint-14", 2000),
	}
	return nodes, catalog
}

// newFixtureService wires a complete service over a pre-cached analysis for one
// membership, so every test here works from ownership data without a Bungie
// call.
func newFixtureService(t *testing.T, vendors LiveAvailabilityReader) (*Service, *fakeParticipant, *fakeParticipant) {
	t.Helper()
	nodes, catalog := serviceFixture()
	m := newAnalysis(t, &fakeCatalog{facts: catalog}, &fakeNodes{nodes: nodes})
	collected := map[uint32]bool{1000: true} // only Fatebringer's collectible
	m.cached(3, "member-1", &analysis{
		catalog:   catalog,
		collected: collected,
		owned:     deriveOwnedItems(catalog, collected),
		tree:      buildTreeStructure(nodes, catalog),
		fetchedAt: time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC),
	})
	chars, recs := &fakeParticipant{}, &fakeParticipant{}
	return NewService(m, vendors, chars, recs), chars, recs
}

func fixtureRequest() MembershipRequest {
	return MembershipRequest{MembershipType: 3, MembershipID: "member-1", AccessToken: "token"}
}

// A summary is a different capability, not a trimmed full result: it counts the
// same tree and reports the same totals, and carries no item surface at all.
func TestGetSummary_CountsWithoutNamingItems(t *testing.T) {
	svc, _, _ := newFixtureService(t, &fakeVendors{})

	got, err := svc.GetSummary(context.Background(), fixtureRequest())
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}

	if len(got.Tree) != 2 || got.Tree[0].Name != "Armor" || got.Tree[1].Name != "Weapons" {
		t.Fatalf("top-level categories = %+v, want [Armor Weapons]", got.Tree)
	}
	weapons := findCounted(got.Tree, "Weapons")
	if weapons.Total != 2 || weapons.Collected != 1 {
		t.Errorf("Weapons counts = %d/%d, want 1/2", weapons.Collected, weapons.Total)
	}
	if got.Totals.Weapons.Total != 2 || got.Totals.Weapons.Collected != 1 {
		t.Errorf("totals.weapons = %d/%d, want 1/2", got.Totals.Weapons.Collected, got.Totals.Weapons.Total)
	}
	if got.Totals.Armor.Total != 1 || got.Totals.Armor.Collected != 0 {
		t.Errorf("totals.armor = %d/%d, want 0/1", got.Totals.Armor.Collected, got.Totals.Armor.Total)
	}
	if got.FetchedAt.IsZero() {
		t.Error("FetchedAt must be the ownership-profile fetch time")
	}

	// No node names its leaves: that is the whole difference from a Full.
	var named func(nodes []CollectionNode) bool
	named = func(nodes []CollectionNode) bool {
		for _, n := range nodes {
			if len(n.Items) > 0 || named(n.Children) {
				return true
			}
		}
		return false
	}
	if named(got.Tree) {
		t.Error("a summary tree must not carry leaf item hashes")
	}
}

// A summary must not reach a vendor at all — not call one and discard the
// answer. That is what keeps the Dashboard's read cheap.
func TestGetSummary_NeverAsksForAvailability(t *testing.T) {
	vendors := &fakeVendors{live: map[uint32]string{100: "Banshee-44"}}
	svc, _, _ := newFixtureService(t, vendors)

	if _, err := svc.GetSummary(context.Background(), fixtureRequest()); err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if vendors.calls != 0 {
		t.Errorf("live availability called %d times during a summary, want 0", vendors.calls)
	}
}

// A summary and a full result for the same membership describe the same
// collection: the counts and totals must agree, or the two capabilities have
// drifted into two different opinions of one thing.
func TestSummaryAndFullAgreeOnCounts(t *testing.T) {
	svc, _, _ := newFixtureService(t, &fakeVendors{})
	ctx := context.Background()

	summary, err := svc.GetSummary(ctx, fixtureRequest())
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	full, err := svc.GetFull(ctx, fixtureRequest())
	if err != nil {
		t.Fatalf("GetFull: %v", err)
	}

	if summary.Totals != full.Totals {
		t.Errorf("totals disagree: summary %+v, full %+v", summary.Totals, full.Totals)
	}
	if summary.FetchedAt != full.FetchedAt {
		t.Errorf("fetchedAt disagrees: summary %v, full %v", summary.FetchedAt, full.FetchedAt)
	}
	for _, name := range []string{"Armor", "Weapons"} {
		s, f := findCounted(summary.Tree, name), findCounted(full.Tree, name)
		if s.Total != f.Total || s.Collected != f.Collected {
			t.Errorf("%s counts disagree: summary %d/%d, full %d/%d", name, s.Collected, s.Total, f.Collected, f.Total)
		}
	}
}

// Every catalogued item appears exactly once, in ascending item-hash order,
// carrying the canonical facts, this membership's ownership, and nothing
// invented. The order is contract: the HTTP adapter relies on it.
func TestGetFull_CompleteOrderedItems(t *testing.T) {
	svc, _, _ := newFixtureService(t, &fakeVendors{})

	got, err := svc.GetFull(context.Background(), fixtureRequest())
	if err != nil {
		t.Fatalf("GetFull: %v", err)
	}

	want := []struct {
		hash      string
		name      string
		collected bool
	}{
		{"100", "Fatebringer", true},
		{"101", "The Palindrome", false},
		{"200", "Helm of Saint-14", false},
	}
	if len(got.Items) != len(want) {
		t.Fatalf("items = %d, want %d", len(got.Items), len(want))
	}
	for i, w := range want {
		item := got.Items[i]
		if item.Item.ItemHash != w.hash || item.Item.Name != w.name {
			t.Errorf("items[%d] = %s/%s, want %s/%s", i, item.Item.ItemHash, item.Item.Name, w.hash, w.name)
		}
		if item.Collected != w.collected {
			t.Errorf("items[%d] (%s) collected = %v, want %v", i, w.hash, item.Collected, w.collected)
		}
		if item.AvailableFrom != "" {
			t.Errorf("items[%d] (%s) availableFrom = %q, want empty with no vendor data", i, w.hash, item.AvailableFrom)
		}
	}

	// A full tree does name its leaves.
	if hc := findCounted(findCounted(got.Tree, "Weapons").Children, "Hand Cannons"); len(hc.Items) != 2 {
		t.Errorf("Hand Cannons items = %+v, want 2 leaf hashes", hc.Items)
	}
}

// Availability is stamped on the items the collection actually tracks. A vendor
// selling something the catalog does not carry has nowhere to land, so it is
// dropped rather than announced.
func TestGetFull_IntersectsAvailabilityWithTrackedItems(t *testing.T) {
	vendors := &fakeVendors{live: map[uint32]string{
		101: "Banshee-44",
		999: "Xûr", // not in the catalog
	}}
	svc, _, _ := newFixtureService(t, vendors)

	got, err := svc.GetFull(context.Background(), fixtureRequest())
	if err != nil {
		t.Fatalf("GetFull: %v", err)
	}

	stamped := map[string]string{}
	for _, item := range got.Items {
		if item.AvailableFrom != "" {
			stamped[item.Item.ItemHash] = item.AvailableFrom
		}
	}
	if len(stamped) != 1 || stamped["101"] != "Banshee-44" {
		t.Errorf("stamped availability = %v, want only item 101 from Banshee-44", stamped)
	}
	if vendors.calls != 1 {
		t.Errorf("live availability called %d times, want exactly 1", vendors.calls)
	}
}

// Availability is best effort. A vendor read that returns nothing — degraded,
// unauthorized, or genuinely empty — produces an empty overlay and must not
// fail an otherwise valid collection.
func TestGetFull_UnavailableVendorsDoNotFailTheResult(t *testing.T) {
	svc, _, _ := newFixtureService(t, &fakeVendors{live: nil})

	got, err := svc.GetFull(context.Background(), fixtureRequest())
	if err != nil {
		t.Fatalf("GetFull must survive an unavailable vendor read: %v", err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("items = %d, want the complete catalog of 3", len(got.Items))
	}
	for _, item := range got.Items {
		if item.AvailableFrom != "" {
			t.Errorf("item %s stamped %q from an unavailable vendor read", item.Item.ItemHash, item.AvailableFrom)
		}
	}
}

// The availability join belongs to one request. Writing it into the cached
// analysis would leak one request's vendor rotation into every later read of
// that membership, including summaries.
func TestGetFull_AvailabilityNeverPersistsIntoTheCachedAnalysis(t *testing.T) {
	vendors := &fakeVendors{live: map[uint32]string{101: "Banshee-44"}}
	svc, _, _ := newFixtureService(t, vendors)
	ctx := context.Background()

	if _, err := svc.GetFull(ctx, fixtureRequest()); err != nil {
		t.Fatalf("first GetFull: %v", err)
	}

	vendors.live = nil // the rotation moved on
	got, err := svc.GetFull(ctx, fixtureRequest())
	if err != nil {
		t.Fatalf("second GetFull: %v", err)
	}
	for _, item := range got.Items {
		if item.AvailableFrom != "" {
			t.Errorf("item %s still stamped %q from the previous request", item.Item.ItemHash, item.AvailableFrom)
		}
	}
}

// A failed core read is the answer. Asking a vendor afterwards would spend a
// Bungie call on a result that cannot be returned.
func TestGetFull_CoreErrorShortCircuitsAvailability(t *testing.T) {
	nodes, catalog := serviceFixture()
	broken := &fakeCatalog{facts: catalog}
	m := newAnalysis(t, broken, &fakeNodes{nodes: nodes})
	m.cached(3, "member-1", &analysis{
		catalog:   catalog,
		collected: map[uint32]bool{},
		owned:     map[uint32]bool{},
		tree:      buildTreeStructure(nodes, catalog),
		fetchedAt: time.Now(),
	})
	// Retire the cached analysis and break the rebuild it now needs, so the
	// core read fails on a path that does not require a live manifest.
	if err := m.OnVersionChanged("next-manifest"); err != nil {
		t.Fatalf("OnVersionChanged: %v", err)
	}
	broken.err = errors.New("catalog exploded")

	vendors := &fakeVendors{live: map[uint32]string{100: "Banshee-44"}}
	svc := NewService(m, vendors, &fakeParticipant{}, &fakeParticipant{})

	if _, err := svc.GetFull(context.Background(), fixtureRequest()); err == nil {
		t.Fatal("GetFull must fail when the core read fails")
	}
	if vendors.calls != 0 {
		t.Errorf("live availability called %d times after a core failure, want 0", vendors.calls)
	}
}

// A refresh reaches exactly the three owners of membership-scoped upstream
// data, each told the membership pair, and all of them before it returns.
func TestRefreshMembership_AdvancesEveryParticipant(t *testing.T) {
	svc, chars, recs := newFixtureService(t, &fakeVendors{})
	membership := Membership{MembershipType: 3, MembershipID: "member-1"}

	// Prove the collections half really was invalidated: a cached analysis is
	// present before the refresh and gone after it.
	if _, found := svc.analysis.cache.Get(analysisCacheKey(3, "member-1")); !found {
		t.Fatal("fixture must start with a cached analysis")
	}

	if err := svc.RefreshMembership(context.Background(), membership); err != nil {
		t.Fatalf("RefreshMembership: %v", err)
	}

	if _, found := svc.analysis.cache.Get(analysisCacheKey(3, "member-1")); found {
		t.Error("the collections analysis was not invalidated")
	}
	for name, p := range map[string]*fakeParticipant{"characters": chars, "records": recs} {
		if len(p.invalidated) != 1 {
			t.Fatalf("%s invalidated %d times, want exactly 1", name, len(p.invalidated))
		}
		if p.invalidated[0] != membership {
			t.Errorf("%s told about %+v, want %+v", name, p.invalidated[0], membership)
		}
	}
}

// Every dependency is required. A missing one is a composition error that must
// stop the process at startup, not surface as a nil-pointer panic on whichever
// request first needs it.
func TestNewService_RequiresEveryDependency(t *testing.T) {
	nodes, catalog := serviceFixture()
	analysis := newAnalysis(t, &fakeCatalog{facts: catalog}, &fakeNodes{nodes: nodes})

	cases := []struct {
		name       string
		analysis   *MembershipAnalysis
		live       LiveAvailabilityReader
		characters RefreshParticipant
		records    RefreshParticipant
	}{
		{"no analysis", nil, &fakeVendors{}, &fakeParticipant{}, &fakeParticipant{}},
		{"no live availability", analysis, nil, &fakeParticipant{}, &fakeParticipant{}},
		{"no characters", analysis, &fakeVendors{}, nil, &fakeParticipant{}},
		{"no records", analysis, &fakeVendors{}, &fakeParticipant{}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("NewService must panic on a missing required dependency")
				}
			}()
			NewService(tc.analysis, tc.live, tc.characters, tc.records)
		})
	}
}
