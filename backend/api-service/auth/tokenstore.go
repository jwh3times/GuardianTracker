package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"guardian-tracker/api-service/observability"
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

// TokenStore manages Bungie OAuth tokens with optional DB write-through.
type TokenStore struct {
	mu           sync.RWMutex
	tokens       map[string]*BungieTokens
	refreshLocks sync.Map     // map[membershipID]*sync.Mutex — serializes per-user refresh
	repo         TokenRepo    // nil = memory-only (degraded mode)
	cipher       *TokenCipher // nil = no encryption
	oauth        *bungieOAuth // Bungie's OAuth token endpoint, shared with SessionIssuer
}

// NewTokenStore creates a token store and starts the background cleanup goroutine.
// repo and cipher may both be nil for degraded/dev mode. tokenURL overrides the
// Bungie OAuth token endpoint (E2E/fake-Bungie); empty uses the real one.
func NewTokenStore(ctx context.Context, clientID, clientSecret, tokenURL string, repo TokenRepo, cipher *TokenCipher) *TokenStore {
	s := &TokenStore{
		tokens: make(map[string]*BungieTokens),
		repo:   repo,
		cipher: cipher,
		oauth:  newBungieOAuth(clientID, clientSecret, tokenURL),
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
			slog.Warn("Bungie tokens retained in memory because user row is missing", observability.ID("membership", membershipID))
		default:
			slog.Warn("Bungie token persistence failed", observability.ID("membership", membershipID), observability.Err(err))
		}
	}

	// Always update memory cache
	s.mu.Lock()
	s.tokens[membershipID] = tokens
	s.mu.Unlock()
	slog.Info("Bungie tokens stored", observability.ID("membership", membershipID), "access_expires_at", tokens.AccessTokenExpiresAt)
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
		return "", fmt.Errorf("no Bungie tokens found")
	}
	if tokens.isRefreshTokenExpired() {
		s.mu.Lock()
		delete(s.tokens, membershipID)
		s.mu.Unlock()
		return "", fmt.Errorf("refresh token expired; re-authentication required")
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
		return "", fmt.Errorf("no Bungie tokens found")
	}
	if !tokens.isAccessTokenExpired() {
		return tokens.AccessToken, nil
	}

	slog.Info("refreshing expired Bungie access token", observability.ID("membership", membershipID))
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
				slog.Info("adopted Bungie tokens from concurrent refresh", observability.ID("membership", membershipID))
				return winner.AccessToken, nil
			case loadErr == nil:
				// Confirmed read: the winner's row is absent, unreadable, or expired,
				// so our freshly minted tokens are strictly newer — write them
				// unconditionally (best-effort).
				if perr := s.persist(membershipID, newTokens, time.Time{}); perr != nil {
					slog.Warn("Bungie token re-persistence failed", observability.ID("membership", membershipID), observability.Err(perr))
				}
			default:
				// Transient read failure — do NOT clobber a possibly-valid winner.
				// Keep newTokens in memory only; the next refresh will reconcile.
				slog.Warn("concurrent Bungie refresh reconciliation failed; retaining tokens in memory",
					observability.ID("membership", membershipID), observability.Err(loadErr))
			}
		} else if err != nil && !errors.Is(err, ErrNoUserRow) {
			slog.Warn("Bungie token persistence after refresh failed", observability.ID("membership", membershipID), observability.Err(err))
		}
	}

	s.mu.Lock()
	s.tokens[membershipID] = newTokens
	s.mu.Unlock()
	slog.Info("refreshed Bungie tokens stored", observability.ID("membership", membershipID), "access_expires_at", newTokens.AccessTokenExpiresAt)
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
			slog.Warn("Bungie token deletion failed", observability.ID("membership", membershipID), observability.Err(err))
		}
	}
}

// refreshBungieToken trades the stored Bungie refresh token for a fresh pair.
// The request itself is the same one login makes with a different grant, so it
// lives in bungieOAuth; only carrying the membership across is local.
func (s *TokenStore) refreshBungieToken(tokens *BungieTokens) (*BungieTokens, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	refreshed, err := s.oauth.refreshTokens(ctx, tokens.RefreshToken)
	if err != nil {
		return nil, err
	}
	refreshed.MembershipID = tokens.MembershipID
	return refreshed, nil
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
					slog.Debug("expired in-memory Bungie tokens removed", observability.ID("membership", id))
				}
			}
			s.mu.Unlock()
		}
	}
}
