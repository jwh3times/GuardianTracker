package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserStore is satisfied by db.UserStore (via main.go wiring).
type UserStore interface {
	Upsert(ctx context.Context, membershipID string, membershipType int16, displayName string, forceAdmin bool) (id int64, tokenVersion int, role int16, err error)
	BumpTokenVersion(ctx context.Context, membershipID string) error
	// CreateSession records a new per-device refresh session at login.
	CreateSession(ctx context.Context, id, membershipID, jti, userAgent string, expiresAt time.Time) error
	// RotateSession advances a session's refresh jti with reuse detection and slides
	// expires_at to newExpiresAt; rotated=true on success, reused=true when a replayed
	// token revoked the session.
	RotateSession(ctx context.Context, id, membershipID, oldJTI, newJTI string, newExpiresAt time.Time) (rotated bool, reused bool, err error)
	// DeleteSession ends one session (this-device logout); DeleteUserSessions ends all.
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, membershipID string) error
}

// oauthStateTTL is how long an issued OAuth state parameter stays valid.
const oauthStateTTL = 10 * time.Minute

// AuthHandler handles Bungie OAuth, token refresh, profile, and logout endpoints.
type AuthHandler struct {
	jwt         *auth.JWT
	tokenStore  *auth.TokenStore
	cfg         *config.Config
	state       *auth.StateSigner
	userStore   UserStore               // nil in degraded mode (no DB)
	revokeCache cache.Cache             // for evicting tver: entries on logout
	revoker     *auth.RevocationChecker // nil in degraded mode
}

func NewAuthHandler(j *auth.JWT, ts *auth.TokenStore, cfg *config.Config, userStore UserStore, revokeCache cache.Cache, revoker *auth.RevocationChecker) *AuthHandler {
	return &AuthHandler{
		jwt:         j,
		tokenStore:  ts,
		cfg:         cfg,
		state:       auth.NewStateSigner(cfg.JWTSecret),
		userStore:   userStore,
		revokeCache: revokeCache,
		revoker:     revoker,
	}
}

// GetBungieAuthURL handles GET /api/auth/bungie
func (h *AuthHandler) GetBungieAuthURL(c *gin.Context) {
	state, err := h.state.Generate()
	if err != nil {
		log.Printf("Error generating CSRF state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize authentication"})
		return
	}
	authURL := fmt.Sprintf(
		"https://www.bungie.net/en/OAuth/Authorize?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
		h.cfg.BungieClientID,
		url.QueryEscape(h.cfg.AuthRedirectURI),
		url.QueryEscape(state),
	)
	c.JSON(http.StatusOK, gin.H{"authUrl": authURL, "state": state})
}

// BungieCallback handles POST /api/auth/bungie/callback
func (h *AuthHandler) BungieCallback(c *gin.Context) {
	code := c.PostForm("code")
	state := c.PostForm("state")

	if code == "" || len(code) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid authorization code"})
		return
	}
	if state == "" || !h.state.Verify(state, time.Now(), oauthStateTTL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired state. Please try logging in again."})
		return
	}

	tokenResp, err := h.exchangeCode(code)
	if err != nil {
		log.Printf("Error exchanging code for token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete authentication"})
		return
	}

	profile, err := h.getBungieProfile(tokenResp.AccessToken)
	if err != nil {
		log.Printf("Error getting user profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		return
	}

	// Upsert user to DB (best-effort — if DB is unavailable, use tokenVersion=1).
	// Membership IDs in ADMIN_MEMBERSHIP_IDS are pinned to admin on every login.
	tokenVersion := 1
	role := auth.RoleStandard
	if h.userStore != nil {
		forceAdmin := h.cfg.IsBootstrapAdmin(profile.MembershipID)
		_, tv, r, err := h.userStore.Upsert(c.Request.Context(), profile.MembershipID, int16(profile.MembershipType), profile.DisplayName, forceAdmin)
		if err != nil {
			log.Printf("user upsert failed for %s: %v", profile.MembershipID, err)
		} else {
			tokenVersion = tv
			role = int(r)
		}
	}

	now := time.Now()
	refreshExpiry := time.Duration(tokenResp.RefreshExpiresIn) * time.Second
	if refreshExpiry <= 0 {
		refreshExpiry = 90 * 24 * time.Hour // Bungie default fallback
	}
	h.tokenStore.Store(profile.MembershipID, &auth.BungieTokens{
		AccessToken:           tokenResp.AccessToken,
		RefreshToken:          tokenResp.RefreshToken,
		AccessTokenExpiresAt:  now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		RefreshTokenExpiresAt: now.Add(refreshExpiry),
		MembershipID:          profile.MembershipID,
	})

	// Each login opens a new per-device session. Both tokens carry its id (sid).
	sessionID := uuid.NewString()
	accessToken, err := h.jwt.GenerateAccessToken(profile, tokenVersion, sessionID)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}
	refreshToken, refreshJTI, err := h.jwt.GenerateRefreshToken(profile, tokenVersion, sessionID)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Persist the session before returning the tokens. The session is load-bearing —
	// the access token is rejected on its next request if no live session row exists —
	// so a failed write must fail the login (500) rather than hand back a session that
	// is dead on arrival.
	if h.userStore != nil {
		expiresAt := now.Add(h.jwt.RefreshTokenTTL())
		if err := h.userStore.CreateSession(c.Request.Context(), sessionID, profile.MembershipID, refreshJTI, c.Request.UserAgent(), expiresAt); err != nil {
			log.Printf("create session failed for %s: %v", profile.MembershipID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        accessToken,
		"refreshToken": refreshToken,
		"user": gin.H{
			"id":             profile.MembershipID,
			"displayName":    profile.DisplayName,
			"membershipId":   profile.MembershipID,
			"membershipType": profile.MembershipType,
			"platform":       auth.GetPlatformName(profile.MembershipType),
			"role":           auth.RoleName(role),
		},
	})
}

// RefreshToken handles POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token is required"})
		return
	}

	claims, err := h.jwt.ValidateToken(body.RefreshToken)
	if err != nil || claims.TokenType != "refresh" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	// Reject refresh attempts after a sign-out-everywhere (token_version bump).
	if h.revoker != nil {
		if err := h.revoker.Check(c.Request.Context(), claims.MembershipID, claims.TokenVersion); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session has been revoked. Please log in again."})
			return
		}
	}

	profile := &auth.BungieUserProfile{
		MembershipID:   claims.MembershipID,
		DisplayName:    claims.DisplayName,
		MembershipType: claims.MembershipType,
	}

	// A token issued before sessions existed carries no sid; adopt it into a fresh
	// session on its first refresh so subsequent refreshes are protected.
	sessionID := claims.SessionID
	legacy := sessionID == "" && h.userStore != nil
	if legacy {
		sessionID = uuid.NewString()
	}

	newRefresh, newJTI, err := h.jwt.GenerateRefreshToken(profile, claims.TokenVersion, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh session"})
		return
	}

	// Rotate the session's refresh jti with reuse detection. A replayed (already
	// rotated) token revokes the session; an unknown/expired session forces re-login.
	// expires_at slides forward to match the freshly issued refresh token.
	if h.userStore != nil {
		expiresAt := time.Now().Add(h.jwt.RefreshTokenTTL())
		if legacy {
			// The adopted session is load-bearing for the new access token, so a failed
			// write must fail the refresh rather than hand back a dead-on-arrival session.
			if cerr := h.userStore.CreateSession(c.Request.Context(), sessionID, claims.MembershipID, newJTI, c.Request.UserAgent(), expiresAt); cerr != nil {
				log.Printf("adopt legacy session failed for %s: %v", claims.MembershipID, cerr)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh session"})
				return
			}
		} else {
			rotated, reused, rerr := h.userStore.RotateSession(c.Request.Context(), sessionID, claims.MembershipID, claims.ID, newJTI, expiresAt)
			switch {
			case reused:
				// Definitive reuse — reject even if the revoking commit later errored.
				log.Printf("refresh-token reuse detected for %s (session %s) — session revoked", claims.MembershipID, sessionID)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Session ended for security reasons. Please log in again."})
				return
			case rerr != nil:
				// Genuine DB error (not reuse): fail open, consistent with the revocation
				// check above — the new jti goes unrecorded, so the next refresh may re-login.
				log.Printf("rotate session failed for %s: %v — allowing refresh", claims.MembershipID, rerr)
			case !rotated:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Session has expired. Please log in again."})
				return
			}
		}
	}

	newAccess, err := h.jwt.GenerateAccessToken(profile, claims.TokenVersion, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        newAccess,
		"refreshToken": newRefresh,
		"user": gin.H{
			"id":             claims.MembershipID,
			"displayName":    claims.DisplayName,
			"membershipId":   claims.MembershipID,
			"membershipType": claims.MembershipType,
			"platform":       claims.Platform,
		},
	})
}

// ValidateToken handles GET /api/auth/validate (JWT-protected)
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user": gin.H{
			"id":             c.GetString("membership_id"),
			"displayName":    c.GetString("display_name"),
			"membershipId":   c.GetString("membership_id"),
			"membershipType": c.GetInt("membership_type"),
			"platform":       c.GetString("platform"),
		},
	})
}

// GetProfile handles GET /api/auth/profile (JWT-protected)
func (h *AuthHandler) GetProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":             c.GetString("membership_id"),
			"displayName":    c.GetString("display_name"),
			"membershipId":   c.GetString("membership_id"),
			"membershipType": c.GetInt("membership_type"),
			"platform":       c.GetString("platform"),
			"role":           auth.RoleName(c.GetInt("role")),
		},
	})
}

// Logout handles POST /api/auth/logout (JWT-protected) — ends only the current
// device's session. Other devices stay signed in, and the account-wide Bungie OAuth
// token (shared across sessions) is left intact.
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID := c.GetString("session_id")

	if h.userStore != nil && sessionID != "" {
		if err := h.userStore.DeleteSession(c.Request.Context(), sessionID); err != nil {
			log.Printf("logout: delete session %s failed: %v", sessionID, err)
		}
	}
	// Evict the session cache entry so this device's access token is rejected on its
	// next request (within the cache window) rather than lingering until expiry.
	if h.revokeCache != nil && sessionID != "" {
		h.revokeCache.Delete("sess:" + sessionID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// LogoutAll handles POST /api/auth/logout/all (JWT-protected) — signs the user out
// of every device: bumps token_version (invalidates all access tokens), deletes all
// sessions, and evicts the account-wide Bungie OAuth token.
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	membershipID := c.GetString("membership_id")

	if h.userStore != nil {
		if err := h.userStore.BumpTokenVersion(c.Request.Context(), membershipID); err != nil {
			log.Printf("logout-all: bump token_version failed for %s: %v", membershipID, err)
		}
		if err := h.userStore.DeleteUserSessions(c.Request.Context(), membershipID); err != nil {
			log.Printf("logout-all: delete sessions failed for %s: %v", membershipID, err)
		}
	}

	// Delete Bungie tokens from memory and DB (account-wide).
	h.tokenStore.Delete(membershipID)

	// Evict the per-user revocation entry so the token_version bump takes effect
	// immediately. Other sessions' cache entries age out within the TTL, but their
	// access tokens already fail the bumped token_version check.
	if h.revokeCache != nil {
		h.revokeCache.Delete("tver:" + membershipID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Signed out of all devices"})
}

// Bungie OAuth helpers

type bungieTokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	MembershipID     string `json:"membership_id"`
}

func (h *AuthHandler) exchangeCode(code string) (*bungieTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", h.cfg.BungieClientID)
	if h.cfg.BungieClientSecret != "" {
		data.Set("client_secret", h.cfg.BungieClientSecret)
	}

	req, err := http.NewRequest("POST", "https://www.bungie.net/platform/app/oauth/token/", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tr bungieTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	return &tr, nil
}

type bungieAPIResponse struct {
	Response struct {
		DestinyMemberships []struct {
			MembershipType int    `json:"membershipType"`
			MembershipID   string `json:"membershipId"`
			DisplayName    string `json:"displayName"`
		} `json:"destinyMemberships"`
	} `json:"Response"`
}

func (h *AuthHandler) getBungieProfile(accessToken string) (*auth.BungieUserProfile, error) {
	req, err := http.NewRequest("GET", "https://www.bungie.net/Platform/User/GetMembershipsForCurrentUser/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-API-Key", h.cfg.BungieAPIKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("profile fetch failed with status %d", resp.StatusCode)
	}

	var apiResp bungieAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse profile response: %w", err)
	}
	if len(apiResp.Response.DestinyMemberships) == 0 {
		return nil, fmt.Errorf("no Destiny memberships found for user")
	}

	m := apiResp.Response.DestinyMemberships[0]
	return &auth.BungieUserProfile{
		MembershipID:   m.MembershipID,
		DisplayName:    m.DisplayName,
		MembershipType: m.MembershipType,
	}, nil
}
