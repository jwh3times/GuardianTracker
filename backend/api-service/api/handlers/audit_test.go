package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

type mockAuditRead struct {
	gotFilter db.AuditFilter
	entries   []db.AuditEntry
	cursor    string
}

func (m *mockAuditRead) List(_ context.Context, f db.AuditFilter) ([]db.AuditEntry, string, error) {
	m.gotFilter = f
	return m.entries, m.cursor, nil
}

func TestListAudit_ReturnsEntriesAndPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &mockAuditRead{
		entries: []db.AuditEntry{{
			ID: 7, EventType: "role.change.admin", Outcome: "success",
			ActorMembershipID: "a", ActorDisplayName: "Admin",
			TargetMembershipID: "b", TargetDisplayName: "Guardian",
			Details: map[string]any{"oldRole": 0, "newRole": 1}, CreatedAt: time.Now(),
		}},
		cursor: "next-cursor",
	}
	h := NewAuditHandler(store)
	r := gin.New()
	r.GET("/api/admin/audit", h.ListAudit)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit?type=role.&outcome=success&limit=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Entries []struct {
			ID        string `json:"id"`
			EventType string `json:"eventType"`
		} `json:"entries"`
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Entries) != 1 || resp.Entries[0].ID != "7" || resp.Entries[0].EventType != "role.change.admin" {
		t.Errorf("entries = %+v, want one role.change.admin id=7", resp.Entries)
	}
	if resp.NextCursor != "next-cursor" {
		t.Errorf("nextCursor = %q, want next-cursor", resp.NextCursor)
	}
	// Query params propagated to the filter.
	if store.gotFilter.EventType != "role." || store.gotFilter.Outcome != "success" || store.gotFilter.Limit != 10 {
		t.Errorf("filter = %+v, want type=role. outcome=success limit=10", store.gotFilter)
	}
}

func TestListAudit_DegradedMode503(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Degraded mode is the adapter set, not a nil store — build it the way
	// production does so the test exercises the real path.
	h := NewAuditHandler(db.NewStores(nil).Audit)
	r := gin.New()
	r.GET("/api/admin/audit", h.ListAudit)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}
