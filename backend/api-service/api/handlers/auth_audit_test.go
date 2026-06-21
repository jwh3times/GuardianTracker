package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"guardian-tracker/api-service/db"

	"github.com/gin-gonic/gin"
)

// spyAudit records events for assertions and can simulate a write failure.
type spyAudit struct {
	mu     sync.Mutex
	events []db.AuditEvent
	err    error
}

func (s *spyAudit) Log(_ context.Context, ev db.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return s.err
}

func (s *spyAudit) types() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.events))
	for i, e := range s.events {
		out[i] = e.EventType
	}
	return out
}

func TestRefreshToken_InvalidToken_AuditsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyAudit{}
	// jwt/tokenStore/cfg are unused on the invalid-token early-return path; a
	// minimal handler with only the audit logger set exercises it.
	h := &AuthHandler{audit: spy}

	r := gin.New()
	r.POST("/api/auth/refresh", h.RefreshToken)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh",
		strings.NewReader(`{"refreshToken":"not-a-jwt"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if got := spy.types(); len(got) != 1 || got[0] != "refresh.failure" {
		t.Errorf("audit events = %v, want [refresh.failure]", got)
	}
}

func TestAudit_BestEffort_DoesNotBlockResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	spy := &spyAudit{err: context.DeadlineExceeded} // audit write fails
	h := &AuthHandler{audit: spy}

	r := gin.New()
	r.POST("/api/auth/refresh", h.RefreshToken)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh",
		strings.NewReader(`{"refreshToken":"not-a-jwt"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Audit failure must not change the HTTP outcome.
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 despite audit error", w.Code)
	}
}
