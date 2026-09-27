package collections

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/manifest"
)

type catalogFunc func(context.Context) ([]items.AcquisitionFacts, error)

func (f catalogFunc) Catalog(ctx context.Context) ([]items.AcquisitionFacts, error) { return f(ctx) }

type nodesFunc func() (map[uint32]*manifest.PresentationNodeDef, error)

func (f nodesFunc) GetAllPresentationNodes() (map[uint32]*manifest.PresentationNodeDef, error) {
	return f()
}

type pendingProfile struct {
	reply    chan string
	canceled <-chan struct{}
}

// The real HTTP client is used, but every upstream request stays in httptest.
// A ready placeholder Manifest keeps cold analysis focused on profile loading;
// Catalog and nodes are explicit fixtures, not reads from that placeholder.
func coldAnalysis(t *testing.T) (*MembershipAnalysis, <-chan pendingProfile, *atomic.Int32) {
	t.Helper()
	requests := make(chan pendingProfile, 16)
	hits := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/Manifest/") {
			fmt.Fprint(w, `{"ErrorCode":1,"Response":{"version":"v1"}}`)
			return
		}
		hits.Add(1)
		p := pendingProfile{reply: make(chan string, 1), canceled: r.Context().Done()}
		requests <- p
		select {
		case body := <-p.reply:
			fmt.Fprint(w, body)
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
			http.Error(w, "fixture timed out", http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	if err := os.WriteFile(dbPath, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest_version.txt"), []byte("v1"), 0600); err != nil {
		t.Fatal(err)
	}
	client := bungie.NewClient("fixture-key", srv.URL, 1000, 1000)
	ms := bungie.NewManifestService(client, dbPath, time.Hour)
	c := cache.NewMemoryCache(time.Minute, 0)
	t.Cleanup(c.Close)
	m := NewMembershipAnalysis(client, ms,
		catalogFunc(func(context.Context) ([]items.AcquisitionFacts, error) { return fixtureCatalog(), nil }),
		nodesFunc(func() (map[uint32]*manifest.PresentationNodeDef, error) { return fixtureNodes(), nil }), c, time.Minute)
	return m, requests, hits
}

func profileBody(collectible uint32) string {
	return fmt.Sprintf(`{"ErrorCode":1,"Response":{"profileCollectibles":{"privacy":1,"data":{"collectibles":{"%d":{"state":0}}}}}}`, collectible)
}

type analysisResult struct {
	a   *analysis
	err error
}

func startAnalysis(ctx context.Context, m *MembershipAnalysis, platform int, member, token string) <-chan analysisResult {
	result := make(chan analysisResult, 1)
	go func() { a, err := m.getAnalysis(ctx, platform, member, token); result <- analysisResult{a, err} }()
	return result
}
func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for fixture event")
		var zero T
		return zero
	}
}
func waitFlight(t *testing.T, m *MembershipAnalysis, count int) *analysisFlight {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		m.flightsMu.Lock()
		var got *analysisFlight
		if len(m.flights) == 1 {
			for _, f := range m.flights {
				if f.waiters == count {
					got = f
				}
			}
		}
		m.flightsMu.Unlock()
		if got != nil {
			return got
		}
		select {
		case <-deadline:
			t.Fatal("waiters did not join the expected flight")
			return nil
		default:
			runtime.Gosched()
		}
	}
}
func assertNoFlights(t *testing.T, m *MembershipAnalysis) {
	t.Helper()
	m.flightsMu.Lock()
	defer m.flightsMu.Unlock()
	if len(m.flights) != 0 {
		t.Fatalf("%d completed/abandoned flights remain", len(m.flights))
	}
}

func TestAnalysisFlight_SummaryMissingAndCollectedStateShareColdRead(t *testing.T) {
	m, requests, hits := coldAnalysis(t)
	svc := NewService(m, &fakeVendors{}, &fakeParticipant{}, &fakeParticipant{})
	summary := make(chan Summary, 1)
	errs := make(chan error, 3)
	go func() { v, e := svc.GetSummary(t.Context(), fixtureRequest()); summary <- v; errs <- e }()
	p := receive(t, requests)
	missing := make(chan map[uint32]struct{}, 1)
	go func() { v, e := m.GetMissingItemHashes(t.Context(), 3, "member-1", "token"); missing <- v; errs <- e }()
	owned := make(chan map[uint32]bool, 1)
	go func() { v, _, _, e := m.CollectedState(t.Context(), 3, "member-1", "token"); owned <- v; errs <- e }()
	waitFlight(t, m, 3)
	p.reply <- profileBody(1000)
	for range 3 {
		if err := receive(t, errs); err != nil {
			t.Fatal(err)
		}
	}
	if got := receive(t, summary); got.Totals.Weapons.Collected != 1 || got.Totals.Weapons.Total != 2 {
		t.Fatalf("summary=%+v", got.Totals)
	}
	if got := receive(t, missing); len(got) != 1 {
		t.Fatalf("missing=%v", got)
	} else if _, ok := got[101]; !ok {
		t.Fatalf("missing=%v", got)
	}
	if got := receive(t, owned); !got[100] || got[101] {
		t.Fatalf("owned=%v", got)
	}
	if _, err := svc.GetSummary(t.Context(), fixtureRequest()); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("profile requests=%d, want 1", hits.Load())
	}
	assertNoFlights(t, m)
}

func TestAnalysisFlight_CallerCancellationIsIndependent(t *testing.T) {
	for _, cancelLeader := range []bool{true, false} {
		t.Run(fmt.Sprintf("leader=%v", cancelLeader), func(t *testing.T) {
			m, requests, hits := coldAnalysis(t)
			leaderCtx, cancelLeaderCtx := context.WithCancel(t.Context())
			defer cancelLeaderCtx()
			followerCtx, cancelFollower := context.WithCancel(t.Context())
			defer cancelFollower()
			leader := startAnalysis(leaderCtx, m, 3, "member-1", "token")
			p := receive(t, requests)
			follower := startAnalysis(followerCtx, m, 3, "member-1", "token")
			waitFlight(t, m, 2)
			canceled, remaining := leader, follower
			if cancelLeader {
				cancelLeaderCtx()
			} else {
				cancelFollower()
				canceled, remaining = follower, leader
			}
			if got := receive(t, canceled); !errors.Is(got.err, context.Canceled) {
				t.Fatalf("canceled error=%v", got.err)
			}
			select {
			case <-p.canceled:
				t.Fatal("one caller canceled the shared HTTP request")
			default:
			}
			p.reply <- profileBody(1000)
			if got := receive(t, remaining); got.err != nil || !got.a.owned[100] {
				t.Fatalf("remaining result=%+v", got)
			}
			if hits.Load() != 1 {
				t.Fatalf("profile requests=%d", hits.Load())
			}
			assertNoFlights(t, m)
		})
	}
}

func TestAnalysisFlight_LastWaiterCancelsColdHTTPRead(t *testing.T) {
	m, requests, _ := coldAnalysis(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	first := startAnalysis(ctx, m, 3, "member-1", "token")
	p := receive(t, requests)
	second := startAnalysis(ctx, m, 3, "member-1", "token")
	waitFlight(t, m, 2)
	cancel()
	for _, ch := range []<-chan analysisResult{first, second} {
		if got := receive(t, ch); !errors.Is(got.err, context.Canceled) {
			t.Fatalf("error=%v", got.err)
		}
	}
	receive(t, p.canceled)
	assertNoFlights(t, m)
	if _, ok := m.cache.Get(analysisCacheKey(3, "member-1")); ok {
		t.Fatal("canceled HTTP read populated analysis cache")
	}
	replacement := startAnalysis(t.Context(), m, 3, "member-1", "token")
	receive(t, requests).reply <- profileBody(1001)
	if got := receive(t, replacement); got.err != nil || !got.a.owned[101] {
		t.Fatalf("replacement=%+v", got)
	}
}

func TestAnalysisFlight_GenerationsSeparateNewCallers(t *testing.T) {
	for _, boundary := range []string{"membership refresh", "manifest swap"} {
		t.Run(boundary, func(t *testing.T) {
			m, requests, hits := coldAnalysis(t)
			old := startAnalysis(t.Context(), m, 3, "member-1", "token")
			first := receive(t, requests)
			if boundary == "membership refresh" {
				m.InvalidateCache(3, "member-1")
			} else {
				swapped(t, m, "v2")
			}
			fresh := startAnalysis(t.Context(), m, 3, "member-1", "token")
			second := receive(t, requests)
			second.reply <- profileBody(1001)
			current := receive(t, fresh)
			if current.err != nil || !current.a.owned[101] {
				t.Fatalf("new result=%+v", current)
			}
			first.reply <- profileBody(1000)
			previous := receive(t, old)
			if previous.err != nil || !previous.a.owned[100] {
				t.Fatalf("old initiator result=%+v", previous)
			}
			cached, found := m.cache.Get(analysisCacheKey(3, "member-1"))
			if !found || cached != current.a {
				t.Fatal("old work replaced the post-boundary analysis")
			}
			if hits.Load() != 2 {
				t.Fatalf("profile requests=%d", hits.Load())
			}
		})
	}
}

func TestAnalysisFlight_IdentityAndCredentialsSeparateWork(t *testing.T) {
	for _, boundary := range []string{"membership", "platform", "credential"} {
		t.Run(boundary, func(t *testing.T) {
			m, requests, hits := coldAnalysis(t)
			old := startAnalysis(t.Context(), m, 3, "member-1", "old-token")
			first := receive(t, requests)
			platform, member, token := 3, "member-1", "old-token"
			switch boundary {
			case "membership":
				member = "member-2"
			case "platform":
				platform = 2
			case "credential":
				token = "replacement-token"
			}
			fresh := startAnalysis(t.Context(), m, platform, member, token)
			second := receive(t, requests)
			second.reply <- profileBody(1001)
			if got := receive(t, fresh); got.err != nil || !got.a.owned[101] {
				t.Fatalf("new result=%+v", got)
			}
			first.reply <- `{"ErrorCode":99,"Message":"fixture old authorization failed"}`
			if got := receive(t, old); got.err == nil {
				t.Fatal("expected old authorization failure")
			}
			if hits.Load() != 2 {
				t.Fatalf("profile requests=%d", hits.Load())
			}
			cached, ok := m.cache.Get(analysisCacheKey(platform, member))
			if !ok || !cached.(*analysis).owned[101] {
				t.Fatal("failure displaced the successful isolated result")
			}
		})
	}
}

func TestAnalysisFlight_FailureIsSharedButNotCached(t *testing.T) {
	m, requests, hits := coldAnalysis(t)
	first := startAnalysis(t.Context(), m, 3, "member-1", "token")
	p := receive(t, requests)
	second := startAnalysis(t.Context(), m, 3, "member-1", "token")
	waitFlight(t, m, 2)
	p.reply <- `{"ErrorCode":99,"Message":"fixture failure"}`
	for _, ch := range []<-chan analysisResult{first, second} {
		if got := receive(t, ch); got.err == nil {
			t.Fatal("expected shared failure")
		}
	}
	assertNoFlights(t, m)
	if _, ok := m.cache.Get(analysisCacheKey(3, "member-1")); ok {
		t.Fatal("failed analysis was cached")
	}
	retry := startAnalysis(t.Context(), m, 3, "member-1", "token")
	receive(t, requests).reply <- profileBody(1000)
	if got := receive(t, retry); got.err != nil {
		t.Fatal(got.err)
	}
	if hits.Load() != 2 {
		t.Fatalf("profile requests=%d", hits.Load())
	}
}

// The node reader deliberately ignores context. An abandoned worker cannot
// publish a tree or analysis even if its dependency completes successfully later.
func TestAnalysisFlight_AbandonmentFencesLateWorkAndAllowsReplacement(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%v", deadline), func(t *testing.T) {
			entered := make(chan chan struct{}, 4)
			var releasesMu sync.Mutex
			var releases []chan struct{}
			nodes := nodesFunc(func() (map[uint32]*manifest.PresentationNodeDef, error) {
				release := make(chan struct{})
				releasesMu.Lock()
				releases = append(releases, release)
				releasesMu.Unlock()
				entered <- release
				<-release
				return fixtureNodes(), nil
			})
			t.Cleanup(func() {
				releasesMu.Lock()
				defer releasesMu.Unlock()
				for _, ch := range releases {
					select {
					case <-ch:
					default:
						close(ch)
					}
				}
			})
			m := newAnalysis(t, catalogFunc(func(context.Context) ([]items.AcquisitionFacts, error) { return fixtureCatalog(), nil }), nodes)
			stale := cachedAnalysis(t, m, 3, "member-1")
			swapped(t, m, "v2")
			if deadline {
				m.loadTimeout = 100 * time.Millisecond
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			first := startAnalysis(ctx, m, 3, "member-1", "token")
			oldRelease := receive(t, entered)
			second := startAnalysis(ctx, m, 3, "member-1", "token")
			oldFlight := waitFlight(t, m, 2)
			want := context.Canceled
			if deadline {
				want = context.DeadlineExceeded
			} else {
				cancel()
			}
			for _, ch := range []<-chan analysisResult{first, second} {
				if got := receive(t, ch); !errors.Is(got.err, want) {
					t.Fatalf("error=%v want=%v", got.err, want)
				}
			}
			assertNoFlights(t, m)
			m.loadTimeout = analysisLoadTimeout
			replacement := startAnalysis(t.Context(), m, 3, "member-1", "token")
			newRelease := receive(t, entered)
			newFlight := waitFlight(t, m, 1)
			if oldFlight == newFlight {
				t.Fatal("replacement joined abandoned work")
			}
			close(oldRelease)
			receive(t, oldFlight.done)
			if got, _ := m.cache.Get(analysisCacheKey(3, "member-1")); got != stale {
				t.Fatal("late abandoned analysis was cached")
			}
			m.treeMu.RLock()
			tree := m.treeStruct
			m.treeMu.RUnlock()
			if tree != nil {
				t.Fatal("late abandoned tree was published")
			}
			if waitFlight(t, m, 1) != newFlight {
				t.Fatal("old completion removed the replacement")
			}
			close(newRelease)
			if got := receive(t, replacement); got.err != nil || got.a == stale {
				t.Fatalf("replacement=%+v", got)
			}
			assertNoFlights(t, m)
		})
	}
}

func TestAnalysisFlight_StaleManifestRebuildIsSharedAndRetainsContext(t *testing.T) {
	type contextKey struct{}
	entered, release := make(chan context.Context, 1), make(chan struct{})
	var reads atomic.Int32
	catalog := catalogFunc(func(ctx context.Context) ([]items.AcquisitionFacts, error) {
		reads.Add(1)
		entered <- ctx
		<-release
		return fixtureCatalog(), nil
	})
	m := newAnalysis(t, catalog, nodesFunc(func() (map[uint32]*manifest.PresentationNodeDef, error) { return fixtureNodes(), nil }))
	stale := cachedAnalysis(t, m, 3, "member-1")
	swapped(t, m, "v2")
	requestCtx, requestCancel := context.WithTimeout(t.Context(), time.Minute)
	defer requestCancel()
	ctx, cancel := context.WithCancel(context.WithValue(requestCtx, contextKey{}, "request-value"))
	defer cancel()
	first := startAnalysis(ctx, m, 3, "member-1", "token")
	shared := receive(t, entered)
	second := startAnalysis(t.Context(), m, 3, "member-1", "token")
	waitFlight(t, m, 2)
	if shared.Value(contextKey{}) != "request-value" {
		t.Fatal("shared context lost request values")
	}
	if deadline, ok := shared.Deadline(); !ok || time.Until(deadline) > analysisLoadTimeout || time.Until(deadline) < 2*time.Minute {
		t.Fatal("shared context must use its own bounded deadline, not the leader's shorter deadline")
	}
	cancel()
	if got := receive(t, first); !errors.Is(got.err, context.Canceled) {
		t.Fatal(got.err)
	}
	if shared.Err() != nil {
		t.Fatal("leader canceled remaining shared work")
	}
	close(release)
	got := receive(t, second)
	if got.err != nil || got.a == stale || !got.a.fetchedAt.Equal(stale.fetchedAt) {
		t.Fatalf("rebuilt=%+v", got)
	}
	if reads.Load() != 1 {
		t.Fatalf("catalog reads=%d", reads.Load())
	}
}

type firstMissCache struct {
	cache.Cache
	first atomic.Bool
	sets  atomic.Int32
}

func (c *firstMissCache) Get(key string) (any, bool) {
	if c.first.CompareAndSwap(false, true) {
		return nil, false
	}
	return c.Cache.Get(key)
}
func (c *firstMissCache) Set(key string, value any, ttl time.Duration) {
	c.sets.Add(1)
	c.Cache.Set(key, value, ttl)
}

func TestAnalysisFlight_RecheckCacheHitDoesNotExtendTTL(t *testing.T) {
	m := newAnalysis(t, &fakeCatalog{}, &fakeNodes{})
	warm := cachedAnalysis(t, m, 3, "member-1")
	c := &firstMissCache{Cache: m.cache}
	m.cache = c
	got, err := m.getAnalysis(t.Context(), 3, "member-1", "token")
	if err != nil || got != warm {
		t.Fatalf("cache recheck=%v error=%v", got, err)
	}
	if c.sets.Load() != 0 {
		t.Fatal("cache recheck extended ownership data TTL")
	}
	if _, err := m.getAnalysis(t.Context(), 3, "member-1", "other-token"); err != nil {
		t.Fatal(err)
	}
	if c.sets.Load() != 0 {
		t.Fatal("warm cache read extended ownership data TTL")
	}
	assertNoFlights(t, m)
}

type pausedSetCache struct {
	cache.Cache
	entered chan struct{}
	release chan struct{}
}

func (c *pausedSetCache) Set(key string, value any, ttl time.Duration) {
	close(c.entered)
	<-c.release
	c.Cache.Set(key, value, ttl)
}

// A commit that acquired the active-flight guard before abandonment wins the
// race. Cancellation still returns promptly once that bounded commit finishes;
// it must not delete a completed result or let the caller continue as success.
func TestAnalysisFlight_CompletedPublicationWinsCancellationRace(t *testing.T) {
	m := newAnalysis(t, &fakeCatalog{facts: fixtureCatalog()}, &fakeNodes{nodes: fixtureNodes()})
	stale := cachedAnalysis(t, m, 3, "member-1")
	swapped(t, m, "v2")
	c := &pausedSetCache{Cache: m.cache, entered: make(chan struct{}), release: make(chan struct{})}
	m.cache = c
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := startAnalysis(ctx, m, 3, "member-1", "token")
	receive(t, c.entered)
	cancel()
	close(c.release)
	if got := receive(t, result); !errors.Is(got.err, context.Canceled) {
		t.Fatalf("canceled caller error=%v", got.err)
	}
	got, err := m.getAnalysis(t.Context(), 3, "member-1", "token")
	if err != nil || got == stale || !got.owned[100] {
		t.Fatalf("completed cached rebuild=%v error=%v", got, err)
	}
	assertNoFlights(t, m)
}
