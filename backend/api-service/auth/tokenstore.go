package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Sentinels the DB adapter translates from db-layer errors so the store can
// react to each case without importing the db package (avoids an import cycle).
var (
	// ErrTokensNotFound: no token row exists for the membership.
	ErrTokensNotFound = errors.New("bungie tokens not found")
	// ErrNoUserRow: no users row exists, so tokens cannot be persisted.
	ErrNoUserRow = errors.New("no users row for membership")
	// errCASLost: a compare-and-swap refresh write lost to a concurrent writer.
	errCASLost = errors.New("token cas lost")
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

	// dbUpdatedAt is the updated_at of the DB row these tokens came from (or last
	// wrote). It is the compare-and-swap baseline for refresh writes: a mismatch
	// means another replica refreshed first and we must adopt its tokens.
	dbUpdatedAt time.Time
}

func (t *BungieTokens) isAccessTokenExpired() bool {
	return time.Now().Add(5 * time.Minute).After(t.AccessTokenExpiresAt)
}

func (t *BungieTokens) isRefreshTokenExpired() bool {
	return time.Now().After(t.RefreshTokenExpiresAt)
}

// EncryptedTokenRecord is what the DB repo returns / accepts.
// Defined here (in auth package) to avoid import cycles with the db package.
type EncryptedTokenRecord struct {
	AccessTokenEnc   []byte
	RefreshTokenEnc  []byte
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	KeyVersion       int16
	UpdatedAt        time.Time
}

// TokenRepo is the interface satisfied by db.BungieTokenStore (via an adapter in main.go).
// Upsert: zero prevUpdatedAt = unconditional write; non-zero = CAS against the row's
// updated_at. Returns the row's new updated_at and whether the write applied.
type TokenRepo interface {
	Get(ctx context.Context, membershipID string) (*EncryptedTokenRecord, error)
	Upsert(ctx context.Context, membershipID string, t *EncryptedTokenRecord, prevUpdatedAt time.Time) (time.Time, bool, error)
	Delete(ctx context.Context, membershipID string) error
}

// bungieTokenURL is Bungie's OAuth token endpoint (overridable in tests).
const bungieTokenURL = "https://www.bungie.net/platform/app/oauth/token/"

// TokenStore manages Bungie OAuth tokens with optional DB write-through.
type TokenStore struct {
	mu           sync.RWMutex
	tokens       map[string]*BungieTokens
	refreshLocks sync.Map // map[membershipID]*sync.Mutex — serializes per-user refresh
	clientID     string
	clientSecret string
	repo         TokenRepo    // nil = memory-only (degraded mode)
	cipher       *TokenCipher // nil = no encryption
	tokenURL     string       // Bungie OAuth token endpoint
}

// NewTokenStore creates a token store and starts the background cleanup goroutine.
// repo and cipher may both be nil for degraded/dev mode (memory-only, no encryption).
func NewTokenStore(ctx context.Context, clientID, clientSecret string, repo TokenRepo, cipher *TokenCipher) *TokenStore {
	s := &TokenStore{
		tokens:       make(map[string]*BungieTokens),
		clientID:     clientID,
		clientSecret: clientSecret,
		repo:         repo,
		cipher:       cipher,
		tokenURL:     bungieTokenURL,
	}
	go s.cleanupLoop(ctx)
	return s
}

// Store saves (or replaces) tokens for a user, writing through to the DB when available.
func (s *TokenStore) Store(membershipID string, tokens *BungieTokens) {
	tokens.MembershipID = membershipID

	// Write to DB first (best-effort — don't fail if DB is down)
	if s.repo != nil && s.cipher != nil {
		switch err := s.persist(membershipID, tokens, time.Time{}); {
		case err == nil:
			// persisted
		case errors.Is(err, ErrNoUserRow):
			log.Printf("TokenStore.Store: no users row for %s — tokens kept in memory only", membershipID)
		default:
			log.Printf("TokenStore.Store: DB persist failed for %s: %v", membershipID, err)
		}
	}

	// Always update memory cache
	s.mu.Lock()
	s.tokens[membershipID] = tokens
	s.mu.Unlock()
	log.Printf("Stored Bungie tokens for user %s (expires: %s)", membershipID, tokens.AccessTokenExpiresAt.Format(time.RFC3339))
}

// persist encrypts and upserts tokens to the DB. A zero prevUpdatedAt writes
// unconditionally; non-zero is a CAS against the row's updated_at (see TokenRepo).
// On a successful write, tokens.dbUpdatedAt is advanced to the row's new updated_at.
// Returns nil on success, errCASLost when a CAS write lost the race, ErrNoUserRow
// when no users row exists, or another error on encrypt/DB failure.
// Callers must ensure repo and cipher are non-nil.
func (s *TokenStore) persist(membershipID string, tokens *BungieTokens, prevUpdatedAt time.Time) error {
	aad := []byte(membershipID)
	accEnc, kv, err := s.cipher.Encrypt([]byte(tokens.AccessToken), aad)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}
	refEnc, _, err := s.cipher.Encrypt([]byte(tokens.RefreshToken), aad)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}
	rec := &EncryptedTokenRecord{
		AccessTokenEnc:   accEnc,
		RefreshTokenEnc:  refEnc,
		AccessExpiresAt:  tokens.AccessTokenExpiresAt,
		RefreshExpiresAt: tokens.RefreshTokenExpiresAt,
		KeyVersion:       kv,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	newUpdatedAt, updated, err := s.repo.Upsert(ctx, membershipID, rec, prevUpdatedAt)
	if err != nil {
		return err // includes ErrNoUserRow
	}
	if !updated {
		return errCASLost
	}
	tokens.dbUpdatedAt = newUpdatedAt
	return nil
}

// loadFromDB reads and decrypts the stored tokens for membershipID.
// A nil result with a nil error means the row is definitively absent or
// unreadable (safe to overwrite). A non-nil error means the read failed
// transiently — callers must NOT treat that as "no winner" and overwrite.
func (s *TokenStore) loadFromDB(membershipID string) (*BungieTokens, error) {
	if s.repo == nil || s.cipher == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec, err := s.repo.Get(ctx, membershipID)
	if errors.Is(err, ErrTokensNotFound) {
		return nil, nil // definitively absent
	}
	if err != nil {
		return nil, err // transient — caller must not overwrite
	}
	aad := []byte(membershipID)
	accessPlain, err1 := s.cipher.Decrypt(rec.AccessTokenEnc, aad, rec.KeyVersion)
	refreshPlain, err2 := s.cipher.Decrypt(rec.RefreshTokenEnc, aad, rec.KeyVersion)
	if err1 != nil || err2 != nil {
		return nil, nil // unreadable — treat as absent
	}
	return &BungieTokens{
		AccessToken:           string(accessPlain),
		RefreshToken:          string(refreshPlain),
		AccessTokenExpiresAt:  rec.AccessExpiresAt,
		RefreshTokenExpiresAt: rec.RefreshExpiresAt,
		MembershipID:          membershipID,
		dbUpdatedAt:           rec.UpdatedAt,
	}, nil
}

// GetValidToken returns a valid Bungie access token, refreshing it first if needed.
// Per-user mutex ensures only one refresh call fires at a time for the same user,
// preventing double-refresh races where the second call would use an already-consumed token.
func (s *TokenStore) GetValidToken(membershipID string) (string, error) {
	s.mu.RLock()
	tokens, exists := s.tokens[membershipID]
	s.mu.RUnlock()

	// Memory miss — try DB
	if !exists {
		if loaded, _ := s.loadFromDB(membershipID); loaded != nil {
			tokens = loaded
			s.mu.Lock()
			s.tokens[membershipID] = tokens
			s.mu.Unlock()
			exists = true
		}
	}

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
	newTokens.MembershipID = membershipID

	if s.repo != nil && s.cipher != nil {
		// CAS against the updated_at we loaded: if another replica refreshed first,
		// adopt its tokens instead of clobbering the row with tokens derived from a
		// refresh token Bungie has already rotated (PLAN §2.6).
		if err := s.persist(membershipID, newTokens, tokens.dbUpdatedAt); errors.Is(err, errCASLost) {
			winner, loadErr := s.loadFromDB(membershipID)
			switch {
			case winner != nil && !winner.isAccessTokenExpired():
				s.mu.Lock()
				s.tokens[membershipID] = winner
				s.mu.Unlock()
				log.Printf("TokenStore: refresh race for %s — adopted newer DB tokens", membershipID)
				return winner.AccessToken, nil
			case loadErr == nil:
				// Confirmed read: the winner's row is absent, unreadable, or expired,
				// so our freshly minted tokens are strictly newer — write them
				// unconditionally (best-effort).
				if perr := s.persist(membershipID, newTokens, time.Time{}); perr != nil {
					log.Printf("TokenStore: unconditional re-persist for %s failed: %v", membershipID, perr)
				}
			default:
				// Transient read failure — do NOT clobber a possibly-valid winner.
				// Keep newTokens in memory only; the next refresh will reconcile.
				log.Printf("TokenStore: refresh race for %s — DB read failed, keeping new tokens in memory: %v", membershipID, loadErr)
			}
		} else if err != nil && !errors.Is(err, ErrNoUserRow) {
			log.Printf("TokenStore: persist after refresh for %s failed: %v", membershipID, err)
		}
	}

	s.mu.Lock()
	s.tokens[membershipID] = newTokens
	s.mu.Unlock()
	log.Printf("Stored Bungie tokens for user %s (expires: %s)", membershipID, newTokens.AccessTokenExpiresAt.Format(time.RFC3339))
	return newTokens.AccessToken, nil
}

// Delete removes tokens for a user (e.g., on logout) from both memory and DB.
func (s *TokenStore) Delete(membershipID string) {
	s.mu.Lock()
	delete(s.tokens, membershipID)
	s.mu.Unlock()
	if s.repo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.repo.Delete(ctx, membershipID); err != nil {
			log.Printf("TokenStore.Delete: DB delete failed for %s: %v", membershipID, err)
		}
	}
}

func (s *TokenStore) refreshBungieToken(tokens *BungieTokens) (*BungieTokens, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", tokens.RefreshToken)
	data.Set("client_id", s.clientID)
	if s.clientSecret != "" {
		data.Set("client_secret", s.clientSecret)
	}

	req, err := http.NewRequest("POST", s.tokenURL, strings.NewReader(data.Encode()))
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
