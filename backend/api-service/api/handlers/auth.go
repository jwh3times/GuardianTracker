package handlers

import (
	"context"
	"net/http"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/config"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/observability"

	"github.com/gin-gonic/gin"
)

// AuditLogger records audit events best-effort. Satisfied by *db.AuditStore.
type AuditLogger interface {
	Log(ctx context.Context, ev db.AuditEvent) error
}

const (
	refreshCookieName = "guardian_refresh_token"
	refreshCookiePath = "/api/auth"
)

// AuthHandler is the HTTP surface of the session lifecycle. It owns status
// codes, the refresh cookie, audit events, and response bodies; what a session
// is and how one is issued belongs to auth.SessionIssuer.
type AuthHandler struct {
	issuer *auth.SessionIssuer
	cfg    *config.Config
	audit  AuditLogger
}

func NewAuthHandler(issuer *auth.SessionIssuer, cfg *config.Config, audit AuditLogger) *AuthHandler {
	return &AuthHandler{issuer: issuer, cfg: cfg, audit: audit}
}

// authFailure is how one issuance failure is reported over HTTP: the status,
// the message the browser shows, the audit event it records, and whether the
// refresh cookie survives it.
type authFailure struct {
	status       int
	message      string
	event        string // audit event type; empty records nothing
	expireCookie bool
}

// loginFailures and refreshFailures map every auth.Reason the two endpoints can
// produce. They are separate tables because the two endpoints genuinely differ:
// the same failed session write is "Failed to create session" on login and
// "Failed to refresh session" on refresh, and only refresh clears the cookie.
//
// A reason missing from its table falls through to a 500 rather than an empty
// success — an unmapped failure must not look like a working login.
var loginFailures = map[auth.Reason]authFailure{
	auth.ReasonInvalidCode:  {http.StatusBadRequest, "Invalid authorization code", "login.failure", false},
	auth.ReasonInvalidState: {http.StatusBadRequest, "Invalid or expired state. Please try logging in again.", "login.failure", false},
	auth.ReasonCodeExchange: {http.StatusInternalServerError, "Failed to complete authentication", "login.failure", false},
	auth.ReasonProfileFetch: {http.StatusInternalServerError, "Failed to retrieve user profile", "login.failure", false},
	auth.ReasonTokenMint:    {http.StatusInternalServerError, "Failed to create session", "", false},
	auth.ReasonSessionWrite: {http.StatusInternalServerError, "Failed to create session", "", false},
}

var refreshFailures = map[auth.Reason]authFailure{
	auth.ReasonMissingCookie: {http.StatusUnauthorized, "Invalid or expired refresh token", "refresh.failure", true},
	auth.ReasonInvalidToken:  {http.StatusUnauthorized, "Invalid or expired refresh token", "refresh.failure", true},
	auth.ReasonRevoked:       {http.StatusUnauthorized, "Session has been revoked. Please log in again.", "refresh.failure", true},
	auth.ReasonReuse:         {http.StatusUnauthorized, "Session ended for security reasons. Please log in again.", "refresh.reuse", true},
	auth.ReasonExpired:       {http.StatusUnauthorized, "Session has expired. Please log in again.", "refresh.failure", true},
	auth.ReasonTokenMint:     {http.StatusInternalServerError, "Failed to refresh session", "", false},
	auth.ReasonSessionWrite:  {http.StatusInternalServerError, "Failed to refresh session", "", false},
}

// GetBungieAuthURL handles GET /api/auth/bungie
func (h *AuthHandler) GetBungieAuthURL(c *gin.Context) {
	authURL, state, err := h.issuer.AuthorizeURL()
	if err != nil {
		observability.Logger(c.Request.Context()).ErrorContext(c.Request.Context(), "OAuth state generation failed", observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize authentication"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"authUrl": authURL, "state": state})
}

// BungieCallback handles POST /api/auth/bungie/callback
func (h *AuthHandler) BungieCallback(c *gin.Context) {
	session, err := h.issuer.Login(c.Request.Context(), c.PostForm("code"), c.PostForm("state"), c.Request.UserAgent())
	if err != nil {
		h.failSession(c, err, loginFailures)
		return
	}

	h.logAudit(c, db.AuditEvent{
		EventType:         "login.success",
		ActorUserID:       session.UserID,
		ActorMembershipID: session.Profile.MembershipID,
		SessionID:         session.SessionID,
		Details:           map[string]any{"role": auth.RoleName(session.Role)},
	})
	h.setRefreshCookie(c, session)
	c.JSON(http.StatusOK, sessionResponse(session))
}

// RefreshToken handles POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var presented string
	if cookie, err := c.Request.Cookie(refreshCookieName); err == nil {
		presented = cookie.Value
	}

	session, err := h.issuer.Refresh(c.Request.Context(), presented, c.Request.UserAgent())
	if err != nil {
		h.failSession(c, err, refreshFailures)
		return
	}

	h.setRefreshCookie(c, session)
	c.JSON(http.StatusOK, sessionResponse(session))
}

// failSession maps one issuance failure onto its HTTP response, audit event and
// cookie treatment.
func (h *AuthHandler) failSession(c *gin.Context, err error, table map[auth.Reason]authFailure) {
	ctx := c.Request.Context()
	sessErr, ok := auth.AsSessionError(err)
	if !ok {
		observability.Logger(ctx).ErrorContext(ctx, "session issuance failed with an unrecognised error", observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}
	failure, mapped := table[sessErr.Reason]
	if !mapped {
		observability.Logger(ctx).ErrorContext(ctx, "unmapped session failure reason",
			"reason", string(sessErr.Reason), observability.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	if failure.status >= http.StatusInternalServerError {
		observability.Logger(ctx).ErrorContext(ctx, "session issuance failed",
			"reason", string(sessErr.Reason), observability.Err(err))
	}
	if failure.event != "" {
		ev := db.AuditEvent{
			EventType:         failure.event,
			Outcome:           "failure",
			ActorMembershipID: sessErr.MembershipID,
			SessionID:         sessErr.SessionID,
		}
		// refresh.reuse is identified by its event type, not by a reason field.
		if failure.event != "refresh.reuse" {
			ev.Details = map[string]any{"reason": string(sessErr.Reason)}
		}
		h.logAudit(c, ev)
	}
	if failure.expireCookie {
		h.expireRefreshCookie(c)
	}
	c.JSON(failure.status, gin.H{"error": failure.message})
}

// ValidateToken handles GET /api/auth/validate (JWT-protected)
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"valid": true, "user": contextUser(c)})
}

// GetProfile handles GET /api/auth/profile (JWT-protected)
func (h *AuthHandler) GetProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": contextUser(c)})
}

// Logout handles POST /api/auth/logout (JWT-protected) — ends only the current
// device's session. Other devices stay signed in, and the account-wide Bungie
// OAuth token (shared across sessions) is left intact.
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID := c.GetString("session_id")
	if err := h.issuer.EndSession(c.Request.Context(), sessionID); err != nil {
		observability.Logger(c.Request.Context()).WarnContext(c.Request.Context(), "logout session deletion failed",
			observability.ID("session", sessionID), observability.Err(err))
	}

	h.logAudit(c, db.AuditEvent{EventType: "logout.session", ActorMembershipID: c.GetString("membership_id"), SessionID: sessionID})
	h.expireRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// LogoutAll handles POST /api/auth/logout/all (JWT-protected) — signs the user
// out of every device.
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	membershipID := c.GetString("membership_id")
	if err := h.issuer.EndAllSessions(c.Request.Context(), membershipID); err != nil {
		observability.Logger(c.Request.Context()).WarnContext(c.Request.Context(), "account sign-out failed",
			observability.ID("membership", membershipID), observability.Err(err))
	}

	h.logAudit(c, db.AuditEvent{EventType: "logout.all", ActorMembershipID: membershipID})
	h.expireRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "Signed out of all devices"})
}

// sessionResponse is the body both the callback and refresh return. One builder
// because they had two, and the copies had already drifted: refresh omitted the
// role the callback sent.
func sessionResponse(s *auth.Session) gin.H {
	return gin.H{
		"token": s.AccessToken,
		"user":  userPayload(s.Profile.MembershipID, s.Profile.DisplayName, s.Profile.MembershipType, s.Role),
	}
}

// contextUser builds the same payload from what the JWT middleware resolved,
// for the two endpoints that answer from the request context alone.
func contextUser(c *gin.Context) gin.H {
	return userPayload(c.GetString("membership_id"), c.GetString("display_name"), c.GetInt("membership_type"), c.GetInt("role"))
}

// userPayload is the only definition of the `user` object the API returns.
func userPayload(membershipID, displayName string, membershipType, role int) gin.H {
	return gin.H{
		"id":             membershipID,
		"displayName":    displayName,
		"membershipId":   membershipID,
		"membershipType": membershipType,
		"platform":       auth.GetPlatformName(membershipType),
		"role":           auth.RoleName(role),
	}
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, s *auth.Session) {
	h.writeRefreshCookie(c, s.RefreshToken, s.RefreshExpiresAt, int(time.Until(s.RefreshExpiresAt)/time.Second))
}

func (h *AuthHandler) expireRefreshCookie(c *gin.Context) {
	h.writeRefreshCookie(c, "", time.Unix(1, 0), -1)
}

func (h *AuthHandler) writeRefreshCookie(c *gin.Context, value string, expires time.Time, maxAge int) {
	secure := h.cfg != nil && h.cfg.IsProduction()
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    value,
		Path:     refreshCookiePath,
		Expires:  expires.UTC(),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// logAudit writes one event best-effort: it never blocks or fails the request.
// IP and User-Agent are taken from the request; the caller sets the rest.
func (h *AuthHandler) logAudit(c *gin.Context, ev db.AuditEvent) {
	if h.audit == nil {
		return
	}
	ev.IP = c.ClientIP()
	ev.UserAgent = c.Request.UserAgent()
	if err := h.audit.Log(c.Request.Context(), ev); err != nil {
		observability.Logger(c.Request.Context()).WarnContext(c.Request.Context(), "audit event persistence failed",
			"event_type", ev.EventType, observability.Err(err))
	}
}
