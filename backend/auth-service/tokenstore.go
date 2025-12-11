package main

import (
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

// BungieTokens holds the OAuth tokens for a user
type BungieTokens struct {
	AccessToken           string    `json:"accessToken"`
	RefreshToken          string    `json:"refreshToken"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
	MembershipID          string    `json:"membershipId"`
}

// IsAccessTokenExpired checks if the access token needs refresh
func (t *BungieTokens) IsAccessTokenExpired() bool {
	// Consider expired 5 minutes before actual expiry for safety
	return time.Now().Add(5 * time.Minute).After(t.AccessTokenExpiresAt)
}

// IsRefreshTokenExpired checks if the refresh token is expired
func (t *BungieTokens) IsRefreshTokenExpired() bool {
	return time.Now().After(t.RefreshTokenExpiresAt)
}

// TokenStore manages Bungie OAuth tokens for all users
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*BungieTokens // keyed by membershipId
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	store := &TokenStore{
		tokens: make(map[string]*BungieTokens),
	}
	// Start background cleanup
	go store.cleanupExpiredTokens()
	return store
}

// Store saves tokens for a user
func (s *TokenStore) Store(membershipID string, tokens *BungieTokens) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens.MembershipID = membershipID
	s.tokens[membershipID] = tokens
	log.Printf("Stored Bungie tokens for user %s (expires: %s)", membershipID, tokens.AccessTokenExpiresAt.Format(time.RFC3339))
}

// Get retrieves tokens for a user
func (s *TokenStore) Get(membershipID string) (*BungieTokens, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tokens, exists := s.tokens[membershipID]
	return tokens, exists
}

// Delete removes tokens for a user
func (s *TokenStore) Delete(membershipID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, membershipID)
}

// GetValidToken retrieves a valid access token, refreshing if necessary
func (s *TokenStore) GetValidToken(membershipID string) (string, error) {
	tokens, exists := s.Get(membershipID)
	if !exists {
		return "", fmt.Errorf("no tokens found for user %s", membershipID)
	}

	// Check if refresh token is expired (user needs to re-login)
	if tokens.IsRefreshTokenExpired() {
		s.Delete(membershipID)
		return "", fmt.Errorf("refresh token expired for user %s, re-authentication required", membershipID)
	}

	// Check if access token needs refresh
	if tokens.IsAccessTokenExpired() {
		log.Printf("Access token expired for user %s, refreshing...", membershipID)
		newTokens, err := s.refreshBungieToken(tokens)
		if err != nil {
			log.Printf("Failed to refresh token for user %s: %v", membershipID, err)
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
		s.Store(membershipID, newTokens)
		return newTokens.AccessToken, nil
	}

	return tokens.AccessToken, nil
}

// refreshBungieToken exchanges a refresh token for new tokens
func (s *TokenStore) refreshBungieToken(tokens *BungieTokens) (*BungieTokens, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", tokens.RefreshToken)
	data.Set("client_id", config.BungieClientID)
	if config.BungieClientSecret != "" {
		data.Set("client_secret", config.BungieClientSecret)
	}

	req, err := http.NewRequest("POST", "https://www.bungie.net/platform/app/oauth/token/", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
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

	var tokenResp struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		ExpiresIn        int    `json:"expires_in"`
		RefreshExpiresIn int    `json:"refresh_expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	now := time.Now()
	return &BungieTokens{
		AccessToken:           tokenResp.AccessToken,
		RefreshToken:          tokenResp.RefreshToken,
		AccessTokenExpiresAt:  now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		RefreshTokenExpiresAt: now.Add(time.Duration(tokenResp.RefreshExpiresIn) * time.Second),
		MembershipID:          tokens.MembershipID,
	}, nil
}

// cleanupExpiredTokens periodically removes expired tokens
func (s *TokenStore) cleanupExpiredTokens() {
	ticker := time.NewTicker(15 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for membershipID, tokens := range s.tokens {
			// Remove if refresh token is expired
			if now.After(tokens.RefreshTokenExpiresAt) {
				delete(s.tokens, membershipID)
				log.Printf("Cleaned up expired tokens for user %s", membershipID)
			}
		}
		s.mu.Unlock()
	}
}

// Global token store instance
var bungieTokenStore *TokenStore

func init() {
	bungieTokenStore = NewTokenStore()
}
