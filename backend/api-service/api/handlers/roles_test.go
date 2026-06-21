package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// --- fakes (satisfy the account + admin store interfaces) ---

type fakeRoleStore struct {
	setRoleErr    error
	lastSetRole   int16
	setRoleCalled bool

	users      []db.AdminUser
	change     *db.RoleChange
	setByIDErr error
	lastActor  string
	lastTarget int64
	lastNew    int16
}

func (f *fakeRoleStore) SetRole(_ context.Context, _ string, role int16) error {
	f.setRoleCalled = true
	f.lastSetRole = role
	return f.setRoleErr
}
func (f *fakeRoleStore) ListUsers(_ context.Context, _ string, _ int) ([]db.AdminUser, error) {
	return f.users, nil
}
func (f *fakeRoleStore) SetRoleByID(_ context.Context, actor string, target int64, newRole int16) (*db.RoleChange, error) {
	f.lastActor, f.lastTarget, f.lastNew = actor, target, newRole
	if f.setByIDErr != nil {
		return nil, f.setByIDErr
	}
	return f.change, nil
}

type fakeFlagStore struct {
	list      []db.FeatureFlag
	updated   *db.FeatureFlag
	updateErr error
	lastKey   string
}

func (f *fakeFlagStore) List(_ context.Context) ([]db.FeatureFlag, error) { return f.list, nil }
func (f *fakeFlagStore) Update(_ context.Context, key string, _ *bool, _ *int16, _ *int64, _ string) (*db.FeatureFlag, error) {
	f.lastKey = key
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.updated, nil
}

func roleTestRouter(membershipID string, role int, register func(*gin.Engine)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("membership_id", membershipID)
		c.Set("role", role)
		c.Next()
	})
	register(r)
	return r
}

func newCache() cache.Cache { return cache.NewMemoryCache(time.Minute, time.Minute) }

// --- account: self opt-in ---

func TestSetRole_OptInAllowedEvictsCache(t *testing.T) {
	store := &fakeRoleStore{}
	c := newCache()
	c.Set("tver:member-1", 1, time.Minute)
	h := NewAccountHandler(store, nil, c, nil)
	r := roleTestRouter("member-1", auth.RoleStandard, func(e *gin.Engine) {
		e.PUT("/api/account/role", h.SetRole)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/account/role", strings.NewReader(`{"role":"beta"}`)))

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", w.Code, w.Body.String())
	}
	if !store.setRoleCalled || store.lastSetRole != int16(auth.RoleBeta) {
		t.Errorf("SetRole called=%v role=%d, want true/1", store.setRoleCalled, store.lastSetRole)
	}
	if _, ok := c.Get("tver:member-1"); ok {
		t.Error("expected revocation cache entry to be evicted after opt-in")
	}
}

func TestSetRole_RejectsAdminTarget(t *testing.T) {
	store := &fakeRoleStore{}
	h := NewAccountHandler(store, nil, newCache(), nil)
	r := roleTestRouter("member-1", auth.RoleStandard, func(e *gin.Engine) {
		e.PUT("/api/account/role", h.SetRole)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/account/role", strings.NewReader(`{"role":"admin"}`)))
	if w.Code != http.StatusForbidden {
		t.Fatalf("opt-in to admin: got %d, want 403", w.Code)
	}
	if store.setRoleCalled {
		t.Error("SetRole must not be called for an admin opt-in attempt")
	}
}

func TestSetRole_AdminCallerRefused(t *testing.T) {
	store := &fakeRoleStore{}
	h := NewAccountHandler(store, nil, newCache(), nil)
	r := roleTestRouter("member-1", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/account/role", h.SetRole)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/account/role", strings.NewReader(`{"role":"beta"}`)))
	if w.Code != http.StatusForbidden {
		t.Fatalf("admin self opt-in: got %d, want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ADMIN_OPT_IN") {
		t.Errorf("expected ADMIN_OPT_IN code, got %s", w.Body.String())
	}
}

func TestSetRole_DegradedMode503(t *testing.T) {
	h := NewAccountHandler(nil, nil, newCache(), nil)
	r := roleTestRouter("member-1", auth.RoleStandard, func(e *gin.Engine) {
		e.PUT("/api/account/role", h.SetRole)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/account/role", strings.NewReader(`{"role":"beta"}`)))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("degraded opt-in: got %d, want 503", w.Code)
	}
}

// --- account: flag evaluation truth table ---

func TestGetFlags_TruthTable(t *testing.T) {
	flags := []db.FeatureFlag{
		{Key: "shipped", MinTier: 0, Enabled: true},
		{Key: "beta-feat", MinTier: 1, Enabled: true},
		{Key: "dev-feat", MinTier: 2, Enabled: false},
	}
	type want struct{ enabled, accessible, locked bool }
	cases := []struct {
		role int
		exp  map[string]want
	}{
		{auth.RoleStandard, map[string]want{
			"shipped":   {true, true, false},
			"beta-feat": {true, false, true}, // enabled but tier too low -> locked
			"dev-feat":  {false, false, false},
		}},
		{auth.RoleAlpha, map[string]want{
			"shipped":   {true, true, false},
			"beta-feat": {true, true, false},
			"dev-feat":  {false, false, false},
		}},
	}
	for _, tc := range cases {
		h := NewAccountHandler(nil, &fakeFlagStore{list: flags}, newCache(), nil)
		r := roleTestRouter("member-1", tc.role, func(e *gin.Engine) { e.GET("/api/flags", h.GetFlags) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/flags", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("role=%d: got %d", tc.role, w.Code)
		}
		var resp struct {
			Role  string `json:"role"`
			Flags []struct {
				Key        string `json:"key"`
				Enabled    bool   `json:"enabled"`
				Accessible bool   `json:"accessible"`
				Locked     bool   `json:"locked"`
			} `json:"flags"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		got := map[string]want{}
		for _, f := range resp.Flags {
			got[f.Key] = want{f.Enabled, f.Accessible, f.Locked}
		}
		for k, exp := range tc.exp {
			if got[k] != exp {
				t.Errorf("role=%d flag=%s: got %+v, want %+v", tc.role, k, got[k], exp)
			}
		}
	}
}

// --- admin: role change ---

func TestAdminSetUserRole_SuccessEvictsTargetCache(t *testing.T) {
	store := &fakeRoleStore{change: &db.RoleChange{
		TargetUserID: 5, TargetMembershipID: "target-m", OldRole: 1, NewRole: 2,
	}}
	c := newCache()
	c.Set("tver:target-m", 3, time.Minute)
	h := NewAdminHandler(store, &fakeFlagStore{}, c)
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/admin/users/:id/role", h.SetUserRole)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/admin/users/5/role", strings.NewReader(`{"role":"alpha"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", w.Code, w.Body.String())
	}
	if store.lastTarget != 5 || store.lastNew != 2 || store.lastActor != "admin-m" {
		t.Errorf("SetRoleByID got target=%d new=%d actor=%s", store.lastTarget, store.lastNew, store.lastActor)
	}
	if _, ok := c.Get("tver:target-m"); ok {
		t.Error("expected target revocation cache entry evicted after admin role change")
	}
	if !strings.Contains(w.Body.String(), `"previousRole":"beta"`) {
		t.Errorf("expected previousRole=beta, got %s", w.Body.String())
	}
}

func TestAdminSetUserRole_LastAdmin409(t *testing.T) {
	store := &fakeRoleStore{setByIDErr: db.ErrLastAdmin}
	h := NewAdminHandler(store, &fakeFlagStore{}, newCache())
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/admin/users/:id/role", h.SetUserRole)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/admin/users/9/role", strings.NewReader(`{"role":"standard"}`)))
	if w.Code != http.StatusConflict {
		t.Fatalf("last admin: got %d, want 409", w.Code)
	}
	if !strings.Contains(w.Body.String(), "LAST_ADMIN") {
		t.Errorf("expected LAST_ADMIN code, got %s", w.Body.String())
	}
}

func TestAdminSetUserRole_NotFound404(t *testing.T) {
	store := &fakeRoleStore{setByIDErr: db.ErrUserNotFound}
	h := NewAdminHandler(store, &fakeFlagStore{}, newCache())
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/admin/users/:id/role", h.SetUserRole)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/admin/users/9/role", strings.NewReader(`{"role":"beta"}`)))
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown target: got %d, want 404", w.Code)
	}
}

func TestAdminSetUserRole_InvalidRole400(t *testing.T) {
	store := &fakeRoleStore{}
	h := NewAdminHandler(store, &fakeFlagStore{}, newCache())
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/admin/users/:id/role", h.SetUserRole)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/admin/users/9/role", strings.NewReader(`{"role":"wizard"}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid role: got %d, want 400", w.Code)
	}
}

func TestAdminListUsers_MapsFields(t *testing.T) {
	store := &fakeRoleStore{users: []db.AdminUser{
		{ID: 1, MembershipID: "m1", MembershipType: 3, DisplayName: "Maya", Role: 3, LastLoginAt: time.Now()},
	}}
	h := NewAdminHandler(store, &fakeFlagStore{}, newCache())
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) { e.GET("/api/admin/users", h.ListUsers) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/users", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{`"role":"admin"`, `"platform":"steam"`, `"displayName":"Maya"`} {
		if !strings.Contains(body, want) {
			t.Errorf("response %s missing %q", body, want)
		}
	}
}

// --- admin: flag update ---

func TestAdminUpdateFlag_SuccessEvictsFlagCache(t *testing.T) {
	store := &fakeFlagStore{updated: &db.FeatureFlag{Key: "god-roll", Name: "God-roll", MinTier: 2, Enabled: true, UpdatedAt: time.Now()}}
	c := newCache()
	c.Set(flagsCacheKey, []db.FeatureFlag{{Key: "god-roll"}}, time.Minute)
	h := NewAdminHandler(&fakeRoleStore{}, store, c)
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/admin/flags/:key", h.UpdateFlag)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/admin/flags/god-roll", strings.NewReader(`{"enabled":true}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", w.Code, w.Body.String())
	}
	if store.lastKey != "god-roll" {
		t.Errorf("Update key = %q, want god-roll", store.lastKey)
	}
	if _, ok := c.Get(flagsCacheKey); ok {
		t.Error("expected flag list cache evicted after flag update")
	}
}

func TestAdminUpdateFlag_RejectsAdminMinTier(t *testing.T) {
	h := NewAdminHandler(&fakeRoleStore{}, &fakeFlagStore{}, newCache())
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/admin/flags/:key", h.UpdateFlag)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/admin/flags/x", strings.NewReader(`{"minTier":"admin"}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("minTier=admin: got %d, want 400", w.Code)
	}
}

func TestAdminUpdateFlag_Unknown404(t *testing.T) {
	store := &fakeFlagStore{updateErr: pgx.ErrNoRows}
	h := NewAdminHandler(&fakeRoleStore{}, store, newCache())
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/admin/flags/:key", h.UpdateFlag)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/admin/flags/nope", strings.NewReader(`{"enabled":true}`)))
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown flag: got %d, want 404", w.Code)
	}
}

func TestAdminUpdateFlag_NothingToUpdate400(t *testing.T) {
	h := NewAdminHandler(&fakeRoleStore{}, &fakeFlagStore{}, newCache())
	r := roleTestRouter("admin-m", auth.RoleAdmin, func(e *gin.Engine) {
		e.PUT("/api/admin/flags/:key", h.UpdateFlag)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/admin/flags/x", strings.NewReader(`{}`)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty patch: got %d, want 400", w.Code)
	}
}
