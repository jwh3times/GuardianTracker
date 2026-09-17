package characters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
)

const currentActivityCharacterID = "2305843009263456789"

// Hashes and names from the 2026-09-15 owner capture.
const (
	lakeOfShadowsHash   uint32 = 739181303
	strikeModeHash      uint32 = 4110605575
	quickplayMasterHash uint32 = 451570814
)

func currentActivityDefinitions() *fakeActivityDefinitions {
	return &fakeActivityDefinitions{
		definitions: map[uint32]*bungie.ActivityDefinition{
			lakeOfShadowsHash:   {Hash: lakeOfShadowsHash, DisplayProperties: bungie.DisplayProperties{Name: "Lake of Shadows"}},
			quickplayMasterHash: {Hash: quickplayMasterHash, DisplayProperties: bungie.DisplayProperties{Name: "Quickplay: Master"}},
		},
		modes: map[uint32]*bungie.ActivityModeDefinition{
			strikeModeHash: {Hash: strikeModeHash, DisplayProperties: bungie.DisplayProperties{Name: "Strike"}},
		},
	}
}

// currentActivityServer answers the component-200 roster and returns
// activityBody for the component-204 request.
func currentActivityServer(t *testing.T, activityBody string, activityRequests *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Destiny2/3/Profile/membership/" {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Query().Get("components") {
		case "200":
			fmt.Fprintf(w, `{"ErrorCode":1,"Response":{"characters":{"data":{"%s":{"characterId":"%s"}}}}}`, currentActivityCharacterID, currentActivityCharacterID)
		case "204":
			if activityRequests != nil {
				*activityRequests++
			}
			if got := r.Header.Get("Authorization"); got != "Bearer token" {
				t.Errorf("Authorization = %q", got)
			}
			fmt.Fprint(w, activityBody)
		default:
			t.Errorf("unexpected components %q", r.URL.Query().Get("components"))
			http.NotFound(w, r)
		}
	}))
}

func activityEntry(fields string) string {
	return fmt.Sprintf(`{"ErrorCode":1,"Response":{"characterActivities":{"privacy":2,"data":{"%s":{%s}}}}}`, currentActivityCharacterID, fields)
}

func TestGetCurrentActivity_ProjectsResolvedActivity(t *testing.T) {
	var requests int
	srv := currentActivityServer(t, activityEntry(fmt.Sprintf(`
		"dateActivityStarted":"2026-09-15T16:37:11Z",
		"currentActivityHash":%d,
		"currentActivityModeHash":%d,
		"currentActivityModeType":18,
		"currentActivityModeHashes":[1164760493,2394616003,%d],
		"currentActivityModeTypes":[7,18,3],
		"currentPlaylistActivityHash":%d`, lakeOfShadowsHash, strikeModeHash, strikeModeHash, quickplayMasterHash)), &requests)
	defer srv.Close()

	svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), nil, currentActivityDefinitions(), cache.NewNoOpCache(), time.Minute)
	before := time.Now().UTC()
	current, err := svc.GetCurrentActivity(context.Background(), 3, "membership", currentActivityCharacterID, "token")
	if err != nil {
		t.Fatalf("GetCurrentActivity: %v", err)
	}
	if requests != 1 {
		t.Fatalf("component-204 requests = %d, want 1", requests)
	}
	if current.State != CurrentActivityReady || current.ActivityName != "Lake of Shadows" ||
		current.ModeName != "Strike" || current.PlaylistName != "Quickplay: Master" {
		t.Fatalf("current activity = %+v", current)
	}
	if current.FetchedAt.Before(before) {
		t.Fatalf("FetchedAt = %v, want response time on or after %v", current.FetchedAt, before)
	}

	raw, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	if body := string(raw); containsAny(body, currentActivityCharacterID, "membership", "Started", "739181303", "nstance") {
		t.Fatalf("response leaks identifiers or unverified freshness facts: %s", body)
	}
}

func TestGetCurrentActivity_DistinguishesStates(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantState    CurrentActivityState
		wantActivity string
		wantMode     string
		wantPlaylist string
	}{
		{"zero hash is idle", activityEntry(`"currentActivityHash":0,"currentActivityModeHash":0`), CurrentActivityIdle, "", "", ""},
		{"absent component is unavailable", `{"ErrorCode":1,"Response":{}}`, CurrentActivityUnavailable, "", "", ""},
		{"null data is unavailable", `{"ErrorCode":1,"Response":{"characterActivities":{"privacy":2,"data":null}}}`, CurrentActivityUnavailable, "", "", ""},
		{"disabled component is unavailable", fmt.Sprintf(`{"ErrorCode":1,"Response":{"characterActivities":{"privacy":2,"disabled":true,"data":{"%s":{"currentActivityHash":0}}}}}`, currentActivityCharacterID), CurrentActivityUnavailable, "", "", ""},
		{"missing Guardian entry is unavailable", `{"ErrorCode":1,"Response":{"characterActivities":{"privacy":2,"data":{}}}}`, CurrentActivityUnavailable, "", "", ""},
		{"absent activity hash is unavailable", activityEntry(`"currentActivityModeHash":0`), CurrentActivityUnavailable, "", "", ""},
		{"unresolved activity hash is unknown", activityEntry(fmt.Sprintf(`"currentActivityHash":12345,"currentActivityModeHash":%d`, strikeModeHash)), CurrentActivityUnknown, "", "Strike", ""},
		{"absent mode and playlist are omitted", activityEntry(fmt.Sprintf(`"currentActivityHash":%d`, lakeOfShadowsHash)), CurrentActivityReady, "Lake of Shadows", "", ""},
		{"unresolved mode and playlist are omitted", activityEntry(fmt.Sprintf(`"currentActivityHash":%d,"currentActivityModeHash":77,"currentPlaylistActivityHash":88`, lakeOfShadowsHash)), CurrentActivityReady, "Lake of Shadows", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := currentActivityServer(t, tt.body, nil)
			defer srv.Close()

			svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), nil, currentActivityDefinitions(), cache.NewNoOpCache(), time.Minute)
			current, err := svc.GetCurrentActivity(context.Background(), 3, "membership", currentActivityCharacterID, "token")
			if err != nil {
				t.Fatalf("GetCurrentActivity: %v", err)
			}
			if current.State != tt.wantState || current.ActivityName != tt.wantActivity ||
				current.ModeName != tt.wantMode || current.PlaylistName != tt.wantPlaylist {
				t.Fatalf("current activity = %+v, want state %q activity %q mode %q playlist %q",
					current, tt.wantState, tt.wantActivity, tt.wantMode, tt.wantPlaylist)
			}
		})
	}
}

func TestGetCurrentActivity_IdleSkipsManifest(t *testing.T) {
	srv := currentActivityServer(t, activityEntry(`"currentActivityHash":0`), nil)
	defer srv.Close()

	svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), nil, &fakeActivityDefinitions{err: errors.New("manifest not ready")}, cache.NewNoOpCache(), time.Minute)
	current, err := svc.GetCurrentActivity(context.Background(), 3, "membership", currentActivityCharacterID, "token")
	if err != nil {
		t.Fatalf("GetCurrentActivity: %v", err)
	}
	if current.State != CurrentActivityIdle {
		t.Fatalf("State = %q, want idle", current.State)
	}
}

func TestGetCurrentActivity_RejectsCharacterBeforeActivityRequest(t *testing.T) {
	var requests int
	srv := currentActivityServer(t, activityEntry(`"currentActivityHash":0`), &requests)
	defer srv.Close()

	svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), nil, currentActivityDefinitions(), cache.NewNoOpCache(), time.Minute)
	_, err := svc.GetCurrentActivity(context.Background(), 3, "membership", "2305843009000000000", "token")
	if !errors.Is(err, ErrCharacterNotFound) {
		t.Fatalf("GetCurrentActivity error = %v, want ErrCharacterNotFound", err)
	}
	if requests != 0 {
		t.Fatal("component-204 request crossed Bungie seam before character validation")
	}
}

func TestGetCurrentActivity_PropagatesFailures(t *testing.T) {
	t.Run("Bungie request", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("components") == "200" {
				fmt.Fprintf(w, `{"ErrorCode":1,"Response":{"characters":{"data":{"%s":{"characterId":"%s"}}}}}`, currentActivityCharacterID, currentActivityCharacterID)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"ErrorCode":5,"ErrorStatus":"SystemDisabled","Message":"maintenance"}`)
		}))
		defer srv.Close()

		svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), nil, currentActivityDefinitions(), cache.NewNoOpCache(), time.Minute)
		_, err := svc.GetCurrentActivity(context.Background(), 3, "membership", currentActivityCharacterID, "token")
		if err == nil || !strings.Contains(err.Error(), "current activity") {
			t.Fatalf("GetCurrentActivity error = %v, want wrapped Bungie failure", err)
		}
	})
	t.Run("Manifest lookup", func(t *testing.T) {
		srv := currentActivityServer(t, activityEntry(fmt.Sprintf(`"currentActivityHash":%d`, lakeOfShadowsHash)), nil)
		defer srv.Close()

		want := errors.New("manifest unavailable")
		svc := NewService(bungie.NewClient("k", srv.URL, 100, 100), nil, &fakeActivityDefinitions{err: want}, cache.NewNoOpCache(), time.Minute)
		_, err := svc.GetCurrentActivity(context.Background(), 3, "membership", currentActivityCharacterID, "token")
		if !errors.Is(err, want) {
			t.Fatalf("GetCurrentActivity error = %v, want wrapped manifest error", err)
		}
	})
}
