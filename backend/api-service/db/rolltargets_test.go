package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestRollTargetStore_CRUDAndOwnership(t *testing.T) {
	pool := testPool(t)
	mid, userID := createTestUser(t, pool)
	_, otherID := createTestUser(t, pool)
	store := NewRollTargetStore(pool)
	ctx := context.Background()

	gotID, err := store.GetUserID(ctx, mid)
	if err != nil || gotID != userID {
		t.Fatalf("GetUserID = %d, %v; want %d, nil", gotID, err, userID)
	}

	hash := uint32(1234567890)
	tgt, err := store.Add(ctx, userID, &hash, true, []string{"Kill Clip", "Outlaw"}, "pvp roll")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if tgt.ItemHash == nil || *tgt.ItemHash != 1234567890 || !tgt.Wanted || tgt.Notes != "pvp roll" {
		t.Errorf("Add returned %+v", tgt)
	}
	if len(tgt.Perks) != 2 || tgt.Perks[0] != "Kill Clip" || tgt.Perks[1] != "Outlaw" {
		t.Errorf("perks round-tripped as %v, want [Kill Clip Outlaw]", tgt.Perks)
	}

	// The same roll twice is a duplicate; a different roll on the same weapon
	// is ordinary, and so is the opposite stance on the same perks.
	if _, err := store.Add(ctx, userID, &hash, true, []string{"Kill Clip", "Outlaw"}, ""); !IsDuplicate(err) {
		t.Errorf("identical roll: err = %v, want a duplicate", err)
	}
	other, err := store.Add(ctx, userID, &hash, true, []string{"Rampage"}, "")
	if err != nil {
		t.Fatalf("second roll on the same weapon: %v", err)
	}
	unwanted, err := store.Add(ctx, userID, &hash, false, []string{"Kill Clip", "Outlaw"}, "")
	if err != nil {
		t.Fatalf("opposite stance on the same perks: %v", err)
	}
	if _, err := store.Delete(ctx, userID, other.ID); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if _, err := store.Delete(ctx, userID, unwanted.ID); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	// A nil perks patch leaves the array alone; notes still change.
	notes := "pve roll"
	upd, err := store.Update(ctx, userID, tgt.ID, nil, &notes)
	if err != nil {
		t.Fatalf("Update notes only: %v", err)
	}
	if len(upd.Perks) != 2 || upd.Notes != "pve roll" {
		t.Errorf("notes-only update produced %+v", upd)
	}

	perks := []string{"Rampage"}
	upd, err = store.Update(ctx, userID, tgt.ID, &perks, nil)
	if err != nil {
		t.Fatalf("Update perks: %v", err)
	}
	if len(upd.Perks) != 1 || upd.Perks[0] != "Rampage" || upd.Notes != "pve roll" {
		t.Errorf("perks update produced %+v", upd)
	}

	// Another user's id must not reach this row, for read or write.
	if _, err := store.Update(ctx, otherID, tgt.ID, &perks, nil); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("foreign update err = %v, want pgx.ErrNoRows", err)
	}
	if ok, err := store.Delete(ctx, otherID, tgt.ID); err != nil || ok {
		t.Errorf("foreign delete = %v, %v; want false, nil", ok, err)
	}
	rows, err := store.List(ctx, otherID)
	if err != nil {
		t.Fatalf("List other: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("another user's list returned %d rows", len(rows))
	}

	rows, err = store.List(ctx, userID)
	if err != nil || len(rows) != 1 || rows[0].ID != tgt.ID {
		t.Fatalf("List = %+v, %v", rows, err)
	}

	if ok, err := store.Delete(ctx, userID, tgt.ID); err != nil || !ok {
		t.Fatalf("Delete = %v, %v; want true, nil", ok, err)
	}
	if ok, _ := store.Delete(ctx, userID, tgt.ID); ok {
		t.Error("second delete reported a row")
	}
}

// The table's own constraints are the last line of defence when a caller
// bypasses the service's validation.
func TestRollTargetStore_ConstraintsRejectNonsense(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewRollTargetStore(pool)
	ctx := context.Background()

	h1, h2, h3 := uint32(1), uint32(2), uint32(3)
	if _, err := store.Add(ctx, userID, &h1, true, []string{}, ""); err == nil {
		t.Error("empty perks array was accepted")
	}
	tooMany := make([]string, 11)
	for i := range tooMany {
		tooMany[i] = "p"
	}
	if _, err := store.Add(ctx, userID, &h2, true, tooMany, ""); err == nil {
		t.Error("11 perks were accepted")
	}
	if _, err := store.Add(ctx, userID, &h3, true, []string{"Outlaw"}, strings.Repeat("x", 501)); err == nil {
		t.Error("a 501-character note was accepted")
	}
}

// Roll targets follow the user row out, like every other membership-scoped table.
func TestRollTargetStore_CascadesWithTheUser(t *testing.T) {
	pool := testPool(t)
	mid, userID := createTestUser(t, pool)
	store := NewRollTargetStore(pool)
	ctx := context.Background()

	h := uint32(99)
	if _, err := store.Add(ctx, userID, &h, true, []string{"Outlaw"}, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE membership_id = $1`, mid); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	rows, err := store.List(ctx, userID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("%d roll targets survived their user", len(rows))
	}
}

// An any-weapon target names perks without naming a weapon. NULLS NOT DISTINCT
// is what keeps it deduplicated: without it PostgreSQL reads every NULL as
// unique and a re-imported file would stack wildcard rows without limit.
func TestRollTargetStore_AnyWeaponTargetsDeduplicate(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewRollTargetStore(pool)
	ctx := context.Background()

	first, err := store.Add(ctx, userID, nil, true, []string{"Outlaw"}, "")
	if err != nil {
		t.Fatalf("Add wildcard: %v", err)
	}
	if first.ItemHash != nil {
		t.Errorf("item hash = %v, want nil", first.ItemHash)
	}
	if _, err := store.Add(ctx, userID, nil, true, []string{"Outlaw"}, ""); !IsDuplicate(err) {
		t.Errorf("identical wildcard: err = %v, want a duplicate", err)
	}
	// A different wildcard roll is a different target.
	if _, err := store.Add(ctx, userID, nil, true, []string{"Rampage"}, ""); err != nil {
		t.Errorf("second wildcard roll: %v", err)
	}

	rows, err := store.List(ctx, userID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("List = %d rows, %v; want 2", len(rows), err)
	}
	for _, r := range rows {
		if r.ItemHash != nil {
			t.Errorf("wildcard row came back with a weapon: %+v", r)
		}
	}
}
