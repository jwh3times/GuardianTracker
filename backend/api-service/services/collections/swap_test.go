package collections

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

// fakeManifestRepo counts reads so a test can prove a refresh actually hit the
// manifest rather than silently reusing stale state.
type fakeManifestRepo struct {
	cols  []manifest.CollectibleWithItem
	nodes map[uint32]*manifest.PresentationNodeDef
	err   error
	reads int
}

func (f *fakeManifestRepo) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	f.reads++
	if f.err != nil {
		return nil, f.err
	}
	return f.cols, nil
}

func (f *fakeManifestRepo) GetAllPresentationNodes() (map[uint32]*manifest.PresentationNodeDef, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.nodes, nil
}

// serviceAtVersion builds a Service whose manifest service reports `version`.
// NewManifestService reads the sibling version file at construction, which is
// the only way to pin a version without running a download.
func serviceAtVersion(t *testing.T, version string, repo ManifestRepo) *Service {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	if err := os.WriteFile(filepath.Join(dir, "manifest_version.txt"), []byte(version), 0644); err != nil {
		t.Fatal(err)
	}
	ms := bungie.NewManifestService(bungie.NewClient("k", "http://unused", 100, 100), dbPath, time.Hour)
	if ms.Version() != version {
		t.Fatalf("fixture manifest version = %q, want %q", ms.Version(), version)
	}
	return &Service{
		manifestService: ms,
		manifest:        repo,
		cache:           cache.NewMemoryCache(time.Minute, 0),
		cacheTTL:        time.Minute,
	}
}

func fixtureRepo() *fakeManifestRepo {
	return &fakeManifestRepo{
		cols: []manifest.CollectibleWithItem{
			col(1000, 100, "Fatebringer"),
			col(1001, 101, "The Palindrome"),
		},
		nodes: map[uint32]*manifest.PresentationNodeDef{
			1:  node(1, "Items", []uint32{10}, nil),
			10: node(10, "Weapons", nil, []uint32{1000, 1001}),
		},
	}
}

// The heart of the design: a manifest swap must refresh the manifest-derived
// half of a cached analysis while KEEPING the profile data, because that half
// cost a rate-limited Bungie fetch. Evicting instead would mean a refetch storm
// across every active user on every hourly swap.
func TestRefreshManifestParts_RebuildsManifestHalfKeepsProfileHalf(t *testing.T) {
	repo := fixtureRepo()
	s := serviceAtVersion(t, "v2", repo)

	collected := map[uint32]bool{1000: true}
	stale := &analysis{
		collectibles:    nil, // deliberately empty: proves a real rebuild happened
		collected:       collected,
		owned:           map[uint32]bool{},
		tree:            nil,
		fetchedAt:       time.Unix(1000, 0).UTC(),
		manifestVersion: "v1",
	}

	got, err := s.refreshManifestParts(stale)
	if err != nil {
		t.Fatalf("refreshManifestParts: %v", err)
	}

	if got == stale {
		t.Fatal("a stale analysis was returned unchanged")
	}
	if repo.reads != 1 {
		t.Errorf("manifest reads = %d, want 1", repo.reads)
	}
	if len(got.collectibles) != 2 {
		t.Errorf("collectibles = %d, want 2 (rebuilt from the manifest)", len(got.collectibles))
	}
	if got.tree == nil {
		t.Error("tree was not rebuilt")
	}
	if got.manifestVersion != "v2" {
		t.Errorf("stamp = %q, want v2", got.manifestVersion)
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

	// The original must be untouched — concurrent readers hold that pointer.
	if stale.manifestVersion != "v1" || len(stale.collectibles) != 0 {
		t.Error("refreshManifestParts mutated the analysis other requests are sharing")
	}
}

// The common path: a matching stamp allocates nothing and reads nothing.
func TestRefreshManifestParts_CurrentStampIsANoOp(t *testing.T) {
	repo := fixtureRepo()
	s := serviceAtVersion(t, "v2", repo)

	fresh := &analysis{collected: map[uint32]bool{}, manifestVersion: "v2"}
	got, err := s.refreshManifestParts(fresh)
	if err != nil {
		t.Fatalf("refreshManifestParts: %v", err)
	}
	if got != fresh {
		t.Error("a current analysis was needlessly rebuilt")
	}
	if repo.reads != 0 {
		t.Errorf("manifest reads = %d, want 0", repo.reads)
	}
}

// Mid-swap the provider returns ErrNotReady. Serving the previous manifest's
// labels for one more request beats failing a request whose data we already have.
func TestRefreshManifestParts_ManifestNotReadyServesStale(t *testing.T) {
	repo := fixtureRepo()
	repo.err = manifest.ErrNotReady
	s := serviceAtVersion(t, "v2", repo)

	stale := &analysis{collected: map[uint32]bool{1000: true}, manifestVersion: "v1"}
	got, err := s.refreshManifestParts(stale)
	if err != nil {
		t.Fatalf("a mid-swap refresh must not fail the request: %v", err)
	}
	if got != stale {
		t.Error("expected the previous analysis to be served unchanged")
	}
}

// With no version knowable there is nothing to compare against, so the check
// must disable itself rather than rebuild on every single read.
func TestRefreshManifestParts_NoVersionSourceIsANoOp(t *testing.T) {
	repo := fixtureRepo()
	s := &Service{manifest: repo, cache: cache.NewMemoryCache(time.Minute, 0)}

	a := &analysis{collected: map[uint32]bool{}, manifestVersion: "v1"}
	got, err := s.refreshManifestParts(a)
	if err != nil {
		t.Fatalf("refreshManifestParts: %v", err)
	}
	if got != a || repo.reads != 0 {
		t.Error("refresh ran without a version to compare against")
	}
}

// OnVersionChanged drops the shared tree. It deliberately does NOT evict the
// per-user entries — refreshManifestParts repairs those lazily instead.
func TestOnVersionChanged_DropsSharedTreeNotUserEntries(t *testing.T) {
	repo := fixtureRepo()
	s := serviceAtVersion(t, "v2", repo)
	s.treeStruct = &TreeStructure{}
	s.cache.Set("collections:3:member-1", &analysis{manifestVersion: "v1"}, time.Minute)

	if err := s.OnVersionChanged("v2"); err != nil {
		t.Fatalf("OnVersionChanged: %v", err)
	}

	if s.treeStruct != nil {
		t.Error("shared tree survived the swap")
	}
	if _, ok := s.cache.Get("collections:3:member-1"); !ok {
		t.Error("per-user analysis was evicted; that discards a rate-limited Bungie fetch")
	}
}
