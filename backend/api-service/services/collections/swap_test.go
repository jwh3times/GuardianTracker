package collections

import (
	"context"
	"testing"
	"time"

	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/manifest"
)

func fixtureCatalog() []items.AcquisitionFacts {
	return []items.AcquisitionFacts{
		weapon(100, "Fatebringer", 1000),
		weapon(101, "The Palindrome", 1001),
	}
}

func fixtureNodes() map[uint32]*manifest.PresentationNodeDef {
	return map[uint32]*manifest.PresentationNodeDef{
		1:  node(1, "Items", []uint32{10}, nil),
		10: node(10, "Weapons", nil, []uint32{1000, 1001}),
	}
}

// swapped advances the analysis past one manifest generation, the way the
// manifest service's observer notification does.
func swapped(t *testing.T, m *MembershipAnalysis, version string) {
	t.Helper()
	if err := m.OnVersionChanged(version); err != nil {
		t.Fatalf("OnVersionChanged(%q): %v", version, err)
	}
}

// The heart of the design: a manifest swap must refresh the manifest-derived
// half of a cached analysis while KEEPING the profile data, because that half
// cost a rate-limited Bungie fetch. Evicting instead would mean a refetch storm
// across every active user on every hourly swap.
func TestRefreshManifestParts_RebuildsManifestHalfKeepsProfileHalf(t *testing.T) {
	catalog := &fakeCatalog{facts: fixtureCatalog()}
	m := newAnalysis(t, catalog, &fakeNodes{nodes: fixtureNodes()})

	collected := map[uint32]bool{1000: true}
	stale := m.cached(3, "member-1", &analysis{
		catalog:   nil, // deliberately empty: proves a real rebuild happened
		collected: collected,
		owned:     map[uint32]bool{},
		tree:      nil,
		fetchedAt: time.Unix(1000, 0).UTC(),
	})
	swapped(t, m, "v2")

	key := analysisCacheKey(3, "member-1")
	got, err := m.refreshManifestParts(context.Background(), key, stale, m.refresh.Begin(3, "member-1"))
	if err != nil {
		t.Fatalf("refreshManifestParts: %v", err)
	}

	if got == stale {
		t.Fatal("a stale analysis was returned unchanged")
	}
	if catalog.reads != 1 {
		t.Errorf("catalog reads = %d, want 1", catalog.reads)
	}
	if len(got.catalog) != 2 {
		t.Errorf("catalog = %d entries, want 2 (rebuilt from Items)", len(got.catalog))
	}
	if got.tree == nil {
		t.Error("tree was not rebuilt")
	}
	if !got.builtUnder.Current() {
		t.Error("the rebuilt analysis is not stamped with the current generation")
	}
	// The expensive half survives, re-derived rather than refetched.
	if len(got.collected) != 1 || !got.collected[1000] {
		t.Errorf("profile data was lost: %+v", got.collected)
	}
	if !got.owned[100] {
		t.Error("owned was not re-derived from the retained profile data")
	}
	if !got.fetchedAt.Equal(stale.fetchedAt) {
		t.Error("fetchedAt changed; no Bungie fetch happened, so it must not move")
	}
	// The replacement is what the next request finds.
	if cached, ok := m.cache.Get(key); !ok || cached.(*analysis) != got {
		t.Error("the rebuilt analysis was not cached in place of the stale one")
	}

	// The original must be untouched — concurrent readers hold that pointer.
	if len(stale.catalog) != 0 || stale.tree != nil {
		t.Error("refreshManifestParts mutated the analysis other requests are sharing")
	}
}

// The common path: an analysis built under the current generation allocates
// nothing and reads nothing.
func TestRefreshManifestParts_CurrentGenerationIsANoOp(t *testing.T) {
	catalog := &fakeCatalog{facts: fixtureCatalog()}
	m := newAnalysis(t, catalog, &fakeNodes{nodes: fixtureNodes()})
	fresh := m.cached(3, "member-1", &analysis{collected: map[uint32]bool{}})

	got, err := m.refreshManifestParts(context.Background(), analysisCacheKey(3, "member-1"), fresh, m.refresh.Begin(3, "member-1"))
	if err != nil {
		t.Fatalf("refreshManifestParts: %v", err)
	}
	if got != fresh {
		t.Error("a current analysis was needlessly rebuilt")
	}
	if catalog.reads != 0 {
		t.Errorf("catalog reads = %d, want 0", catalog.reads)
	}
}

// An analysis carrying no attempt at all — a cache entry from before this
// field existed, or a zero value — must rebuild rather than pass as fresh.
func TestRefreshManifestParts_UnstampedAnalysisRebuilds(t *testing.T) {
	catalog := &fakeCatalog{facts: fixtureCatalog()}
	m := newAnalysis(t, catalog, &fakeNodes{nodes: fixtureNodes()})

	got, err := m.refreshManifestParts(context.Background(), analysisCacheKey(3, "member-1"),
		&analysis{collected: map[uint32]bool{}}, m.refresh.Begin(3, "member-1"))
	if err != nil {
		t.Fatalf("refreshManifestParts: %v", err)
	}
	if len(got.catalog) != 2 || catalog.reads != 1 {
		t.Errorf("catalog = %d entries after %d reads, want a rebuild", len(got.catalog), catalog.reads)
	}
}

// Mid-swap the catalog reports the manifest as not ready. Serving the previous
// manifest's labels for one more request beats failing a request whose data we
// already have.
func TestRefreshManifestParts_ManifestNotReadyServesStale(t *testing.T) {
	m := newAnalysis(t, &fakeCatalog{err: manifest.ErrNotReady}, &fakeNodes{nodes: fixtureNodes()})
	stale := m.cached(3, "member-1", &analysis{collected: map[uint32]bool{1000: true}})
	swapped(t, m, "v2")

	got, err := m.refreshManifestParts(context.Background(), analysisCacheKey(3, "member-1"), stale, m.refresh.Begin(3, "member-1"))
	if err != nil {
		t.Fatalf("a mid-swap refresh must not fail the request: %v", err)
	}
	if got != stale {
		t.Error("expected the previous analysis to be served unchanged")
	}
}

// ADR 0014: work that began under one generation still answers the request that
// started it, but must not be left behind for anyone else. Here the swap lands
// while the catalog is being read.
func TestRefreshManifestParts_RetiredGenerationIsNotCached(t *testing.T) {
	catalog := &fakeCatalog{facts: fixtureCatalog()}
	m := newAnalysis(t, catalog, &fakeNodes{nodes: fixtureNodes()})
	stale := m.cached(3, "member-1", &analysis{collected: map[uint32]bool{1000: true}})
	swapped(t, m, "v2")
	catalog.onRead = func() { swapped(t, m, "v3") } // a second swap, mid-rebuild

	key := analysisCacheKey(3, "member-1")
	got, err := m.refreshManifestParts(context.Background(), key, stale, m.refresh.Begin(3, "member-1"))
	if err != nil {
		t.Fatalf("refreshManifestParts: %v", err)
	}
	if len(got.catalog) != 2 {
		t.Fatalf("the caller must still receive the coherent result it loaded: %+v", got)
	}
	cached, ok := m.cache.Get(key)
	if !ok {
		t.Fatal("the previous analysis was evicted")
	}
	if cached.(*analysis) != stale {
		t.Error("an analysis built under a retired generation was published")
	}
	if m.treeStruct != nil {
		t.Error("a tree built under a retired generation was published")
	}
}

// OnVersionChanged drops the shared tree. It deliberately does NOT evict the
// per-membership entries — refreshManifestParts repairs those lazily instead.
func TestOnVersionChanged_DropsSharedTreeNotMembershipEntries(t *testing.T) {
	m := newAnalysis(t, &fakeCatalog{facts: fixtureCatalog()}, &fakeNodes{nodes: fixtureNodes()})
	m.storeTree(&TreeStructure{})
	m.cached(3, "member-1", &analysis{})

	swapped(t, m, "v2")

	if m.treeStruct != nil {
		t.Error("shared tree survived the swap")
	}
	if _, ok := m.cache.Get(analysisCacheKey(3, "member-1")); !ok {
		t.Error("per-membership analysis was evicted; that discards a rate-limited Bungie fetch")
	}
}

// The shared tree is built once and reused until a swap retires it, so a second
// membership does not re-read the presentation nodes.
func TestTreeStructure_SharedUntilTheNextSwap(t *testing.T) {
	catalog := fixtureCatalog()
	nodes := &fakeNodes{nodes: fixtureNodes()}
	m := newAnalysis(t, &fakeCatalog{facts: catalog}, nodes)

	first, err := m.treeStructure(m.publication.Begin(), catalog)
	if err != nil {
		t.Fatalf("treeStructure: %v", err)
	}
	second, err := m.treeStructure(m.publication.Begin(), catalog)
	if err != nil {
		t.Fatalf("treeStructure: %v", err)
	}
	if first != second || nodes.reads != 1 {
		t.Errorf("tree rebuilt per call (reads = %d); it is user-independent", nodes.reads)
	}

	swapped(t, m, "v2")
	third, err := m.treeStructure(m.publication.Begin(), catalog)
	if err != nil {
		t.Fatalf("treeStructure: %v", err)
	}
	if third == first || nodes.reads != 2 {
		t.Errorf("tree survived a swap (reads = %d)", nodes.reads)
	}
}
