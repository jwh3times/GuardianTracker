package records

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
)

// blockingRecords serves the profile-records component, holding the first
// request open until the test releases it, so a refresh can land in the middle
// of a load deterministically rather than by goroutine luck.
type blockingRecords struct {
	entered  chan struct{}
	release  chan struct{}
	once     sync.Once
	mu       sync.Mutex
	requests int
}

func (b *blockingRecords) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	b.mu.Lock()
	b.requests++
	b.mu.Unlock()

	b.once.Do(func() { close(b.entered) })
	<-b.release // closed, so only the first request is actually held

	fmt.Fprint(w, `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{}}}}}`)
}

func (b *blockingRecords) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.requests
}

// Profile records already loading when a refresh lands still answer their own
// request, but must not refill the entry the refresh just cleared — otherwise
// catalysts, crafting, and seals keep serving pre-refresh data after the
// refresh reported success.
func TestGetProfileRecords_ALoadInFlightDuringARefreshIsNotReused(t *testing.T) {
	records := &blockingRecords{entered: make(chan struct{}), release: make(chan struct{})}
	srv := httptest.NewServer(records)
	defer srv.Close()

	c := cache.NewMemoryCache(time.Minute, 0)
	defer c.Close()
	s := NewService(bungie.NewClient("k", srv.URL, 100, 100), &fakeRecordsManifest{}, c, time.Minute)

	done := make(chan error, 1)
	go func() {
		_, _, err := s.getProfileRecords(context.Background(), 3, "member-1", "token")
		done <- err
	}()

	<-records.entered                // the load is underway
	s.InvalidateCache(3, "member-1") // ... and the user asks for a refresh
	close(records.release)           // ... and only then does the load finish

	if err := <-done; err != nil {
		t.Fatalf("the initiating request must still get its own result: %v", err)
	}

	if cached, found := c.Get(recordsCacheKey(3, "member-1")); found {
		t.Fatalf("pre-refresh records %v were published after the refresh returned", cached)
	}

	if _, _, err := s.getProfileRecords(context.Background(), 3, "member-1", "token"); err != nil {
		t.Fatalf("getProfileRecords after refresh: %v", err)
	}
	if got := records.count(); got != 2 {
		t.Errorf("Bungie requests = %d, want 2 — the read after the refresh was served from stale cache", got)
	}
}

// One user's refresh must not throw away another user's in-flight work.
func TestGetProfileRecords_ARefreshDoesNotRetireAnotherMembershipsLoad(t *testing.T) {
	records := &blockingRecords{entered: make(chan struct{}), release: make(chan struct{})}
	srv := httptest.NewServer(records)
	defer srv.Close()

	c := cache.NewMemoryCache(time.Minute, 0)
	defer c.Close()
	s := NewService(bungie.NewClient("k", srv.URL, 100, 100), &fakeRecordsManifest{}, c, time.Minute)

	done := make(chan error, 1)
	go func() {
		_, _, err := s.getProfileRecords(context.Background(), 3, "member-1", "token")
		done <- err
	}()

	<-records.entered
	s.InvalidateCache(3, "member-2") // somebody else refreshes
	close(records.release)

	if err := <-done; err != nil {
		t.Fatalf("getProfileRecords: %v", err)
	}
	if _, found := c.Get(recordsCacheKey(3, "member-1")); !found {
		t.Error("another membership's refresh discarded this membership's loaded records")
	}
}

// A manifest swap and a membership refresh are independent axes. A swap must
// not discard the rate-limited profile records, which is what the manifest
// fence has always deliberately left alone.
func TestOnVersionChanged_LeavesProfileRecordsAlone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"profileRecords":{"data":{"records":{}}}}}`)
	}))
	defer srv.Close()

	c := cache.NewMemoryCache(time.Minute, 0)
	defer c.Close()
	s := NewService(bungie.NewClient("k", srv.URL, 100, 100), &fakeRecordsManifest{}, c, time.Minute)

	if _, _, err := s.getProfileRecords(context.Background(), 3, "member-1", "token"); err != nil {
		t.Fatalf("getProfileRecords: %v", err)
	}
	if err := s.OnVersionChanged("v2"); err != nil {
		t.Fatalf("OnVersionChanged: %v", err)
	}

	if _, found := c.Get(recordsCacheKey(3, "member-1")); !found {
		t.Error("a manifest swap evicted profile records; only a membership refresh should")
	}
}
