// Package adapters translates db stores into the consumer-side interfaces that
// auth and weekly declare.
//
// It exists because those packages must not import db — each declares the
// narrow interface it needs and the wiring supplies something satisfying it.
// The translation is not mechanical: db and auth have separate sentinel errors
// for the same conditions, and mapping between them is load-bearing. See
// TokenRepo.Get.
//
// These lived in main.go, where nothing could reach them.
package adapters

import (
	"context"
	"errors"
	"time"

	"guardian-tracker/api-service/auth"
	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/weekly"
)

// tokenRepo adapts a db.TokenRepo to auth.TokenRepo.
type tokenRepo struct{ s db.TokenRepo }

// NewTokenRepo wraps a db token store for auth.TokenStore.
func NewTokenRepo(s db.TokenRepo) auth.TokenRepo { return &tokenRepo{s: s} }

// Get translates db.ErrTokensNotFound to auth.ErrTokensNotFound.
//
// That translation is load-bearing, not cosmetic. auth.TokenStore's CAS
// reconciliation treats "definitively absent" and "the read failed" as opposite
// cases: the first is safe to overwrite, the second must not be, because the
// row may exist with a newer value. Any other error must therefore pass through
// untranslated — reporting a transient failure as "not found" would let the
// store clobber a concurrent replica's refresh.
func (a *tokenRepo) Get(ctx context.Context, membershipID string) (*auth.EncryptedTokenRecord, error) {
	t, err := a.s.Get(ctx, membershipID)
	if err != nil {
		if errors.Is(err, db.ErrTokensNotFound) {
			return nil, auth.ErrTokensNotFound
		}
		return nil, err
	}
	return &auth.EncryptedTokenRecord{
		AccessTokenEnc:   t.AccessTokenEnc,
		RefreshTokenEnc:  t.RefreshTokenEnc,
		AccessExpiresAt:  t.AccessExpiresAt,
		RefreshExpiresAt: t.RefreshExpiresAt,
		KeyVersion:       t.KeyVersion,
		UpdatedAt:        t.UpdatedAt,
	}, nil
}

// Upsert translates db.ErrNoUserRow, which tells the caller the write failed
// because the user row is gone rather than because the CAS lost a race.
func (a *tokenRepo) Upsert(ctx context.Context, membershipID string, t *auth.EncryptedTokenRecord, prev time.Time) (time.Time, bool, error) {
	at, ok, err := a.s.Upsert(ctx, membershipID, &db.EncryptedTokens{
		AccessTokenEnc:   t.AccessTokenEnc,
		RefreshTokenEnc:  t.RefreshTokenEnc,
		AccessExpiresAt:  t.AccessExpiresAt,
		RefreshExpiresAt: t.RefreshExpiresAt,
		KeyVersion:       t.KeyVersion,
	}, prev)
	if errors.Is(err, db.ErrNoUserRow) {
		return at, ok, auth.ErrNoUserRow
	}
	return at, ok, err
}

func (a *tokenRepo) Delete(ctx context.Context, membershipID string) error {
	return a.s.Delete(ctx, membershipID)
}

// sessionStore adapts a db.UserRepo to auth.SessionStore.
//
// Its whole job is one translation, applied uniformly: db.ErrUnavailable
// becomes auth.ErrUnavailable on every method, so auth can tell "there is no
// database" from "the write failed" without importing db. That distinction
// decides whether a failed session write fails the login — see
// auth.SessionIssuer.recordSession.
type sessionStore struct{ s db.UserRepo }

// NewSessionStore wraps a db user store for auth.SessionIssuer.
func NewSessionStore(s db.UserRepo) auth.SessionStore { return &sessionStore{s: s} }

// unavailable translates the one sentinel this seam carries. Everything else
// passes through: reporting a real failure as "no database" would let a login
// succeed with a session row that was supposed to exist and does not.
func unavailable(err error) error {
	if errors.Is(err, db.ErrUnavailable) {
		return auth.ErrUnavailable
	}
	return err
}

func (a *sessionStore) Upsert(ctx context.Context, membershipID string, membershipType int16, displayName string, forceAdmin bool) (int64, int, int16, error) {
	id, tv, role, err := a.s.Upsert(ctx, membershipID, membershipType, displayName, forceAdmin)
	return id, tv, role, unavailable(err)
}

func (a *sessionStore) CreateSession(ctx context.Context, id, membershipID, jti, userAgent string, expiresAt time.Time) error {
	return unavailable(a.s.CreateSession(ctx, id, membershipID, jti, userAgent, expiresAt))
}

func (a *sessionStore) RotateSession(ctx context.Context, id, membershipID, oldJTI, newJTI string, newExpiresAt time.Time) (bool, bool, error) {
	rotated, reused, err := a.s.RotateSession(ctx, id, membershipID, oldJTI, newJTI, newExpiresAt)
	return rotated, reused, unavailable(err)
}

func (a *sessionStore) DeleteSession(ctx context.Context, id string) error {
	return unavailable(a.s.DeleteSession(ctx, id))
}

func (a *sessionStore) DeleteUserSessions(ctx context.Context, membershipID string) error {
	return unavailable(a.s.DeleteUserSessions(ctx, membershipID))
}

func (a *sessionStore) BumpTokenVersion(ctx context.Context, membershipID string) error {
	return unavailable(a.s.BumpTokenVersion(ctx, membershipID))
}

// WishlistLister is the two-method slice of db.WishlistRepo this adapter reads.
// Narrow on purpose: weekly only lists, and depending on the full seven-method
// repo would make every stand-in implement five methods it never calls.
type WishlistLister interface {
	GetUserID(ctx context.Context, membershipID string) (int64, error)
	List(ctx context.Context, userID int64) ([]db.WishlistItem, error)
}

// weeklyWishlist adapts a wishlist store to weekly.WishlistReader, projecting
// the stored rows down to the one field weekly reads.
type weeklyWishlist struct{ s WishlistLister }

// NewWeeklyWishlist wraps a db wishlist store for weekly.Service.
func NewWeeklyWishlist(s WishlistLister) weekly.WishlistReader { return &weeklyWishlist{s: s} }

func (a *weeklyWishlist) GetUserID(ctx context.Context, membershipID string) (int64, error) {
	return a.s.GetUserID(ctx, membershipID)
}

func (a *weeklyWishlist) List(ctx context.Context, userID int64) ([]weekly.WishlistItem, error) {
	items, err := a.s.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]weekly.WishlistItem, len(items))
	for i, it := range items {
		out[i] = weekly.WishlistItem{ItemHash: it.ItemHash}
	}
	return out, nil
}
