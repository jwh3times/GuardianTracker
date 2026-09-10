package characters

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

// blockingRoster serves one roster, holding the first request open until the
// test releases it. That is what lets a refresh land in the middle of a load
// deterministically, rather than hoping a goroutine interleaves the right way.
type blockingRoster struct {
	entered  chan struct{}
	release  chan struct{}
	once     sync.Once
	mu       sync.Mutex
	requests int
}

func (b *blockingRoster) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	b.mu.Lock()
	b.requests++
	b.mu.Unlock()

	b.once.Do(func() { close(b.entered) })
	<-b.release // closed, so only the first request is actually held

	fmt.Fprint(w, `{"ErrorCode":1,"Response":{"characters":{"data":{
		"c1":{"characterId":"c1","classType":0,"raceType":0,"light":2000,
			"emblemPath":"/e1.png","dateLastPlayed":"2026-06-01T00:00:00Z"}}}}}`)
}

func (b *blockingRoster) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.requests
}

// A refresh that returns must mean the next read fetches fresh data. A roster
// load already in flight when the refresh lands still answers its own request,
// but must not refill the entry the refresh just cleared — otherwise the
// refresh reported success and the next reader still sees pre-refresh data.
func TestGetCharacters_ALoadInFlightDuringARefreshIsNotReused(t *testing.T) {
	roster := &blockingRoster{entered: make(chan struct{}), release: make(chan struct{})}
	srv := httptest.NewServer(roster)
	defer srv.Close()

	c := cache.NewMemoryCache(time.Minute, 0)
	defer c.Close()
	svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), c, time.Minute)

	type result struct {
		chars []Character
		err   error
	}
	done := make(chan result, 1)
	go func() {
		chars, err := svc.GetCharacters(context.Background(), 3, "member-1", "tok")
		done <- result{chars, err}
	}()

	<-roster.entered                   // the load is underway
	svc.InvalidateCache(3, "member-1") // ... and the user asks for a refresh
	close(roster.release)              // ... and only then does the load finish

	got := <-done
	if got.err != nil {
		t.Fatalf("the initiating request must still get its own result: %v", got.err)
	}
	if len(got.chars) != 1 {
		t.Fatalf("chars = %d, want the roster it loaded", len(got.chars))
	}

	if cached, found := c.Get(charactersCacheKey(3, "member-1")); found {
		t.Fatalf("a pre-refresh roster %v was published after the refresh returned", cached)
	}

	// The promise the refresh made, stated the way a user would: the next read
	// really does go back to Bungie.
	if _, err := svc.GetCharacters(context.Background(), 3, "member-1", "tok"); err != nil {
		t.Fatalf("GetCharacters after refresh: %v", err)
	}
	if got := roster.count(); got != 2 {
		t.Errorf("Bungie requests = %d, want 2 — the read after the refresh was served from stale cache", got)
	}
}

// One user's refresh must not throw away another user's in-flight work.
func TestGetCharacters_ARefreshDoesNotRetireAnotherMembershipsLoad(t *testing.T) {
	roster := &blockingRoster{entered: make(chan struct{}), release: make(chan struct{})}
	srv := httptest.NewServer(roster)
	defer srv.Close()

	c := cache.NewMemoryCache(time.Minute, 0)
	defer c.Close()
	svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), c, time.Minute)

	done := make(chan error, 1)
	go func() {
		_, err := svc.GetCharacters(context.Background(), 3, "member-1", "tok")
		done <- err
	}()

	<-roster.entered
	svc.InvalidateCache(3, "member-2") // somebody else refreshes
	close(roster.release)

	if err := <-done; err != nil {
		t.Fatalf("GetCharacters: %v", err)
	}
	if _, found := c.Get(charactersCacheKey(3, "member-1")); !found {
		t.Error("another membership's refresh discarded this membership's loaded roster")
	}
}
