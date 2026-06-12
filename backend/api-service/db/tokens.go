package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EncryptedTokens is the DB representation of Bungie OAuth tokens.
type EncryptedTokens struct {
	AccessTokenEnc   []byte
	RefreshTokenEnc  []byte
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	KeyVersion       int16
	UpdatedAt        time.Time
}

// ErrTokensNotFound is returned by Get when no token row exists for the membership.
// ErrNoUserRow is returned by Upsert when no users row exists, so the caller can
// distinguish "user record missing" (tokens can't persist) from a lost CAS race.
var (
	ErrTokensNotFound = errors.New("db: bungie tokens not found")
	ErrNoUserRow      = errors.New("db: no users row for membership")
)

// BungieTokenStore persists encrypted Bungie OAuth tokens.
type BungieTokenStore struct{ pool *pgxpool.Pool }

func NewBungieTokenStore(pool *pgxpool.Pool) *BungieTokenStore {
	return &BungieTokenStore{pool: pool}
}

func (s *BungieTokenStore) Get(ctx context.Context, membershipID string) (*EncryptedTokens, error) {
	var t EncryptedTokens
	err := s.pool.QueryRow(ctx, `
		SELECT bt.access_token_enc, bt.refresh_token_enc,
		       bt.access_expires_at, bt.refresh_expires_at,
		       bt.key_version, bt.updated_at
		FROM bungie_tokens bt
		JOIN users u ON u.id = bt.user_id
		WHERE u.membership_id = $1`, membershipID,
	).Scan(&t.AccessTokenEnc, &t.RefreshTokenEnc, &t.AccessExpiresAt, &t.RefreshExpiresAt, &t.KeyVersion, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokensNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Upsert writes the tokens for membershipID and returns the row's new updated_at.
//
// A zero prevUpdatedAt means "unconditional write" (login path). A non-zero
// prevUpdatedAt is a compare-and-swap: the update only applies if the row's
// updated_at still equals it (refresh path — a lost race means another replica
// already stored a newer token, and the caller should re-read the winner's row).
//
// updated == false with a nil error means the CAS lost (the row changed under
// us). A missing users row instead returns ErrNoUserRow, so the caller can tell
// "tokens can't persist yet" apart from "another replica wrote first".
func (s *BungieTokenStore) Upsert(ctx context.Context, membershipID string, t *EncryptedTokens, prevUpdatedAt time.Time) (newUpdatedAt time.Time, updated bool, err error) {
	conflictCond := ""
	args := []any{
		membershipID,
		t.AccessTokenEnc, t.RefreshTokenEnc,
		t.AccessExpiresAt, t.RefreshExpiresAt, t.KeyVersion,
	}
	if !prevUpdatedAt.IsZero() {
		conflictCond = " WHERE bungie_tokens.updated_at = $7"
		args = append(args, prevUpdatedAt)
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO bungie_tokens (user_id, access_token_enc, refresh_token_enc,
			access_expires_at, refresh_expires_at, key_version, updated_at)
		SELECT u.id, $2, $3, $4, $5, $6, now()
		FROM users u WHERE u.membership_id = $1
		ON CONFLICT (user_id) DO UPDATE
			SET access_token_enc   = EXCLUDED.access_token_enc,
				refresh_token_enc  = EXCLUDED.refresh_token_enc,
				access_expires_at  = EXCLUDED.access_expires_at,
				refresh_expires_at = EXCLUDED.refresh_expires_at,
				key_version        = EXCLUDED.key_version,
				updated_at         = now()`+conflictCond+`
		RETURNING updated_at`,
		args...,
	).Scan(&newUpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// No row was inserted/updated: either the CAS predicate didn't match an
		// existing row (lost race) or there's no users row at all. Disambiguate.
		var exists bool
		if e := s.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM users WHERE membership_id = $1)`, membershipID,
		).Scan(&exists); e == nil && !exists {
			return time.Time{}, false, ErrNoUserRow
		}
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return newUpdatedAt, true, nil
}

func (s *BungieTokenStore) Delete(ctx context.Context, membershipID string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM bungie_tokens
		WHERE user_id = (SELECT id FROM users WHERE membership_id = $1)`,
		membershipID,
	)
	return err
}
