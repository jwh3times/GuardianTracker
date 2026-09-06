package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"guardian-tracker/api-service/auth"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	h, _ := newAuthHandler(t)
	h.audit = spy

	r := gin.New()
	r.POST("/api/auth/refresh", h.RefreshToken)
	req := newRefreshRequest("/api/auth/refresh", "not-a-jwt")
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
	h, _ := newAuthHandler(t)
	h.audit = spy

	r := gin.New()
	r.POST("/api/auth/refresh", h.RefreshToken)
	req := newRefreshRequest("/api/auth/refresh", "not-a-jwt")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Audit failure must not change the HTTP outcome.
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 despite audit error", w.Code)
	}
}

func TestRefreshToken_SuccessAuditUsesIssuedSession(t *testing.T) {
	for _, existingSession := range []bool{true, false} {
		for _, auditFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("existing_session=%t/audit_fails=%t", existingSession, auditFails), func(t *testing.T) {
				spy := &spyAudit{}
				if auditFails {
					spy.err = context.DeadlineExceeded
				}
				store := newFakeUserStore()
				handler, jwt := newAuthHandlerWith(t, store, spy)
				profile := &auth.DestinyMembership{MembershipID: testUserID, DisplayName: "TestGuardian", MembershipType: 3}
				sessionID := ""
				if existingSession {
					sessionID = "fixture-session"
				}
				oldRefresh, jti, err := jwt.GenerateRefreshToken(profile, 1, sessionID)
				if err != nil {
					t.Fatal(err)
				}
				if existingSession {
					store.sessions[sessionID] = jti
				}
				router := gin.New()
				router.POST("/api/auth/refresh", handler.RefreshToken)
				request := newRefreshRequest("/api/auth/refresh", oldRefresh)
				request.RemoteAddr = "127.0.0.1:1234"
				request.Header.Set("User-Agent", "fixture-agent")
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, request)
				if recorder.Code != http.StatusOK {
					t.Fatalf("refresh=%d, want200", recorder.Code)
				}
				cookie := findRefreshCookie(t, recorder)
				if cookie.Value == oldRefresh || !cookie.HttpOnly || cookie.MaxAge <= 0 {
					t.Fatal("successful refresh did not rotate its HttpOnly cookie")
				}
				refreshClaims, err := jwt.ValidateToken(cookie.Value)
				if err != nil {
					t.Fatal(err)
				}
				var body struct {
					Token string `json:"token"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				accessClaims, err := jwt.ValidateToken(body.Token)
				if err != nil {
					t.Fatal(err)
				}
				if refreshClaims.TokenType != "refresh" || accessClaims.TokenType != "access" || refreshClaims.SessionID == "" || accessClaims.SessionID != refreshClaims.SessionID {
					t.Fatal("response did not contain a valid refreshed JWT pair")
				}
				if existingSession && refreshClaims.SessionID != sessionID {
					t.Fatal("refresh replaced existing session ID")
				}
				// Exact shape forbids credential-bearing extras and verifies the adopted
				// session's new ID comes from the issuer result, not the old request JWT.
				expected := []db.AuditEvent{{
					EventType: "refresh.success", Outcome: "success", ActorMembershipID: testUserID,
					SessionID: refreshClaims.SessionID, IP: "127.0.0.1", UserAgent: "fixture-agent",
				}}
				if !reflect.DeepEqual(spy.events, expected) {
					t.Fatalf("refresh audit=%+v, want=%+v", spy.events, expected)
				}
			})
		}
	}
}

func TestRefreshToken_ReuseAddsNoSuccessAudit(t *testing.T) {
	spy := &spyAudit{}
	store := newFakeUserStore()
	handler, jwt := newAuthHandlerWith(t, store, spy)
	profile := &auth.DestinyMembership{MembershipID: testUserID, DisplayName: "TestGuardian", MembershipType: 3}
	token, jti, err := jwt.GenerateRefreshToken(profile, 1, "fixture-session")
	if err != nil {
		t.Fatal(err)
	}
	store.sessions["fixture-session"] = jti
	router := gin.New()
	router.POST("/api/auth/refresh", handler.RefreshToken)
	for _, want := range []int{http.StatusOK, http.StatusUnauthorized} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, newRefreshRequest("/api/auth/refresh", token))
		if recorder.Code != want {
			t.Fatalf("refresh=%d want=%d", recorder.Code, want)
		}
	}
	if got := spy.types(); !reflect.DeepEqual(got, []string{"refresh.success", "refresh.reuse"}) {
		t.Fatalf("audit events=%v", got)
	}
}
