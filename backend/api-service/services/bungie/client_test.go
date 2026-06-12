package bungie

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestGetManifest_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "test-key" {
			t.Errorf("X-API-Key = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"version":"123.45",
			"mobileWorldContentPaths":{"en":"/common/world.content"}}}`)
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.URL, 100, 100)
	m, err := c.GetManifest(context.Background())
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if m.Response.Version != "123.45" {
		t.Errorf("Version = %q", m.Response.Version)
	}
	if m.Response.MobileWorldContentPaths.En != "/common/world.content" {
		t.Errorf("En path = %q", m.Response.MobileWorldContentPaths.En)
	}
}

func TestParseResponse_BungieErrorCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ErrorCode":5,"ErrorStatus":"SystemDisabled","Message":"down for maintenance","ThrottleSeconds":0}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	_, err := c.GetManifest(context.Background())
	var bErr *BungieError
	if !errors.As(err, &bErr) {
		t.Fatalf("err = %v, want *BungieError", err)
	}
	if bErr.ErrorCode != 5 || bErr.ErrorStatus != "SystemDisabled" {
		t.Errorf("BungieError = %+v", bErr)
	}
}

func TestParseResponse_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{not json`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	if _, err := c.GetManifest(context.Background()); err == nil {
		t.Fatal("malformed JSON parsed without error")
	}
}

func TestDoRequestWithRetry_RetriesOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"version":"ok"}}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	m, err := c.GetManifest(context.Background())
	if err != nil {
		t.Fatalf("GetManifest after retries: %v", err)
	}
	if m.Response.Version != "ok" {
		t.Errorf("Version = %q", m.Response.Version)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3 (two 5xx then success)", calls.Load())
	}
}

func TestDoRequestWithRetry_HonorsRetryAfterOn429(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"version":"ok"}}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	if _, err := c.GetManifest(context.Background()); err != nil {
		t.Fatalf("GetManifest after 429: %v", err)
	}
	if calls.Load() != 2 {
		t.Errorf("calls = %d, want 2", calls.Load())
	}
}

func TestDoRequestWithRetry_MaxRetriesExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	if _, err := c.GetManifest(context.Background()); err == nil {
		t.Fatal("expected max-retries error")
	}
}

func TestGetProfile_SendsAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer bungie-token" {
			t.Errorf("Authorization = %q", got)
		}
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{}}`)
	}))
	defer srv.Close()

	c := NewClient("k", srv.URL, 100, 100)
	if _, err := c.GetProfile(context.Background(), 3, "4611686018467260757", "bungie-token", []int{800}); err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
}
