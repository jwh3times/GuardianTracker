package bungie

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordWaits replaces the client's backoff sleep so tests assert the waits
// the client chose instead of spending them.
func recordWaits(c *Client) *[]time.Duration {
	var mu sync.Mutex
	waits := []time.Duration{}
	c.sleep = func(ctx context.Context, d time.Duration) error {
		mu.Lock()
		waits = append(waits, d)
		mu.Unlock()
		return ctx.Err()
	}
	return &waits
}

// throttleThenOK answers the first `throttled` calls with a Bungie envelope
// throttle (HTTP 200, the shape the OpenAPI spec documents) and then succeeds.
func throttleThenOK(t *testing.T, throttled int32, code, seconds int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= throttled {
			fmt.Fprintf(w, `{"ErrorCode":%d,"ErrorStatus":"Throttled","Message":"slow down","ThrottleSeconds":%d}`, code, seconds)
			return
		}
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"version":"ok"}}`)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestEnvelopeThrottleCodesAreRetried(t *testing.T) {
	for _, code := range []int{31, 35, 36, 37, 51, 54, 55, 56, 57} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			srv, calls := throttleThenOK(t, 1, code, 2)
			c := NewClient("k", srv.URL, 100, 100)
			waits := recordWaits(c)

			m, err := c.GetManifest(context.Background())
			if err != nil {
				t.Fatalf("GetManifest after throttle: %v", err)
			}
			if m.Response.Version != "ok" || calls.Load() != 2 {
				t.Errorf("version = %q calls = %d, want ok after 2", m.Response.Version, calls.Load())
			}
			if len(*waits) != 1 || (*waits)[0] != 2*time.Second {
				t.Errorf("waits = %v, want [2s] from ThrottleSeconds", *waits)
			}
		})
	}
}

// Owner decision on #402: a throttle that names no wait still waits, on the
// same linear floor HTTP 429/5xx retries use.
func TestZeroThrottleSecondsUsesLinearBackoff(t *testing.T) {
	srv, calls := throttleThenOK(t, 2, 36, 0)
	c := NewClient("k", srv.URL, 100, 100)
	waits := recordWaits(c)

	if _, err := c.GetManifest(context.Background()); err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
	if want := []time.Duration{time.Second, 2 * time.Second}; fmt.Sprint(*waits) != fmt.Sprint(want) {
		t.Errorf("waits = %v, want %v", *waits, want)
	}
}

func TestLongThrottleIsSurfacedWithoutWaiting(t *testing.T) {
	srv, calls := throttleThenOK(t, 1, 35, 60)
	c := NewClient("k", srv.URL, 100, 100)
	waits := recordWaits(c)

	_, err := c.GetManifest(context.Background())
	var bErr *BungieError
	if !errors.As(err, &bErr) || bErr.ErrorCode != 35 || bErr.ThrottleSeconds != 60 {
		t.Fatalf("err = %v, want the code-35 throttle carrying 60s", err)
	}
	if calls.Load() != 1 || len(*waits) != 0 {
		t.Errorf("calls = %d waits = %v, want one call and no wait", calls.Load(), *waits)
	}
}

func TestPersistentThrottleReturnsTheThrottleAfterTheBudget(t *testing.T) {
	srv, calls := throttleThenOK(t, 100, 57, 1)
	c := NewClient("k", srv.URL, 100, 100)
	waits := recordWaits(c)

	_, err := c.GetManifest(context.Background()) // budget: 3 retries
	var bErr *BungieError
	if !errors.As(err, &bErr) || !IsThrottle(bErr.ErrorCode) {
		t.Fatalf("err = %v, want the throttle BungieError", err)
	}
	if calls.Load() != 4 || len(*waits) != 3 {
		t.Errorf("calls = %d waits = %d, want 4 calls and no wait after the last", calls.Load(), len(*waits))
	}
}

func TestNonThrottleBungieErrorIsNotRetried(t *testing.T) {
	srv, calls := throttleThenOK(t, 100, 5, 0)
	c := NewClient("k", srv.URL, 100, 100)
	recordWaits(c)

	if _, err := c.GetManifest(context.Background()); err == nil {
		t.Fatal("expected the code-5 error")
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestExhaustedHTTP429IsRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := NewClient("k", srv.URL, 100, 100)
	waits := recordWaits(c)

	_, err := c.GetManifest(context.Background())
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("err = %v, want ErrRateLimited", err)
	}
	if len(*waits) != 3 {
		t.Errorf("waits = %v, want 3 (none after the last attempt)", *waits)
	}
}

func TestThrottleWaitStopsWhenTheCallerLeaves(t *testing.T) {
	srv, calls := throttleThenOK(t, 100, 36, 5)
	c := NewClient("k", srv.URL, 100, 100)
	ctx, cancel := context.WithCancel(context.Background())
	c.sleep = func(context.Context, time.Duration) error {
		cancel()
		return ctx.Err()
	}

	if _, err := c.GetManifest(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}
