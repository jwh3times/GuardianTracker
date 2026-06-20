package db

import (
	"context"
	"encoding/json"
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
