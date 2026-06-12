package db

import (
	"context"
	"testing"
)

func TestMigrate_Idempotent(t *testing.T) {
	pool := testPool(t) // testPool already ran Migrate once
	ctx := context.Background()

	// Running again must be a no-op, not an error.
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var applied int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if applied < 1 {
		t.Errorf("schema_migrations rows = %d, want >= 1", applied)
	}

	// Every table from 0001_init.sql must exist.
	for _, table := range []string{"users", "wishlist_items", "user_preferences", "bungie_tokens"} {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, table,
		).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s missing after migrate", table)
		}
	}
}
