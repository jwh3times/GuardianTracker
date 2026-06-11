package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/config"

	"github.com/gin-gonic/gin"
)

// UserStore is satisfied by db.UserStore (via main.go wiring).
type UserStore interface {
	Upsert(ctx context.Context, membershipID string, membershipType int16, displayName string) (id int64, tokenVersion int, err error)
	BumpTokenVersion(ctx context.Context, membershipID string) error
}

// AuthHandler handles Bungie OAuth, token refresh, profile, and logout endpoints.
type AuthHandler struct {
	jwt         *auth.JWT
	tokenStore  *auth.TokenStore
	cfg         *config.Config
	csrf        *csrfStore
	userStore   UserStore             // nil in degraded mode (no DB)
	revokeCache cache.Cache           // for evicting tver: entries on logout
	revoker     *auth.RevocationChecker // nil in degraded mode
}

func NewAuthHandler(ctx context.Context, j *auth.JWT, ts *auth.TokenStore, cfg *config.Config, userStore UserStore, revokeCache cache.Cache, revoker *auth.RevocationChecker) *AuthHandler {
	h := &AuthHandler{
		jwt:         j,
		tokenStore:  ts,
		cfg:         cfg,
		csrf:        newCSRFStore(),
		userStore:   userStore,
		revokeCache: revokeCache,
		revoker:     revoker,
	}
	go h.csrf.cleanupLoop(ctx)
	return h
}

// GetBungieAuthURL handles GET /api/auth/bungie
func (h *AuthHandler) GetBungieAuthURL(c *gin.Context) {
	state, err := h.csrf.generate()
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
	if state == "" || !h.csrf.validateAndConsume(state) {
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

	// Upsert user to DB (best-effort — if DB is unavailable, use tokenVersion=1)
	tokenVersion := 1
	if h.userStore != nil {
		_, tv, err := h.userStore.Upsert(c.Request.Context(), profile.MembershipID, int16(profile.MembershipType), profile.DisplayName)
		if err != nil {
			log.Printf("user upsert failed for %s: %v", profile.MembershipID, err)
		} else {
			tokenVersion = tv
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

	accessToken, err := h.jwt.GenerateAccessToken(profile, tokenVersion)
	if err != nil {
		log.Printf("Error generating access token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}
	refreshToken, err := h.jwt.GenerateRefreshToken(profile, tokenVersion)
	if err != nil {
		log.Printf("Error generating refresh token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
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

	// Reject refresh attempts for revoked sessions (e.g. logout on another device).
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

	newAccess, err := h.jwt.GenerateAccessToken(profile, claims.TokenVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh session"})
		return
	}
	newRefresh, err := h.jwt.GenerateRefreshToken(profile, claims.TokenVersion)
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
		},
	})
}

// Logout handles POST /api/auth/logout (JWT-protected)
func (h *AuthHandler) Logout(c *gin.Context) {
	membershipID := c.GetString("membership_id")

	// Bump token_version to invalidate all issued JWTs
	if h.userStore != nil {
		if err := h.userStore.BumpTokenVersion(c.Request.Context(), membershipID); err != nil {
			log.Printf("logout: bump token_version failed for %s: %v", membershipID, err)
		}
	}

	// Delete Bungie tokens from memory and DB
	h.tokenStore.Delete(membershipID)

	// Evict revocation cache key so the next request hits DB immediately
	if h.revokeCache != nil {
		h.revokeCache.Delete("tver:" + membershipID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
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

// csrfStore is a small in-process CSRF state machine.
type csrfStore struct {
	mu     sync.RWMutex
	states map[string]time.Time
}

func newCSRFStore() *csrfStore {
	return &csrfStore{states: make(map[string]time.Time)}
}

func (s *csrfStore) generate() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.states[state] = time.Now().Add(10 * time.Minute)
	s.mu.Unlock()
	return state, nil
}

func (s *csrfStore) validateAndConsume(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.states[state]
	if !ok {
		return false
	}
	delete(s.states, state)
	return time.Now().Before(expiry)
}

func (s *csrfStore) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for state, expiry := range s.states {
				if now.After(expiry) {
					delete(s.states, state)
				}
			}
			s.mu.Unlock()
		}
	}
}
