package db

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditEvent is one entry written to audit_log. Zero-value Outcome defaults to
// "success"; empty IP is stored as NULL; nil Details is stored as '{}'.
type AuditEvent struct {
	EventType         string
	Outcome           string
	ActorUserID       *int64
	ActorMembershipID string
	TargetUserID      *int64
	SessionID         string
	IP                string
	UserAgent         string
	Details           map[string]any
}

// execer is satisfied by both *pgxpool.Pool and pgx.Tx, so insertAudit can run
// standalone (best-effort) or inside a caller's transaction (role/flag changes).
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// insertAudit writes one audit row using the given querier (pool or tx).
func insertAudit(ctx context.Context, q execer, ev AuditEvent) error {
	outcome := ev.Outcome
	if outcome == "" {
		outcome = "success"
	}
	detailsJSON := []byte("{}")
	if ev.Details != nil {
		b, err := json.Marshal(ev.Details)
		if err != nil {
			return err
		}
		detailsJSON = b
	}
	var ip any // nil -> NULL::inet; non-empty -> text cast to inet
	if ev.IP != "" {
		ip = ev.IP
	}
	var sessionID any
	if ev.SessionID != "" {
		sessionID = ev.SessionID
	}
	_, err := q.Exec(ctx, `
		INSERT INTO audit_log
			(event_type, outcome, actor_user_id, actor_membership_id,
			 target_user_id, session_id, ip, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7::inet, $8, $9::jsonb)`,
		ev.EventType, outcome, ev.ActorUserID, ev.ActorMembershipID,
		ev.TargetUserID, sessionID, ip, truncateUserAgent(ev.UserAgent), string(detailsJSON))
	return err
}

// AuditStore is the DB layer for the unified audit trail.
type AuditStore struct{ pool *pgxpool.Pool }

func NewAuditStore(pool *pgxpool.Pool) *AuditStore { return &AuditStore{pool: pool} }

// Log writes one audit event best-effort on the store's own connection. Callers
// that need the write to share a mutation's transaction use insertAudit directly.
func (s *AuditStore) Log(ctx context.Context, ev AuditEvent) error {
	return insertAudit(ctx, s.pool, ev)
}

// DeleteOlderThan removes audit rows created before cutoff, returning the count.
func (s *AuditStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM audit_log WHERE created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// AuditFilter narrows the audit feed. EventType matches exactly, or as a prefix
// when it ends in '.' (e.g. "role." -> role.change.admin + role.optin). Cursor is
// an opaque keyset token returned by a prior List call.
type AuditFilter struct {
	EventType string
	Actor     string // actor membership id
	Target    string // target membership id
	Outcome   string
	After     time.Time
	Before    time.Time
	Cursor    string
	Limit     int
}

// AuditEntry is one row joined to user display names for the admin feed.
type AuditEntry struct {
	ID                 int64
	EventType          string
	Outcome            string
	ActorMembershipID  string
	ActorDisplayName   string
	TargetMembershipID string
	TargetDisplayName  string
	IP                 string
	UserAgent          string
	Details            map[string]any
	CreatedAt          time.Time
}

type auditCursor struct {
	T  time.Time `json:"t"`
	ID int64     `json:"id"`
}

func encodeCursor(t time.Time, id int64) string {
	b, _ := json.Marshal(auditCursor{T: t, ID: id})
	return base64.URLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (auditCursor, bool) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return auditCursor{}, false
	}
	var c auditCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return auditCursor{}, false
	}
	return c, true
}

// List returns audit entries newest-first with keyset pagination. nextCursor is
// non-empty when more rows remain.
func (s *AuditStore) List(ctx context.Context, f AuditFilter) ([]AuditEntry, string, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var conds []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}

	if f.EventType != "" {
		if strings.HasSuffix(f.EventType, ".") {
			add("a.event_type LIKE $%d", f.EventType+"%")
		} else {
			add("a.event_type = $%d", f.EventType)
		}
	}
	if f.Actor != "" {
		add("a.actor_membership_id = $%d", f.Actor)
	}
	if f.Target != "" {
		add("a.target_user_id = (SELECT id FROM users WHERE membership_id = $%d)", f.Target)
	}
	if f.Outcome != "" {
		add("a.outcome = $%d", f.Outcome)
	}
	if !f.After.IsZero() {
		add("a.created_at >= $%d", f.After)
	}
	if !f.Before.IsZero() {
		add("a.created_at <= $%d", f.Before)
	}
	if c, ok := decodeCursor(f.Cursor); ok {
		args = append(args, c.T, c.ID)
		conds = append(conds, fmt.Sprintf("(a.created_at, a.id) < ($%d, $%d)", len(args)-1, len(args)))
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, limit+1) // fetch one extra to detect "more"

	query := fmt.Sprintf(`
		SELECT a.id, a.event_type, a.outcome,
		       a.actor_membership_id, COALESCE(au.display_name, ''),
		       COALESCE(tu.membership_id, ''), COALESCE(tu.display_name, ''),
		       COALESCE(host(a.ip), ''), a.user_agent, a.details, a.created_at
		FROM audit_log a
		LEFT JOIN users au ON au.id = a.actor_user_id
		LEFT JOIN users tu ON tu.id = a.target_user_id
		%s
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $%d`, where, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var detailsRaw []byte
		if err := rows.Scan(&e.ID, &e.EventType, &e.Outcome,
			&e.ActorMembershipID, &e.ActorDisplayName,
			&e.TargetMembershipID, &e.TargetDisplayName,
			&e.IP, &e.UserAgent, &detailsRaw, &e.CreatedAt); err != nil {
			return nil, "", err
		}
		if len(detailsRaw) > 0 {
			_ = json.Unmarshal(detailsRaw, &e.Details)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(out) > limit {
		last := out[limit-1]
		nextCursor = encodeCursor(last.CreatedAt, last.ID)
		out = out[:limit]
	}
	return out, nextCursor, nil
}
