package db

import (
	"context"
	"time"

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
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *BungieTokenStore) Upsert(ctx context.Context, membershipID string, t *EncryptedTokens, prevUpdatedAt time.Time) (updated bool, err error) {
	var rows int64
	tag, err := s.pool.Exec(ctx, `
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
				updated_at         = now()
			WHERE bungie_tokens.updated_at = $7`,
		membershipID,
		t.AccessTokenEnc, t.RefreshTokenEnc,
		t.AccessExpiresAt, t.RefreshExpiresAt, t.KeyVersion,
		prevUpdatedAt,
	)
	if err != nil {
		return false, err
	}
	rows = tag.RowsAffected()
	return rows > 0, nil
}

func (s *BungieTokenStore) Delete(ctx context.Context, membershipID string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM bungie_tokens
		WHERE user_id = (SELECT id FROM users WHERE membership_id = $1)`,
		membershipID,
	)
	return err
}
