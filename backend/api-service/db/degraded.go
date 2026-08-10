package db

import (
	"context"
	"errors"
	"time"
)

// ErrUnavailable is returned by every store method when the service is running
// without a database (no DATABASE_URL). Handlers map it to a single 503
// DB_UNAVAILABLE via handlers.HandleStoreError.
//
// It exists so "the database is absent" travels as an error like any other
// failure, rather than as a nil store that each of twenty call sites had to
// notice and interpret for itself. Three of those call sites — the admin
// handlers — never did notice, and would have panicked on a nil interface if
// their RequireAdmin middleware had not happened to reject the request first.
var ErrUnavailable = errors.New("db: database not configured")

// The store interfaces. Stores holds these rather than concrete pointers so a
// degraded implementation can take their place, which is what makes the nil
// impossible. Consumers keep their own narrower interfaces; these are satisfied
// structurally by both the real stores and the degraded ones.

type UserRepo interface {
	Upsert(ctx context.Context, membershipID string, membershipType int16, displayName string, forceAdmin bool) (id int64, tokenVersion int, role int16, err error)
	GetAuthInfo(ctx context.Context, membershipID string) (tokenVersion int, role int16, found bool, err error)
	GetTokenVersion(ctx context.Context, membershipID string) (int, error)
	GetRole(ctx context.Context, membershipID string) (int16, error)
	BumpTokenVersion(ctx context.Context, membershipID string) error
	CreateSession(ctx context.Context, id, membershipID, jti, userAgent string, expiresAt time.Time) error
	RotateSession(ctx context.Context, id, membershipID, oldJTI, newJTI string, newExpiresAt time.Time) (rotated bool, reused bool, err error)
	DeleteExpiredSessions(ctx context.Context) (int64, error)
	SessionExists(ctx context.Context, id string) (bool, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, membershipID string) error
	SetRole(ctx context.Context, membershipID string, role int16) error
	CountAdmins(ctx context.Context) (int, error)
	ListUsers(ctx context.Context, q string, limit int) ([]AdminUser, error)
	SetRoleByID(ctx context.Context, actorMembershipID string, targetUserID int64, newRole int16) (*RoleChange, error)
}

type TokenRepo interface {
	Get(ctx context.Context, membershipID string) (*EncryptedTokens, error)
	Upsert(ctx context.Context, membershipID string, t *EncryptedTokens, prevUpdatedAt time.Time) (newUpdatedAt time.Time, updated bool, err error)
	Delete(ctx context.Context, membershipID string) error
}

type WishlistRepo interface {
	GetUserID(ctx context.Context, membershipID string) (int64, error)
	List(ctx context.Context, userID int64) ([]WishlistItem, error)
	Add(ctx context.Context, userID int64, hash uint32, prio int16, notes string) (*WishlistItem, error)
	Update(ctx context.Context, userID, id int64, prio *int16, notes *string) (*WishlistItem, error)
	Delete(ctx context.Context, userID, id int64) (bool, error)
	BulkDelete(ctx context.Context, userID int64, ids []int64) (int64, error)
	BulkSetPriority(ctx context.Context, userID int64, ids []int64, prio int16) (int64, error)
}

type PrefsRepo interface {
	Get(ctx context.Context, userID int64) (*UserPreferences, error)
	Upsert(ctx context.Context, userID int64, cardStyle string, personalize, completeOnboarding bool) (*UserPreferences, error)
}

type FlagRepo interface {
	List(ctx context.Context) ([]FeatureFlag, error)
	Get(ctx context.Context, key string) (*FeatureFlag, error)
	Update(ctx context.Context, key string, enabled *bool, minTier *int16, actorUserID *int64, actorMembershipID string) (*FeatureFlag, error)
}

type AuditRepo interface {
	Log(ctx context.Context, ev AuditEvent) error
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
	List(ctx context.Context, f AuditFilter) ([]AuditEntry, string, error)
}

// Pinger is the readiness probe's view of the pool. *pgxpool.Pool satisfies it
// natively.
type Pinger interface {
	Ping(ctx context.Context) error
}

// --- degraded implementations ---
//
// Every method fails with ErrUnavailable. None returns an empty result: an
// empty wishlist and an empty admin roster are both indistinguishable from real
// data, and that ambiguity is the bug class this package is removing.
//
// The two callers that must stay lenient already treat any error as a reason to
// carry on — auth.RequireFlag fails open on a flag it cannot resolve, and
// auth.RevocationChecker fails open rather than locking everyone out — so they
// need no special case here.

type degradedUsers struct{}

func (degradedUsers) Upsert(context.Context, string, int16, string, bool) (int64, int, int16, error) {
	return 0, 0, 0, ErrUnavailable
}
func (degradedUsers) GetAuthInfo(context.Context, string) (int, int16, bool, error) {
	return 0, 0, false, ErrUnavailable
}
func (degradedUsers) GetTokenVersion(context.Context, string) (int, error) {
	return 0, ErrUnavailable
}
func (degradedUsers) GetRole(context.Context, string) (int16, error) { return 0, ErrUnavailable }
func (degradedUsers) BumpTokenVersion(context.Context, string) error { return ErrUnavailable }
func (degradedUsers) CreateSession(context.Context, string, string, string, string, time.Time) error {
	return ErrUnavailable
}
func (degradedUsers) RotateSession(context.Context, string, string, string, string, time.Time) (bool, bool, error) {
	return false, false, ErrUnavailable
}
func (degradedUsers) DeleteExpiredSessions(context.Context) (int64, error) {
	return 0, ErrUnavailable
}
func (degradedUsers) SessionExists(context.Context, string) (bool, error) {
	return false, ErrUnavailable
}
func (degradedUsers) DeleteSession(context.Context, string) error      { return ErrUnavailable }
func (degradedUsers) DeleteUserSessions(context.Context, string) error { return ErrUnavailable }
func (degradedUsers) SetRole(context.Context, string, int16) error     { return ErrUnavailable }
func (degradedUsers) CountAdmins(context.Context) (int, error)         { return 0, ErrUnavailable }
func (degradedUsers) ListUsers(context.Context, string, int) ([]AdminUser, error) {
	return nil, ErrUnavailable
}
func (degradedUsers) SetRoleByID(context.Context, string, int64, int16) (*RoleChange, error) {
	return nil, ErrUnavailable
}

type degradedTokens struct{}

func (degradedTokens) Get(context.Context, string) (*EncryptedTokens, error) {
	return nil, ErrUnavailable
}
func (degradedTokens) Upsert(context.Context, string, *EncryptedTokens, time.Time) (time.Time, bool, error) {
	return time.Time{}, false, ErrUnavailable
}
func (degradedTokens) Delete(context.Context, string) error { return ErrUnavailable }

type degradedWishlist struct{}

func (degradedWishlist) GetUserID(context.Context, string) (int64, error) {
	return 0, ErrUnavailable
}
func (degradedWishlist) List(context.Context, int64) ([]WishlistItem, error) {
	return nil, ErrUnavailable
}
func (degradedWishlist) Add(context.Context, int64, uint32, int16, string) (*WishlistItem, error) {
	return nil, ErrUnavailable
}
func (degradedWishlist) Update(context.Context, int64, int64, *int16, *string) (*WishlistItem, error) {
	return nil, ErrUnavailable
}
func (degradedWishlist) Delete(context.Context, int64, int64) (bool, error) {
	return false, ErrUnavailable
}
func (degradedWishlist) BulkDelete(context.Context, int64, []int64) (int64, error) {
	return 0, ErrUnavailable
}
func (degradedWishlist) BulkSetPriority(context.Context, int64, []int64, int16) (int64, error) {
	return 0, ErrUnavailable
}

type degradedPrefs struct{}

func (degradedPrefs) Get(context.Context, int64) (*UserPreferences, error) {
	return nil, ErrUnavailable
}
func (degradedPrefs) Upsert(context.Context, int64, string, bool, bool) (*UserPreferences, error) {
	return nil, ErrUnavailable
}

type degradedFlags struct{}

func (degradedFlags) List(context.Context) ([]FeatureFlag, error) { return nil, ErrUnavailable }
func (degradedFlags) Get(context.Context, string) (*FeatureFlag, error) {
	return nil, ErrUnavailable
}
func (degradedFlags) Update(context.Context, string, *bool, *int16, *int64, string) (*FeatureFlag, error) {
	return nil, ErrUnavailable
}

type degradedAudit struct{}

func (degradedAudit) Log(context.Context, AuditEvent) error { return ErrUnavailable }
func (degradedAudit) DeleteOlderThan(context.Context, time.Time) (int64, error) {
	return 0, ErrUnavailable
}
func (degradedAudit) List(context.Context, AuditFilter) ([]AuditEntry, string, error) {
	return nil, "", ErrUnavailable
}

type degradedPinger struct{}

func (degradedPinger) Ping(context.Context) error { return ErrUnavailable }
