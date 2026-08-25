package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/characters"
	"guardian-tracker/api-service/services/collections"
	"guardian-tracker/api-service/services/records"
	"guardian-tracker/api-service/services/search"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

const testUserID = "4611686018467260757"

func init() { gin.SetMode(gin.TestMode) }

// authedRouter wires a single route behind a middleware that stands in for the
// JWT middleware by injecting membership_id / membership_type into the context.
func authedRouter(method, path, ctxMembershipID string, h gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Handle(method, path, func(c *gin.Context) {
		c.Set("membership_id", ctxMembershipID)
		c.Set("membership_type", 3)
		h(c)
	})
	return r
}

func do(r *gin.Engine, method, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, target, nil))
	return w
}

func newTokenStore(t *testing.T) *auth.TokenStore {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return auth.NewTokenStore(ctx, "cid", "", nil, nil)
}

func storeValidToken(ts *auth.TokenStore, id string) {
	ts.Store(id, &auth.BungieTokens{
		AccessToken:           "valid-token",
		RefreshToken:          "r",
		AccessTokenExpiresAt:  time.Now().Add(time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
	})
}

// ---------------- Health ----------------

func TestHealthHandlers(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	ms := bungie.NewManifestService(bungie.NewClient("k", "http://x", 100, 100), dbPath, time.Hour)
	h := NewHealthHandler(ms, nil)

	r := gin.New()
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)
	r.GET("/manifest", h.ManifestStatus)

	if w := do(r, http.MethodGet, "/health"); w.Code != http.StatusOK {
		t.Errorf("health = %d", w.Code)
	}
	// Not ready until the db file exists.
	if w := do(r, http.MethodGet, "/ready"); w.Code != http.StatusServiceUnavailable {
		t.Errorf("ready (no db) = %d, want 503", w.Code)
	}
	if err := os.WriteFile(dbPath, []byte("db"), 0644); err != nil {
		t.Fatal(err)
	}
	if w := do(r, http.MethodGet, "/ready"); w.Code != http.StatusOK {
		t.Errorf("ready (db present) = %d, want 200", w.Code)
	}
	if w := do(r, http.MethodGet, "/manifest"); w.Code != http.StatusOK {
		t.Errorf("manifest status = %d", w.Code)
	}
}

// ---------------- Search ----------------

func TestSearchHandler(t *testing.T) {
	h := NewSearchHandler(search.NewService(nil, "")) // index never built → not ready
	r := gin.New()
	r.GET("/api/items/search", h.Search)

	// Query under 2 chars short-circuits to an empty 200.
	if w := do(r, http.MethodGet, "/api/items/search?q=a"); w.Code != http.StatusOK {
		t.Errorf("short query = %d, want 200", w.Code)
	}
	// Valid query but index not ready → 503.
	if w := do(r, http.MethodGet, "/api/items/search?q=gjallarhorn&limit=99"); w.Code != http.StatusServiceUnavailable {
		t.Errorf("not-ready search = %d, want 503", w.Code)
	}
}

// searchManifestDB writes a minimal manifest SQLite file plus its sibling
// version file, and skips when the sqlite3 driver is a CGO-disabled stub.
func searchManifestDB(t *testing.T) string {
	t.Helper()
	probe, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("sqlite3 driver unavailable: %v", err)
	}
	defer probe.Close()
	if err := probe.Ping(); err != nil {
		t.Skipf("sqlite3 driver unavailable (CGO disabled?): %v", err)
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "manifest.sqlite")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE DestinyInventoryItemDefinition (id INTEGER PRIMARY KEY, json TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	blob := `{"hash":1,"displayProperties":{"name":"Gjallarhorn"},"itemType":3,"itemSubType":10,"inventory":{"tierType":6}}`
	if _, err := db.Exec(`INSERT INTO DestinyInventoryItemDefinition (id, json) VALUES (0, ?)`, blob); err != nil {
		t.Fatalf("insert: %v", err)
	}
	db.Close()

	if err := os.WriteFile(filepath.Join(dir, "manifest_version.txt"), []byte("v-test"), 0644); err != nil {
		t.Fatalf("write version: %v", err)
	}
	return dbPath
}

// The regression for the cold-start retry gap: the handler returned 503 on
// !IsReady *before* reaching Search, the only caller of the index's retry path.
// An index that was never built (or whose first build failed) therefore stayed
// dead until the next hourly manifest swap. Serving a not-ready request must
// itself kick the rebuild.
func TestSearchHandler_NotReadyKicksIndexBuild(t *testing.T) {
	dbPath := searchManifestDB(t)
	ms := bungie.NewManifestService(bungie.NewClient("k", "http://unused", 100, 100), dbPath, time.Hour)
	svc := search.NewService(ms, dbPath)
	if svc.IsReady() {
		t.Fatal("index should start unbuilt — nothing has called BuildIndex")
	}

	r := gin.New()
	r.GET("/api/items/search", NewSearchHandler(svc).Search)

	// The build is asynchronous and this fixture indexes a single item, so the
	// goroutine can finish before the handler re-checks IsReady. Both outcomes
	// are correct: 503 while the build is still running, 200 once it has landed.
	// Asserting 503 here pins one side of a race the handler never promised —
	// the deterministic not-ready 503 is covered by TestSearchHandler, which
	// uses a service that can never become ready. What this test owns is the
	// regression below: a not-ready search must itself kick the build.
	if w := do(r, http.MethodGet, "/api/items/search?q=gjallarhorn"); w.Code != http.StatusServiceUnavailable && w.Code != http.StatusOK {
		t.Fatalf("first search = %d, want 503 or 200", w.Code)
	}

	// … but it must have started the build the 503 path used to skip.
	deadline := time.Now().Add(5 * time.Second)
	for !svc.IsReady() {
		if time.Now().After(deadline) {
			t.Fatal("a not-ready search never triggered an index build")
		}
		time.Sleep(time.Millisecond)
	}

	if w := do(r, http.MethodGet, "/api/items/search?q=gjallarhorn"); w.Code != http.StatusOK {
		t.Errorf("search after recovery = %d, want 200", w.Code)
	}
}

// ---------------- Characters (full gating path) ----------------

func charactersHandler(t *testing.T, bungieURL string, ts *auth.TokenStore) *CharactersHandler {
	t.Helper()
	svc := characters.NewService(bungie.NewClient("k", bungieURL, 100, 100), cache.NewMemoryCache(time.Minute, 0), time.Minute)
	return NewCharactersHandler(svc, ts)
}

func TestCharacters_BadParams(t *testing.T) {
	h := charactersHandler(t, "http://x", newTokenStore(t))
	r := authedRouter(http.MethodGet, "/api/characters/:membershipType/:membershipId", testUserID, h.GetCharacters)
	if w := do(r, http.MethodGet, "/api/characters/99/"+testUserID); w.Code != http.StatusBadRequest {
		t.Errorf("bad type = %d, want 400", w.Code)
	}
}

func TestCharacters_OwnershipMismatch(t *testing.T) {
	h := charactersHandler(t, "http://x", newTokenStore(t))
	// Context identity differs from the requested membership id → 403.
	r := authedRouter(http.MethodGet, "/api/characters/:membershipType/:membershipId", "9999999999", h.GetCharacters)
	if w := do(r, http.MethodGet, "/api/characters/3/"+testUserID); w.Code != http.StatusForbidden {
		t.Errorf("ownership mismatch = %d, want 403", w.Code)
	}
}

func TestCharacters_NoToken(t *testing.T) {
	h := charactersHandler(t, "http://x", newTokenStore(t)) // nothing stored
	r := authedRouter(http.MethodGet, "/api/characters/:membershipType/:membershipId", testUserID, h.GetCharacters)
	if w := do(r, http.MethodGet, "/api/characters/3/"+testUserID); w.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d, want 401", w.Code)
	} else if !strings.Contains(w.Body.String(), "BUNGIE_REAUTH_REQUIRED") {
		t.Errorf("no token body = %s, want BUNGIE_REAUTH_REQUIRED", w.Body.String())
	}
}

func TestCharacters_ExpiredRefreshToken(t *testing.T) {
	ts := newTokenStore(t)
	ts.Store(testUserID, &auth.BungieTokens{
		AccessToken:           "stale",
		AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(-time.Hour), // refresh expired → re-auth required
	})
	h := charactersHandler(t, "http://x", ts)
	r := authedRouter(http.MethodGet, "/api/characters/:membershipType/:membershipId", testUserID, h.GetCharacters)
	if w := do(r, http.MethodGet, "/api/characters/3/"+testUserID); w.Code != http.StatusUnauthorized {
		t.Errorf("expired refresh = %d, want 401", w.Code)
	} else if !strings.Contains(w.Body.String(), "BUNGIE_REAUTH_REQUIRED") {
		t.Errorf("expired refresh body = %s, want BUNGIE_REAUTH_REQUIRED", w.Body.String())
	}
}

func TestCharacters_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":1,"Response":{"characters":{"data":{
			"c1":{"characterId":"c1","classType":1,"raceType":1,"light":2010,"dateLastPlayed":"2026-06-10T00:00:00Z"}}}}}`)
	}))
	defer srv.Close()

	ts := newTokenStore(t)
	storeValidToken(ts, testUserID)
	h := charactersHandler(t, srv.URL, ts)
	r := authedRouter(http.MethodGet, "/api/characters/:membershipType/:membershipId", testUserID, h.GetCharacters)

	w := do(r, http.MethodGet, "/api/characters/3/"+testUserID)
	if w.Code != http.StatusOK {
		t.Fatalf("success = %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Hunter") {
		t.Errorf("body missing mapped class: %s", w.Body.String())
	}
}

func TestCharacters_BungieError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ErrorCode":5,"ErrorStatus":"private"}`) // PRIVACY
	}))
	defer srv.Close()

	ts := newTokenStore(t)
	storeValidToken(ts, testUserID)
	h := charactersHandler(t, srv.URL, ts)
	r := authedRouter(http.MethodGet, "/api/characters/:membershipType/:membershipId", testUserID, h.GetCharacters)
	if w := do(r, http.MethodGet, "/api/characters/3/"+testUserID); w.Code != http.StatusForbidden {
		t.Errorf("privacy error = %d, want 403", w.Code)
	}
}

// ---------------- Records (nil manifest → 503 MANIFEST_NOT_READY) ----------------

func recordsHandler(t *testing.T, ts *auth.TokenStore) *RecordsHandler {
	t.Helper()
	svc := records.NewService(bungie.NewClient("k", "http://x", 100, 100), nil, cache.NewMemoryCache(time.Minute, 0), time.Minute)
	return NewRecordsHandler(svc, ts)
}

func TestRecords_AllEndpoints_ManifestNotReady(t *testing.T) {
	ts := newTokenStore(t)
	storeValidToken(ts, testUserID)
	h := recordsHandler(t, ts)

	endpoints := []struct {
		name string
		fn   gin.HandlerFunc
	}{
		{"catalysts", h.GetCatalysts},
		{"crafting", h.GetCrafting},
		{"seals", h.GetSeals},
	}
	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			r := authedRouter(http.MethodGet, "/api/x/:membershipType/:membershipId", testUserID, ep.fn)
			w := do(r, http.MethodGet, "/api/x/3/"+testUserID)
			if w.Code != http.StatusServiceUnavailable {
				t.Errorf("%s = %d, want 503 MANIFEST_NOT_READY", ep.name, w.Code)
			}
			if !strings.Contains(w.Body.String(), "MANIFEST_NOT_READY") {
				t.Errorf("%s body = %s", ep.name, w.Body.String())
			}
		})
	}
}

func TestRecords_BadParams(t *testing.T) {
	h := recordsHandler(t, newTokenStore(t))
	r := authedRouter(http.MethodGet, "/api/x/:membershipType/:membershipId", testUserID, h.GetSeals)
	if w := do(r, http.MethodGet, "/api/x/0/"+testUserID); w.Code != http.StatusBadRequest {
		t.Errorf("bad params = %d, want 400", w.Code)
	}
}

// ---------------- Collections (gating + refresh) ----------------

func collectionsHandler(t *testing.T, ts *auth.TokenStore) *CollectionsHandler {
	t.Helper()
	c := cache.NewMemoryCache(time.Minute, 0)
	collSvc := collections.NewService(bungie.NewClient("k", "http://x", 100, 100), nil, nil, c, time.Minute)
	charSvc := characters.NewService(bungie.NewClient("k", "http://x", 100, 100), c, time.Minute)
	recSvc := records.NewService(bungie.NewClient("k", "http://x", 100, 100), nil, c, time.Minute)
	return NewCollectionsHandler(collSvc, charSvc, recSvc, ts, nil)
}

func TestCollections_OwnershipMismatch(t *testing.T) {
	h := collectionsHandler(t, newTokenStore(t))
	r := authedRouter(http.MethodGet, "/api/collections/:membershipType/:membershipId", "9999999999", h.GetCollections)
	if w := do(r, http.MethodGet, "/api/collections/3/"+testUserID); w.Code != http.StatusForbidden {
		t.Errorf("ownership = %d, want 403", w.Code)
	}
}

func TestCollections_Refresh(t *testing.T) {
	h := collectionsHandler(t, newTokenStore(t))
	r := authedRouter(http.MethodPost, "/api/collections/:membershipType/:membershipId/refresh", testUserID, h.RefreshCollections)
	w := do(r, http.MethodPost, "/api/collections/3/"+testUserID+"/refresh")
	if w.Code != http.StatusOK {
		t.Errorf("refresh = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "success") {
		t.Errorf("refresh body = %s", w.Body.String())
	}
}

func TestCollections_RefreshBadParams(t *testing.T) {
	h := collectionsHandler(t, newTokenStore(t))
	r := authedRouter(http.MethodPost, "/api/collections/:membershipType/:membershipId/refresh", testUserID, h.RefreshCollections)
	if w := do(r, http.MethodPost, "/api/collections/99/"+testUserID+"/refresh"); w.Code != http.StatusBadRequest {
		t.Errorf("refresh bad params = %d, want 400", w.Code)
	}
}
