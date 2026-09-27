package records

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
)

// Three natural consumers all observe the same cold profile entry before any
// loader can finish. This barrier does not wait for upstream request count, so
// it remains valid when the owner coalesces those misses into one HTTP request.
type simultaneousMissCache struct {
	cache.Cache
	key     string
	mu      sync.Mutex
	arrived int
	release chan struct{}
}

func (c *simultaneousMissCache) Get(key string) (any, bool) {
	value, found := c.Cache.Get(key)
	if key != c.key {
		return value, found
	}
	c.mu.Lock()
	c.arrived++
	initial := c.arrived <= 3
	if c.arrived == 3 {
		close(c.release)
	}
	c.mu.Unlock()
	if initial {
		select {
		case <-c.release:
		case <-time.After(3 * time.Second):
		}
	}
	return value, found
}

func TestProfileFlight_NaturalConsumersShareColdComponent900(t *testing.T) {
	var profiles atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/Profile/") || r.URL.Query().Get("components") != "900" {
			t.Errorf("unexpected fixture request %s", r.URL.Path)
			w.WriteHeader(400)
			return
		}
		profiles.Add(1)
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{}}}}}`)
	}))
	defer srv.Close()
	c := cache.NewMemoryCache(time.Minute, 0)
	defer c.Close()
	c.Set(coreSettingsCacheKey, &bungie.CoreSettings{ExoticCatalystsRootNodeHash: 9000, ActiveSealsRootNodeHash: 9200}, time.Hour)
	barrier := &simultaneousMissCache{Cache: c, key: recordsCacheKey(3, "fixture-member"), release: make(chan struct{})}
	s := NewService(bungie.NewClient("fixture-key", srv.URL, 100, 100), catalystFixture(t), barrier, time.Minute)
	consumers := []func(context.Context) (time.Time, error){
		func(ctx context.Context) (time.Time, error) {
			_, at, e := s.GetCatalysts(ctx, 3, "fixture-member", "fixture-token")
			return at, e
		},
		func(ctx context.Context) (time.Time, error) {
			_, at, e := s.GetCrafting(ctx, 3, "fixture-member", "fixture-token")
			return at, e
		},
		func(ctx context.Context) (time.Time, error) {
			_, at, e := s.GetSeals(ctx, 3, "fixture-member", "fixture-token")
			return at, e
		},
	}
	type result struct {
		at  time.Time
		err error
	}
	results := make(chan result, 3)
	for _, call := range consumers {
		go func() { at, err := call(t.Context()); results <- result{at, err} }()
	}
	var fetchedAt time.Time
	for range consumers {
		select {
		case got := <-results:
			if got.err != nil {
				t.Fatal(got.err)
			}
			if got.at.IsZero() {
				t.Fatal("missing fetchedAt")
			}
			if !fetchedAt.IsZero() && !got.at.Equal(fetchedAt) {
				t.Fatal("overlapping views received different profile samples")
			}
			fetchedAt = got.at
		case <-time.After(5 * time.Second):
			t.Fatal("consumer timed out")
		}
	}
	t.Logf("three simultaneous cold Catalysts/Crafting/Seals reads: component900 calls=%d", profiles.Load())
	if profiles.Load() != 1 {
		t.Fatalf("component900 calls=%d, want1 shared read", profiles.Load())
	}
	for _, call := range consumers {
		at, err := call(t.Context())
		if err != nil || !at.Equal(fetchedAt) {
			t.Fatalf("warm read at=%v err=%v", at, err)
		}
	}
	if profiles.Load() != 1 {
		t.Fatal("warm consumers fetched component900 again")
	}
}

type profilePending struct {
	reply    chan string
	canceled <-chan struct{}
}
type profileResult struct {
	resp *bungie.RecordsProfileResponse
	at   time.Time
	err  error
}

func profileFixture(t *testing.T, c cache.Cache) (*Service, <-chan profilePending, *atomic.Int32) {
	t.Helper()
	requests := make(chan profilePending, 16)
	hits := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		p := profilePending{reply: make(chan string, 1), canceled: r.Context().Done()}
		requests <- p
		select {
		case body := <-p.reply:
			fmt.Fprint(w, body)
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
			http.Error(w, "fixture timeout", 400)
		}
	}))
	t.Cleanup(srv.Close)
	return NewService(bungie.NewClient("fixture-key", srv.URL, 1000, 1000), nil, c, time.Minute), requests, hits
}
func memoryProfileFixture(t *testing.T) (*Service, <-chan profilePending, *atomic.Int32) {
	c := cache.NewMemoryCache(time.Minute, 0)
	t.Cleanup(c.Close)
	return profileFixture(t, c)
}
func recordBody(state int) string {
	return fmt.Sprintf(`{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{"1":{"state":%d}}}}}}`, state)
}
func startProfile(ctx context.Context, s *Service, platform int, member, token string) <-chan profileResult {
	done := make(chan profileResult, 1)
	go func() {
		resp, at, err := s.getProfileRecords(ctx, platform, member, token)
		done <- profileResult{resp, at, err}
	}()
	return done
}
func profileReceive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(3 * time.Second):
		t.Fatal("fixture event timed out")
		var zero T
		return zero
	}
}
func waitProfile(t *testing.T, s *Service, n int) *profileFlight {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		s.profileMu.Lock()
		var found *profileFlight
		if len(s.profileFlights) == 1 {
			for _, f := range s.profileFlights {
				if f.waiters == n {
					found = f
				}
			}
		}
		s.profileMu.Unlock()
		if found != nil {
			return found
		}
		select {
		case <-deadline:
			t.Fatal("expected waiters did not join")
			return nil
		default:
			runtime.Gosched()
		}
	}
}
func noProfileFlights(t *testing.T, s *Service) {
	t.Helper()
	s.profileMu.Lock()
	defer s.profileMu.Unlock()
	if len(s.profileFlights) != 0 {
		t.Fatal("finished/abandoned profile flight retained")
	}
}
func requireProfile(t *testing.T, got profileResult, state int) {
	t.Helper()
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.resp.Response.ProfileRecords.Data.Records["1"].State != state || got.at.IsZero() {
		t.Fatalf("unexpected profile/time: %+v", got)
	}
}

func TestProfileFlight_CallerCancellationAndContextValues(t *testing.T) {
	for _, cancelLeader := range []bool{true, false} {
		t.Run(fmt.Sprintf("leader=%v", cancelLeader), func(t *testing.T) {
			type key struct{}
			s, requests, hits := memoryProfileFixture(t)
			base, baseCancel := context.WithTimeout(context.WithValue(t.Context(), key{}, "request-value"), time.Minute)
			defer baseCancel()
			leaderCtx, cancelFirst := context.WithCancel(base)
			defer cancelFirst()
			followerCtx, cancelSecond := context.WithCancel(t.Context())
			defer cancelSecond()
			first := startProfile(leaderCtx, s, 3, "member", "token")
			p := profileReceive(t, requests)
			second := startProfile(followerCtx, s, 3, "member", "token")
			f := waitProfile(t, s, 2)
			if f.ctx.Value(key{}) != "request-value" {
				t.Fatal("shared context lost values")
			}
			if deadline, ok := f.ctx.Deadline(); !ok || time.Until(deadline) < 2*time.Minute || time.Until(deadline) > profileLoadTimeout {
				t.Fatal("shared work inherited leader deadline or has no bound")
			}
			canceled, remaining := first, second
			if cancelLeader {
				cancelFirst()
			} else {
				cancelSecond()
				canceled, remaining = second, first
			}
			if got := profileReceive(t, canceled); !errors.Is(got.err, context.Canceled) {
				t.Fatalf("error=%v", got.err)
			}
			if f.ctx.Err() != nil {
				t.Fatal("one waiter canceled shared work")
			}
			p.reply <- recordBody(1)
			requireProfile(t, profileReceive(t, remaining), 1)
			if hits.Load() != 1 {
				t.Fatalf("profile requests=%d", hits.Load())
			}
			noProfileFlights(t, s)
		})
	}
}

func TestProfileFlight_LastWaiterCancelsHTTP(t *testing.T) {
	s, requests, _ := memoryProfileFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	first := startProfile(ctx, s, 3, "member", "token")
	p := profileReceive(t, requests)
	second := startProfile(ctx, s, 3, "member", "token")
	waitProfile(t, s, 2)
	cancel()
	for _, ch := range []<-chan profileResult{first, second} {
		if got := profileReceive(t, ch); !errors.Is(got.err, context.Canceled) {
			t.Fatalf("error=%v", got.err)
		}
	}
	profileReceive(t, p.canceled)
	noProfileFlights(t, s)
	if _, ok := s.cache.Get(recordsCacheKey(3, "member")); ok {
		t.Fatal("canceled response was cached")
	}
	next := startProfile(t.Context(), s, 3, "member", "token")
	profileReceive(t, requests).reply <- recordBody(2)
	requireProfile(t, profileReceive(t, next), 2)
}

func TestProfileFlight_RefreshSeparatesButManifestSwapShares(t *testing.T) {
	for _, refresh := range []bool{true, false} {
		t.Run(fmt.Sprintf("refresh=%v", refresh), func(t *testing.T) {
			s, requests, hits := memoryProfileFixture(t)
			old := startProfile(t.Context(), s, 3, "member", "token")
			first := profileReceive(t, requests)
			if refresh {
				s.InvalidateCache(3, "member")
			} else {
				if err := s.OnVersionChanged("next"); err != nil {
					t.Fatal(err)
				}
			}
			fresh := startProfile(t.Context(), s, 3, "member", "token")
			if refresh {
				second := profileReceive(t, requests)
				second.reply <- recordBody(2)
				current := profileReceive(t, fresh)
				requireProfile(t, current, 2)
				first.reply <- recordBody(1)
				requireProfile(t, profileReceive(t, old), 1)
				cached, _ := s.cache.Get(recordsCacheKey(3, "member"))
				if cached.(*cachedRecords).resp != current.resp {
					t.Fatal("pre-refresh result replaced current cache")
				}
				if hits.Load() != 2 {
					t.Fatalf("profile requests=%d", hits.Load())
				}
			} else {
				waitProfile(t, s, 2)
				first.reply <- recordBody(1)
				a, b := profileReceive(t, old), profileReceive(t, fresh)
				requireProfile(t, a, 1)
				requireProfile(t, b, 1)
				if a.resp != b.resp || !a.at.Equal(b.at) {
					t.Fatal("Manifest swap split the raw profile sample")
				}
				if hits.Load() != 1 {
					t.Fatalf("profile requests=%d", hits.Load())
				}
				cached, ok := s.cache.Get(recordsCacheKey(3, "member"))
				if !ok || cached.(*cachedRecords).resp != a.resp {
					t.Fatal("Manifest swap retired raw profile publication")
				}
			}
		})
	}
}

func TestProfileFlight_MembershipPlatformAndCredentialIsolation(t *testing.T) {
	for _, dimension := range []string{"membership", "platform", "credential"} {
		t.Run(dimension, func(t *testing.T) {
			s, requests, hits := memoryProfileFixture(t)
			old := startProfile(t.Context(), s, 3, "member", "token")
			first := profileReceive(t, requests)
			platform, member, token := 3, "member", "token"
			switch dimension {
			case "membership":
				member = "other"
			case "platform":
				platform = 2
			case "credential":
				token = "new-token"
			}
			fresh := startProfile(t.Context(), s, platform, member, token)
			second := profileReceive(t, requests)
			second.reply <- recordBody(2)
			requireProfile(t, profileReceive(t, fresh), 2)
			first.reply <- `{"ErrorCode":99,"Message":"fixture old authorization failed"}`
			if got := profileReceive(t, old); got.err == nil {
				t.Fatal("old authorization failure was lost")
			}
			if hits.Load() != 2 {
				t.Fatalf("profile requests=%d", hits.Load())
			}
		})
	}
}

func TestProfileFlight_FailuresSharedNotCached(t *testing.T) {
	s, requests, hits := memoryProfileFixture(t)
	first := startProfile(t.Context(), s, 3, "member", "token")
	p := profileReceive(t, requests)
	second := startProfile(t.Context(), s, 3, "member", "token")
	waitProfile(t, s, 2)
	p.reply <- `{"ErrorCode":99,"Message":"fixture failure"}`
	for _, ch := range []<-chan profileResult{first, second} {
		if got := profileReceive(t, ch); got.err == nil {
			t.Fatal("expected failure")
		}
	}
	noProfileFlights(t, s)
	if _, ok := s.cache.Get(recordsCacheKey(3, "member")); ok {
		t.Fatal("failure cached")
	}
	next := startProfile(t.Context(), s, 3, "member", "token")
	profileReceive(t, requests).reply <- recordBody(1)
	requireProfile(t, profileReceive(t, next), 1)
	if hits.Load() != 2 {
		t.Fatalf("profile requests=%d", hits.Load())
	}
}

func TestProfileFlight_NilAndNoOpCachesShareOnlyOverlappingWork(t *testing.T) {
	for _, kind := range []string{"nil", "noop"} {
		t.Run(kind, func(t *testing.T) {
			var c cache.Cache
			if kind == "noop" {
				c = cache.NewNoOpCache()
			}
			s, requests, hits := profileFixture(t, c)
			first := startProfile(t.Context(), s, 3, "member", "token")
			p := profileReceive(t, requests)
			second := startProfile(t.Context(), s, 3, "member", "token")
			waitProfile(t, s, 2)
			p.reply <- recordBody(1)
			a, b := profileReceive(t, first), profileReceive(t, second)
			requireProfile(t, a, 1)
			requireProfile(t, b, 1)
			if a.resp != b.resp || !a.at.Equal(b.at) {
				t.Fatal("overlapping responses not shared")
			}
			noProfileFlights(t, s)
			next := startProfile(t.Context(), s, 3, "member", "token")
			profileReceive(t, requests).reply <- recordBody(2)
			requireProfile(t, profileReceive(t, next), 2)
			if hits.Load() != 2 {
				t.Fatalf("uncached sequential read reused completed flight; requests=%d", hits.Load())
			}
		})
	}
}

type profileRecheckCache struct {
	cache.Cache
	gets atomic.Int32
	sets atomic.Int32
}

func (c *profileRecheckCache) Get(key string) (any, bool) {
	if c.gets.Add(1) == 1 {
		return nil, false
	}
	return c.Cache.Get(key)
}
func (c *profileRecheckCache) Set(key string, v any, ttl time.Duration) {
	c.sets.Add(1)
	c.Cache.Set(key, v, ttl)
}
func TestProfileFlight_RecheckHitPreservesTTLAndFetchedAt(t *testing.T) {
	c := cache.NewMemoryCache(time.Minute, 0)
	defer c.Close()
	warm := &cachedRecords{resp: &bungie.RecordsProfileResponse{}, fetchedAt: time.Unix(123, 0)}
	c.Set(recordsCacheKey(3, "member"), warm, time.Minute)
	wrapped := &profileRecheckCache{Cache: c}
	s := NewService(nil, nil, wrapped, time.Minute)
	resp, at, err := s.getProfileRecords(t.Context(), 3, "member", "token")
	if err != nil || resp != warm.resp || !at.Equal(warm.fetchedAt) || wrapped.sets.Load() != 0 {
		t.Fatalf("recheck resp=%v at=%v err=%v sets=%d", resp, at, err, wrapped.sets.Load())
	}
	_, _, err = s.getProfileRecords(t.Context(), 3, "member", "new-token")
	if err != nil || wrapped.sets.Load() != 0 {
		t.Fatal("warm cache extended TTL")
	}
}

func TestProfileFlight_WrongCacheTypeWarnsWithoutIdentityOrToken(t *testing.T) {
	s, requests, _ := memoryProfileFixture(t)
	s.cache.Set(recordsCacheKey(3, "private-member"), "wrong type", time.Minute)
	var logs bytes.Buffer
	ctx := observability.WithLogger(t.Context(), slog.New(slog.NewTextHandler(&logs, nil)))
	got := startProfile(ctx, s, 3, "private-member", "private-token")
	profileReceive(t, requests).reply <- recordBody(1)
	requireProfile(t, profileReceive(t, got), 1)
	if !strings.Contains(logs.String(), "cached value has the wrong type") {
		t.Fatal("wrong-type warning lost")
	}
	for _, secret := range []string{"private-member", "private-token", "records:3:"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatal("cache warning exposed identity or credential")
		}
	}
}

// A cache read has no context parameter. Hold the first and replacement inner
// rechecks so late completion and identity-safe map removal are deterministic.
type profilePausedCache struct {
	cache.Cache
	gets    atomic.Int32
	entered chan chan struct{}
}

func (c *profilePausedCache) Get(key string) (any, bool) {
	n := c.gets.Add(1)
	if n == 2 || n == 5 {
		release := make(chan struct{})
		c.entered <- release
		<-release
	}
	return c.Cache.Get(key)
}
func TestProfileFlight_AbandonedRecheckCannotRemoveReplacement(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%v", timeout), func(t *testing.T) {
			s, requests, hits := memoryProfileFixture(t)
			paused := &profilePausedCache{Cache: s.cache, entered: make(chan chan struct{}, 2)}
			s.cache = paused
			if timeout {
				s.profileTimeout = 100 * time.Millisecond
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			first := startProfile(ctx, s, 3, "member", "token")
			oldRelease := profileReceive(t, paused.entered)
			second := startProfile(ctx, s, 3, "member", "token")
			oldFlight := waitProfile(t, s, 2)
			want := context.Canceled
			if timeout {
				want = context.DeadlineExceeded
			} else {
				cancel()
			}
			for _, ch := range []<-chan profileResult{first, second} {
				if got := profileReceive(t, ch); !errors.Is(got.err, want) {
					t.Fatalf("error=%v want=%v", got.err, want)
				}
			}
			noProfileFlights(t, s)
			s.profileTimeout = profileLoadTimeout
			replacement := startProfile(t.Context(), s, 3, "member", "token")
			newRelease := profileReceive(t, paused.entered)
			newFlight := waitProfile(t, s, 1)
			if oldFlight == newFlight {
				t.Fatal("replacement joined abandoned flight")
			}
			close(oldRelease)
			profileReceive(t, oldFlight.done)
			if waitProfile(t, s, 1) != newFlight {
				t.Fatal("late completion removed replacement")
			}
			if _, ok := paused.Cache.Get(recordsCacheKey(3, "member")); ok {
				t.Fatal("abandoned work repopulated cache")
			}
			close(newRelease)
			profileReceive(t, requests).reply <- recordBody(2)
			requireProfile(t, profileReceive(t, replacement), 2)
			if hits.Load() != 1 {
				t.Fatalf("abandoned cache read initiated HTTP; requests=%d", hits.Load())
			}
		})
	}
}
