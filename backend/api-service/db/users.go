package db

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Role tier values stored in users.role (mirrors auth.Role*).
const (
	roleStandard int16 = 0
	roleAdmin    int16 = 3
)

// Session bounds: a per-user cap (oldest-by-use are evicted past it) keeps a single
// user — or a replayed pre-session token — from accumulating unbounded sessions, and
// user_agent is stored truncated to keep a hostile header from bloating the row.
const (
	maxSessionsPerUser = 20
	maxUserAgentRunes  = 256
)

// ErrLastAdmin is returned when a role change would leave the system with no admins.
// ErrUserNotFound is returned when a targeted user row does not exist.
var (
	ErrLastAdmin    = errors.New("db: cannot remove the last admin")
	ErrUserNotFound = errors.New("db: user not found")
)

// User represents a Guardian Tracker user record.
type User struct {
	ID             int64
	MembershipID   string
	MembershipType int16
	DisplayName    string
	Role           int16
	TokenVersion   int
	CreatedAt      time.Time
	LastLoginAt    time.Time
}

// AdminUser is the projection returned to the admin console user list.
type AdminUser struct {
	ID             int64
	MembershipID   string
	MembershipType int16
	DisplayName    string
	Role           int16
	LastLoginAt    time.Time
}

// RoleChange describes the result of an admin-driven role change, carrying the
// target's membership ID so the caller can evict its revocation cache entry.
type RoleChange struct {
	TargetUserID       int64
	TargetMembershipID string
	TargetDisplayName  string
	OldRole            int16
	NewRole            int16
}

// UserStore handles user DB operations.
type UserStore struct{ pool *pgxpool.Pool }

func NewUserStore(pool *pgxpool.Pool) *UserStore { return &UserStore{pool: pool} }

// Upsert inserts or updates the user record and returns current id, token_version,
// and role. When forceAdmin is true (the ADMIN_MEMBERSHIP_IDS bootstrap), the row
// is pinned to admin; otherwise an existing row keeps its current role and a new
// row defaults to standard.
func (s *UserStore) Upsert(ctx context.Context, membershipID string, membershipType int16, displayName string, forceAdmin bool) (id int64, tokenVersion int, role int16, err error) {
	// The admin (3) / standard (0) literals are inlined rather than bound: inside a
	// CASE, pgx can't infer the param type and Postgres would treat it as text.
	err = s.pool.QueryRow(ctx, `
		INSERT INTO users (membership_id, membership_type, display_name, role)
		VALUES ($1, $2, $3, CASE WHEN $4 THEN 3 ELSE 0 END)
		ON CONFLICT (membership_id) DO UPDATE
			SET display_name  = EXCLUDED.display_name,
				last_login_at = now(),
				role          = CASE WHEN $4 THEN 3 ELSE users.role END
		RETURNING id, token_version, role`,
		membershipID, membershipType, displayName, forceAdmin,
	).Scan(&id, &tokenVersion, &role)
	return
}

// GetAuthInfo returns the current token_version and role for one membership.
// Used by the revocation checker, which caches both so role changes propagate
// within the cache window with no token churn (TODO 13.2).
func (s *UserStore) GetAuthInfo(ctx context.Context, membershipID string) (tokenVersion int, role int16, err error) {
	err = s.pool.QueryRow(ctx,
		`SELECT token_version, role FROM users WHERE membership_id = $1`, membershipID,
	).Scan(&tokenVersion, &role)
	return
}

// GetTokenVersion returns the current token_version for revocation checks.
func (s *UserStore) GetTokenVersion(ctx context.Context, membershipID string) (int, error) {
	var v int
	err := s.pool.QueryRow(ctx, `SELECT token_version FROM users WHERE membership_id = $1`, membershipID).Scan(&v)
	return v, err
}

// GetRole returns the current role for one membership.
func (s *UserStore) GetRole(ctx context.Context, membershipID string) (int16, error) {
	var r int16
	err := s.pool.QueryRow(ctx, `SELECT role FROM users WHERE membership_id = $1`, membershipID).Scan(&r)
	return r, err
}

// BumpTokenVersion increments token_version to invalidate all issued JWTs.
func (s *UserStore) BumpTokenVersion(ctx context.Context, membershipID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET token_version = token_version + 1 WHERE membership_id = $1`, membershipID)
	return err
}

// CreateSession records a new per-device refresh session. The user_id is resolved
// from membershipID so callers need only the JWT claim; if no such user exists the
// insert is a no-op (zero rows), and the session simply won't be created. The
// user_agent is bounded, and any sessions beyond maxSessionsPerUser (oldest by
// last_used_at) are evicted in the same transaction so the table can't grow without
// limit from one user re-logging in or replaying a pre-session token.
func (s *UserStore) CreateSession(ctx context.Context, id, membershipID, jti, userAgent string, expiresAt time.Time) error {
	userAgent = truncateUserAgent(userAgent)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful commit

	if _, err := tx.Exec(ctx,
		`INSERT INTO refresh_sessions (id, user_id, membership_id, current_jti, user_agent, expires_at)
		 SELECT $1, u.id, $2, $3, $4, $5 FROM users u WHERE u.membership_id = $2`,
		id, membershipID, jti, userAgent, expiresAt); err != nil {
		return err
	}

	// Keep only the most-recently-used maxSessionsPerUser sessions for this user.
	// The row just inserted has last_used_at = now(), so it is never the one evicted.
	if _, err := tx.Exec(ctx,
		`DELETE FROM refresh_sessions
		 WHERE membership_id = $1 AND id NOT IN (
		     SELECT id FROM refresh_sessions WHERE membership_id = $1
		     ORDER BY last_used_at DESC LIMIT $2
		 )`, membershipID, maxSessionsPerUser); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// truncateUserAgent bounds a client-supplied User-Agent to a sane length and forces
// valid UTF-8 so a hostile or malformed header can neither bloat the row nor break
// the insert (Postgres TEXT rejects invalid UTF-8).
func truncateUserAgent(ua string) string {
	ua = strings.ToValidUTF8(ua, "")
	if utf8.RuneCountInString(ua) > maxUserAgentRunes {
		ua = string([]rune(ua)[:maxUserAgentRunes])
	}
	return ua
}

// RotateSession advances a session's refresh jti from oldJTI to newJTI, with automatic
// reuse detection. It locks the session row, then:
//   - not found / expired   -> (rotated=false, reused=false): caller rejects (re-login)
//   - current_jti != oldJTI -> reuse/theft: the session is DELETED -> (false, reused=true)
//   - current_jti == oldJTI -> rotate (set newJTI, bump last_used_at, slide expires_at
//     forward to newExpiresAt) -> (rotated=true, false)
//
// expires_at is advanced on every rotation to match the freshly issued refresh token,
// so a continuously active session does not hard-expire at its original creation time.
// Deleting the session on reuse revokes the whole family, so once a rotated refresh
// token is replayed neither the attacker's nor the legitimate client's copy works
// without a fresh login. The lock makes two concurrent uses of one token serialize:
// the first rotates, the second sees a mismatched jti and is treated as reuse.
func (s *UserStore) RotateSession(ctx context.Context, id, membershipID, oldJTI, newJTI string, newExpiresAt time.Time) (rotated bool, reused bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful commit

	var currentJTI string
	var expiresAt time.Time
	err = tx.QueryRow(ctx,
		`SELECT current_jti, expires_at FROM refresh_sessions
		 WHERE id = $1 AND membership_id = $2 FOR UPDATE`, id, membershipID,
	).Scan(&currentJTI, &expiresAt)
	if err == pgx.ErrNoRows {
		return false, false, nil // unknown/already-revoked session
	}
	if err != nil {
		return false, false, err
	}
	if time.Now().After(expiresAt) {
		_, derr := tx.Exec(ctx, `DELETE FROM refresh_sessions WHERE id = $1`, id)
		if derr != nil {
			return false, false, derr
		}
		return false, false, tx.Commit(ctx)
	}
	if currentJTI != oldJTI {
		// Replay of a rotated token — revoke the whole session.
		if _, derr := tx.Exec(ctx, `DELETE FROM refresh_sessions WHERE id = $1`, id); derr != nil {
			return false, true, derr
		}
		return false, true, tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE refresh_sessions SET current_jti = $2, last_used_at = now(), expires_at = $3 WHERE id = $1`,
		id, newJTI, newExpiresAt); err != nil {
		return false, false, err
	}
	return true, false, tx.Commit(ctx)
}

// DeleteExpiredSessions removes all sessions past their expiry. Run periodically so
// abandoned (never-rotated, never-logged-out) sessions don't accumulate. Returns the
// number of rows removed.
func (s *UserStore) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM refresh_sessions WHERE expires_at <= now()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// SessionExists reports whether a non-expired session with this id exists. The
// access-token revocation check uses it so ending one session (this-device logout)
// invalidates that device's access token within the cache window.
func (s *UserStore) SessionExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM refresh_sessions WHERE id = $1 AND expires_at > now())`, id,
	).Scan(&exists)
	return exists, err
}

// DeleteSession ends a single session (this-device logout).
func (s *UserStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM refresh_sessions WHERE id = $1`, id)
	return err
}

// DeleteUserSessions ends every session for a user (sign-out-everywhere).
func (s *UserStore) DeleteUserSessions(ctx context.Context, membershipID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM refresh_sessions WHERE membership_id = $1`, membershipID)
	return err
}

// SetRole sets a user's role by membership ID (self opt-in path). The token
// version is intentionally left untouched so opt-in does not log the user out;
// the caller evicts the revocation cache entry so the change takes effect on the
// next request. Returns ErrUserNotFound when no row matches.
func (s *UserStore) SetRole(ctx context.Context, membershipID string, role int16) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET role = $2 WHERE membership_id = $1`, membershipID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// CountAdmins returns the number of users currently holding the admin role.
func (s *UserStore) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE role = $1`, roleAdmin).Scan(&n)
	return n, err
}

// ListUsers returns users for the admin console, newest-active first. When q is
// non-empty it filters on a case-insensitive display-name substring.
func (s *UserStore) ListUsers(ctx context.Context, q string, limit int) ([]AdminUser, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, membership_id, membership_type, display_name, role, last_login_at
		FROM users
		WHERE ($1 = '' OR display_name ILIKE '%' || $1 || '%')
		ORDER BY last_login_at DESC
		LIMIT $2`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminUser
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.MembershipID, &u.MembershipType, &u.DisplayName, &u.Role, &u.LastLoginAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetRoleByID performs an admin-driven role change inside a single transaction:
// it locks the target (and, when demoting an admin, all admin rows) to enforce
// last-admin protection, bumps the target's token_version, and writes a
// role_audit row. The caller evicts the target's revocation cache entry using the
// returned membership ID. actorMembershipID identifies the acting admin for audit.
//
// Returns ErrUserNotFound (unknown target), ErrLastAdmin (would orphan the
// system with zero admins), or the applied change.
func (s *UserStore) SetRoleByID(ctx context.Context, actorMembershipID string, targetUserID int64, newRole int16) (*RoleChange, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful commit

	// Resolve the acting admin's id for the audit row (nullable — never blocks the change).
	var actorID *int64
	if actorMembershipID != "" {
		var aid int64
		if err := tx.QueryRow(ctx, `SELECT id FROM users WHERE membership_id = $1`, actorMembershipID).Scan(&aid); err == nil {
			actorID = &aid
		}
	}

	var oldRole int16
	var targetMembership, targetDisplay string
	err = tx.QueryRow(ctx,
		`SELECT role, membership_id, display_name FROM users WHERE id = $1 FOR UPDATE`, targetUserID,
	).Scan(&oldRole, &targetMembership, &targetDisplay)
	if err == pgx.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	change := &RoleChange{
		TargetUserID:       targetUserID,
		TargetMembershipID: targetMembership,
		TargetDisplayName:  targetDisplay,
		OldRole:            oldRole,
		NewRole:            newRole,
	}
	if oldRole == newRole {
		// No-op change — commit nothing, report success.
		return change, tx.Commit(ctx)
	}

	// Last-admin protection: lock every admin row so two concurrent demotions
	// can't both observe count>1 and race the system down to zero admins.
	if oldRole == roleAdmin && newRole != roleAdmin {
		rows, err := tx.Query(ctx, `SELECT id FROM users WHERE role = $1 FOR UPDATE`, roleAdmin)
		if err != nil {
			return nil, err
		}
		admins := 0
		for rows.Next() {
			admins++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if admins <= 1 {
			return nil, ErrLastAdmin
		}
	}

	// Bump token_version so the target's stale sessions re-sync immediately
	// (admin path only; self opt-in leaves token_version untouched).
	if _, err := tx.Exec(ctx,
		`UPDATE users SET role = $2, token_version = token_version + 1 WHERE id = $1`,
		targetUserID, newRole); err != nil {
		return nil, err
	}
	if err := insertAudit(ctx, tx, AuditEvent{
		EventType:    "role.change.admin",
		ActorUserID:  actorID,
		TargetUserID: &targetUserID,
		Details:      map[string]any{"oldRole": oldRole, "newRole": newRole},
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return change, nil
}
