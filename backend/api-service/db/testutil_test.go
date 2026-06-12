package db

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testUserSeq atomic.Int64

// testPool connects to TEST_DATABASE_URL and ensures migrations are applied.
// Integration tests are skipped when TEST_DATABASE_URL is unset (CI exports it;
// locally: docker compose up -d postgres + point at it).
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping db integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect %s: %v", url, err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// createTestUser inserts a user with a unique membership id; the row (and its
// wishlist/token rows, via ON DELETE CASCADE) is removed at test cleanup.
func createTestUser(t *testing.T, pool *pgxpool.Pool) (membershipID string, userID int64) {
	t.Helper()
	membershipID = fmt.Sprintf("test-%d-%d", time.Now().UnixNano(), testUserSeq.Add(1))
	users := NewUserStore(pool)
	userID, _, err := users.Upsert(context.Background(), membershipID, 3, "Test Guardian")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE membership_id = $1`, membershipID)
	})
	return membershipID, userID
}
