package db

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrate runs any unapplied SQL migrations from db/migrations/ in order.
// Uses pg_advisory_lock(1952999275) to prevent concurrent runs on multiple replicas.
// 1952999275 is the decimal equivalent of 0x7461726b ("mark").
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("migrate: acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(1952999275)"); err != nil {
		return fmt.Errorf("migrate: acquire advisory lock: %w", err)
	}
	defer conn.Exec(ctx, "SELECT pg_advisory_unlock(1952999275)") //nolint:errcheck

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INT         PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}

	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("migrate: query applied versions: %w", err)
	}
	applied := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}
	rows.Close()

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("migrate: read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		ver, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
		if err != nil {
			return fmt.Errorf("migrate: bad filename %s: %w", e.Name(), err)
		}
		if applied[ver] {
			continue
		}
		sql, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", e.Name(), err)
		}
		// Each migration applies atomically: a failed multi-statement file must not
		// leave a half-applied schema (and must not be recorded as applied).
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrate: begin tx for %s: %w", e.Name(), err)
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			tx.Rollback(ctx) //nolint:errcheck
			return fmt.Errorf("migrate: apply %s: %w", e.Name(), err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", ver); err != nil {
			tx.Rollback(ctx) //nolint:errcheck
			return fmt.Errorf("migrate: record %s: %w", e.Name(), err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", e.Name(), err)
		}
	}
	return nil
}
