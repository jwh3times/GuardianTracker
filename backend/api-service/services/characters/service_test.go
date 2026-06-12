package characters

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

func newTestService(t *testing.T, handler http.HandlerFunc) (*Service, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client := bungie.NewClient("k", srv.URL, 100, 100)
	c := cache.NewMemoryCache(time.Minute, time.Minute)
	return NewService(client, c, time.Minute), srv.Close
}

func TestGetCharacters_MapsAndSortsMostRecentFirst(t *testing.T) {
	svc, closeSrv := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"characters":{"data":{
			"older":{"characterId":"older","classType":0,"raceType":0,"light":2000,
				"emblemPath":"/e1.png","dateLastPlayed":"2026-06-01T00:00:00Z"},
			"newer":{"characterId":"newer","classType":2,"raceType":2,"light":2010,
				"emblemPath":"","dateLastPlayed":"2026-06-10T00:00:00Z"}}}}}`)
	})
	defer closeSrv()

	chars, err := svc.GetCharacters(context.Background(), 3, "id", "tok")
	if err != nil {
		t.Fatalf("GetCharacters: %v", err)
	}
	if len(chars) != 2 {
		t.Fatalf("len = %d, want 2", len(chars))
	}
	// Most recently played sorts first.
	if chars[0].CharacterID != "newer" {
		t.Errorf("chars[0] = %q, want newer", chars[0].CharacterID)
	}
	if chars[0].ClassName != "Warlock" || chars[0].RaceName != "Exo" {
		t.Errorf("newer class/race = %q/%q", chars[0].ClassName, chars[0].RaceName)
	}
	// absoluteURL prefixes a non-empty path and leaves empty paths empty.
	if chars[1].EmblemPath != "https://www.bungie.net/e1.png" {
		t.Errorf("older emblem = %q", chars[1].EmblemPath)
	}
	if chars[0].EmblemPath != "" {
		t.Errorf("empty emblem should stay empty, got %q", chars[0].EmblemPath)
	}
}

func TestGetCharacters_UsesCacheOnSecondCall(t *testing.T) {
	var calls int
	svc, closeSrv := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"characters":{"data":{
			"c1":{"characterId":"c1","classType":1,"raceType":1,"light":2000,"dateLastPlayed":"2026-06-10T00:00:00Z"}}}}}`)
	})
	defer closeSrv()

	if _, err := svc.GetCharacters(context.Background(), 3, "id", "tok"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := svc.GetCharacters(context.Background(), 3, "id", "tok"); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if calls != 1 {
		t.Errorf("upstream calls = %d, want 1 (second served from cache)", calls)
	}

	// InvalidateCache forces a refetch.
	svc.InvalidateCache(3, "id")
	if _, err := svc.GetCharacters(context.Background(), 3, "id", "tok"); err != nil {
		t.Fatalf("third call: %v", err)
	}
	if calls != 2 {
		t.Errorf("upstream calls = %d, want 2 after invalidation", calls)
	}
}

func TestGetCharacters_PropagatesUpstreamError(t *testing.T) {
	svc, closeSrv := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":5,"ErrorStatus":"SystemDisabled","Message":"down"}`)
	})
	defer closeSrv()

	if _, err := svc.GetCharacters(context.Background(), 3, "id", "tok"); err == nil {
		t.Fatal("expected error from upstream ErrorCode 5")
	}
}
