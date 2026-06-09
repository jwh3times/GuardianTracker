package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BungieUserProfile holds the Bungie account info we care about after OAuth.
type BungieUserProfile struct {
	MembershipID   string `json:"membershipId"`
	DisplayName    string `json:"displayName"`
	MembershipType int    `json:"membershipType"`
}

// BungieTokens holds the OAuth tokens for one user.
type BungieTokens struct {
	AccessToken           string    `json:"accessToken"`
	RefreshToken          string    `json:"refreshToken"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
	MembershipID          string    `json:"membershipId"`
}

func (t *BungieTokens) isAccessTokenExpired() bool {
	return time.Now().Add(5 * time.Minute).After(t.AccessTokenExpiresAt)
}

func (t *BungieTokens) isRefreshTokenExpired() bool {
	return time.Now().After(t.RefreshTokenExpiresAt)
}

// TokenStore manages Bungie OAuth tokens in memory for all active users.
type TokenStore struct {
	mu           sync.RWMutex
	tokens       map[string]*BungieTokens
	refreshLocks sync.Map // map[membershipID]*sync.Mutex — serializes per-user refresh
	clientID     string
	clientSecret string
}

// NewTokenStore creates a token store and starts the background cleanup goroutine.
// The cleanup goroutine exits when ctx is cancelled.
func NewTokenStore(ctx context.Context, clientID, clientSecret string) *TokenStore {
	s := &TokenStore{
		tokens:       make(map[string]*BungieTokens),
		clientID:     clientID,
		clientSecret: clientSecret,
	}
	go s.cleanupLoop(ctx)
	return s
}

// Store saves (or replaces) tokens for a user.
func (s *TokenStore) Store(membershipID string, tokens *BungieTokens) {
	tokens.MembershipID = membershipID
	s.mu.Lock()
	s.tokens[membershipID] = tokens
	s.mu.Unlock()
	log.Printf("Stored Bungie tokens for user %s (expires: %s)", membershipID, tokens.AccessTokenExpiresAt.Format(time.RFC3339))
}

// GetValidToken returns a valid Bungie access token, refreshing it first if needed.
// Per-user mutex ensures only one refresh call fires at a time for the same user,
// preventing double-refresh races where the second call would use an already-consumed token.
func (s *TokenStore) GetValidToken(membershipID string) (string, error) {
	s.mu.RLock()
	tokens, exists := s.tokens[membershipID]
	s.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("no tokens found for user %s", membershipID)
	}
	if tokens.isRefreshTokenExpired() {
		s.mu.Lock()
		delete(s.tokens, membershipID)
		s.mu.Unlock()
		return "", fmt.Errorf("refresh token expired for user %s, re-authentication required", membershipID)
	}
	if !tokens.isAccessTokenExpired() {
		return tokens.AccessToken, nil
	}

	// Access token expired. Acquire per-user lock to serialize the refresh call.
	mu, _ := s.refreshLocks.LoadOrStore(membershipID, &sync.Mutex{})
	userMu := mu.(*sync.Mutex)
	userMu.Lock()
	defer userMu.Unlock()

	// Re-check under the per-user lock: a concurrent goroutine may have refreshed by now.
	s.mu.RLock()
	tokens = s.tokens[membershipID]
	s.mu.RUnlock()
	if tokens == nil {
		return "", fmt.Errorf("no tokens found for user %s", membershipID)
	}
	if !tokens.isAccessTokenExpired() {
		return tokens.AccessToken, nil
	}

	log.Printf("Access token expired for user %s, refreshing...", membershipID)
	newTokens, err := s.refreshBungieToken(tokens)
	if err != nil {
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}
	s.Store(membershipID, newTokens)
	return newTokens.AccessToken, nil
}

// Delete removes tokens for a user (e.g., on logout).
func (s *TokenStore) Delete(membershipID string) {
	s.mu.Lock()
	delete(s.tokens, membershipID)
	s.mu.Unlock()
}

func (s *TokenStore) refreshBungieToken(tokens *BungieTokens) (*BungieTokens, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", tokens.RefreshToken)
	data.Set("client_id", s.clientID)
	if s.clientSecret != "" {
		data.Set("client_secret", s.clientSecret)
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
		return nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tr struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		ExpiresIn        int    `json:"expires_in"`
		RefreshExpiresIn int    `json:"refresh_expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	now := time.Now()
	refreshExpiry := time.Duration(tr.RefreshExpiresIn) * time.Second
	if refreshExpiry <= 0 {
		refreshExpiry = 90 * 24 * time.Hour // Bungie default fallback
	}
	return &BungieTokens{
		AccessToken:           tr.AccessToken,
		RefreshToken:          tr.RefreshToken,
		AccessTokenExpiresAt:  now.Add(time.Duration(tr.ExpiresIn) * time.Second),
		RefreshTokenExpiresAt: now.Add(refreshExpiry),
		MembershipID:          tokens.MembershipID,
	}, nil
}

func (s *TokenStore) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, t := range s.tokens {
				if now.After(t.RefreshTokenExpiresAt) {
					delete(s.tokens, id)
					log.Printf("Cleaned up expired tokens for user %s", id)
				}
			}
			s.mu.Unlock()
		}
	}
}
