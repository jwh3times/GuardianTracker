package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestFlagStore_SeedAndList(t *testing.T) {
	pool := testPool(t)
	flags := NewFlagStore(pool)
	list, err := flags.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Migration 0002 seeds the design's flag list.
	if len(list) < 9 {
		t.Fatalf("seeded flags = %d, want >= 9", len(list))
	}
	byKey := map[string]FeatureFlag{}
	for _, f := range list {
		byKey[f.Key] = f
	}
	// Shipped features are seeded enabled at standard tier (no one loses access).
	if f, ok := byKey["global-search"]; !ok || !f.Enabled || f.MinTier != 0 {
		t.Errorf("global-search = %+v, want enabled min_tier=0", f)
	}
	// Not-yet-built features are seeded disabled at a higher tier.
	if f, ok := byKey["god-roll"]; !ok || f.Enabled || f.MinTier != 2 {
		t.Errorf("god-roll = %+v, want disabled min_tier=2", f)
	}
}

func TestFlagStore_Update(t *testing.T) {
	pool := testPool(t)
	flags := NewFlagStore(pool)
	ctx := context.Background()

	// Snapshot then restore so this test doesn't bleed into others sharing the DB.
	orig, err := flags.Get(ctx, "god-roll")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	t.Cleanup(func() {
		_, _ = flags.Update(ctx, "god-roll", &orig.Enabled, &orig.MinTier)
	})

	enabled := true
	var minTier int16 = 1
	updated, err := flags.Update(ctx, "god-roll", &enabled, &minTier)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.Enabled || updated.MinTier != 1 {
		t.Errorf("updated = %+v, want enabled min_tier=1", updated)
	}
	// Partial update: leave enabled, only change min_tier back.
	var mt2 int16 = 2
	updated, err = flags.Update(ctx, "god-roll", nil, &mt2)
	if err != nil {
		t.Fatalf("partial Update: %v", err)
	}
	if !updated.Enabled || updated.MinTier != 2 {
		t.Errorf("partial update = %+v, want enabled min_tier=2", updated)
	}
}

func TestFlagStore_UpdateUnknown(t *testing.T) {
	pool := testPool(t)
	enabled := true
	if _, err := NewFlagStore(pool).Update(context.Background(), "no-such-flag", &enabled, nil); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("Update unknown: got %v, want pgx.ErrNoRows", err)
	}
}
