package db

import (
	"context"
	"log/slog"
	"time"

	"guardian-tracker/api-service/observability"
)

// DefaultPruneInterval is how often the background pruners run in production.
// Tests pass a short interval instead.
const DefaultPruneInterval = time.Hour

// StartSessionPruner periodically deletes expired refresh sessions so abandoned
// sessions (never rotated, never logged out) don't accumulate. It runs until ctx
// is cancelled at shutdown.
//
// A failed sweep is logged and the loop continues: pruning is maintenance, and
// one transient database error must not silently end it for the process's
// lifetime.
func StartSessionPruner(ctx context.Context, users UserRepo, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := users.DeleteExpiredSessions(ctx)
				if err != nil {
					slog.Error("session pruning failed", observability.Err(err))
				} else if n > 0 {
					slog.Info("expired sessions pruned", "count", n)
				}
			}
		}
	}()
}

// StartAuditPruner periodically deletes audit_log rows older than retentionDays,
// bounding table growth and the retention window for stored client IPs. Runs
// until ctx is cancelled at shutdown, surviving errors for the same reason.
func StartAuditPruner(ctx context.Context, audit AuditRepo, retentionDays int, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().AddDate(0, 0, -retentionDays)
				n, err := audit.DeleteOlderThan(ctx, cutoff)
				if err != nil {
					slog.Error("audit pruning failed", observability.Err(err))
				} else if n > 0 {
					slog.Info("expired audit rows pruned", "count", n)
				}
			}
		}
	}()
}
