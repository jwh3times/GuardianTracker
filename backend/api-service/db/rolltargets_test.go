package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestRollTargetStore_ImportProvenanceSurvivesEditsAndDuplicates(t *testing.T) {
	pool := testPool(t)
	_, userID := createTestUser(t, pool)
	store := NewRollTargetStore(pool)
	ctx := context.Background()
	const firstID = "11111111-1111-4111-8111-111111111111"
	const secondID = "22222222-2222-4222-8222-222222222222"
	first, err := store.Add(ctx, userID, nil, true, []string{"Outlaw"}, "", firstID, "My DIM rolls")
	if err != nil {
		t.Fatal(err)
	}
	if first.ImportID != firstID || first.ImportTitle != "My DIM rolls" {
		t.Fatalf("provenance = %+v", first)
	}
	if _, err := store.Add(ctx, userID, nil, true, []string{"Outlaw"}, "", secondID, "New title"); !IsDuplicate(err) {
		t.Fatalf("duplicate = %v", err)
	}
	notes, perks := "edited", []string{"Rampage"}
	updated, err := store.Update(ctx, userID, first.ID, &perks, &notes)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ImportID != firstID || updated.ImportTitle != "My DIM rolls" {
		t.Fatalf("updated provenance = %+v", updated)
	}
	rows, err := store.List(ctx, userID)
	if err != nil || len(rows) != 1 || rows[0].ImportID != firstID || rows[0].ImportTitle != "My DIM rolls" {
		t.Fatalf("List = %+v, %v", rows, err)
	}
}

func TestRollTargetStore_DeleteImportIsScopedAndUncapped(t *testing.T) {
	pool := testPool(t)
	_, me := createTestUser(t, pool)
	_, other := createTestUser(t, pool)
	s := NewRollTargetStore(pool)
	ctx := context.Background()
	const firstID = "11111111-1111-4111-8111-111111111111"
	const secondID = "22222222-2222-4222-8222-222222222222"
	for i := uint32(1); i <= 101; i++ {
		if _, err := s.Add(ctx, me, &i, true, []string{"Outlaw"}, "", firstID, "Same title"); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []struct {
		user      int64
		id, title string
	}{{me, secondID, "Same title"}, {me, "", ""}, {other, firstID, "Same title"}} {
		if _, err := s.Add(ctx, row.user, nil, row.id != "", []string{"Outlaw"}, "", row.id, row.title); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := s.DeleteImport(ctx, other, secondID); err != nil || n != 0 {
		t.Fatalf("foreign import = %d, %v", n, err)
	}
	if n, err := s.DeleteImport(ctx, me, firstID); err != nil || n != 101 {
		t.Fatalf("delete = %d, %v", n, err)
	}
	if n, err := s.DeleteImport(ctx, me, firstID); err != nil || n != 0 {
		t.Fatalf("missing import = %d, %v", n, err)
	}
	rows, err := s.List(ctx, me)
	if err != nil || len(rows) != 2 {
		t.Fatalf("remaining = %+v, %v", rows, err)
	}
	for _, row := range rows {
		if row.ImportID != "" && row.ImportID != secondID {
			t.Fatalf("unexpected survivor %+v", row)
		}
	}
	rows, err = s.List(ctx, other)
	if err != nil || len(rows) != 1 || rows[0].ImportID != firstID {
		t.Fatalf("foreign remaining = %+v, %v", rows, err)
	}
}

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
	tgt, err := store.Add(ctx, userID, &hash, true, []string{"Kill Clip", "Outlaw"}, "pvp roll", "", "")
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
	if _, err := store.Add(ctx, userID, &hash, true, []string{"Kill Clip", "Outlaw"}, "", "", ""); !IsDuplicate(err) {
		t.Errorf("identical roll: err = %v, want a duplicate", err)
	}
	other, err := store.Add(ctx, userID, &hash, true, []string{"Rampage"}, "", "", "")
	if err != nil {
		t.Fatalf("second roll on the same weapon: %v", err)
	}
	unwanted, err := store.Add(ctx, userID, &hash, false, []string{"Kill Clip", "Outlaw"}, "", "", "")
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
	if _, err := store.Add(ctx, userID, &h1, true, []string{}, "", "", ""); err == nil {
		t.Error("empty perks array was accepted")
	}
	tooMany := make([]string, 11)
	for i := range tooMany {
		tooMany[i] = "p"
	}
	if _, err := store.Add(ctx, userID, &h2, true, tooMany, "", "", ""); err == nil {
		t.Error("11 perks were accepted")
	}
	if _, err := store.Add(ctx, userID, &h3, true, []string{"Outlaw"}, strings.Repeat("x", 501), "", ""); err == nil {
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
	if _, err := store.Add(ctx, userID, &h, true, []string{"Outlaw"}, "", "", ""); err != nil {
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

func TestRollTargetStore_BulkDelete_OwnershipScoped(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := NewRollTargetStore(pool)

	_, me := createTestUser(t, pool)
	_, other := createTestUser(t, pool)
	h1, h2, h3 := uint32(1001), uint32(1002), uint32(1003)
	a, _ := s.Add(ctx, me, &h1, true, []string{"Outlaw"}, "", "", "")
	b, _ := s.Add(ctx, me, &h2, true, []string{"Outlaw"}, "", "", "")
	foreign, _ := s.Add(ctx, other, &h3, true, []string{"Outlaw"}, "", "", "")

	// Delete two owned + one foreign id; only the two owned are removed.
	removed, err := s.BulkDelete(ctx, me, []int64{a.ID, b.ID, foreign.ID})
	if err != nil {
		t.Fatalf("BulkDelete: %v", err)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2 (foreign id skipped)", removed)
	}
	remaining, _ := s.List(ctx, me)
	if len(remaining) != 0 {
		t.Errorf("owned targets remaining = %d, want 0", len(remaining))
	}
	stillForeign, _ := s.List(ctx, other)
	if len(stillForeign) != 1 {
		t.Errorf("foreign target wrongly deleted; remaining = %d, want 1", len(stillForeign))
	}
}

func TestRollTargetStore_BulkDelete_EmptyIDs(t *testing.T) {
	pool := testPool(t)
	removed, err := NewRollTargetStore(pool).BulkDelete(context.Background(), 1, []int64{})
	if err != nil || removed != 0 {
		t.Fatalf("empty ids: removed=%d err=%v, want 0, nil", removed, err)
	}
}

func TestRollTargetStore_DeleteAll_OwnershipScoped(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	s := NewRollTargetStore(pool)

	_, me := createTestUser(t, pool)
	_, other := createTestUser(t, pool)
	h1, h2, h3 := uint32(2001), uint32(2002), uint32(2003)
	if _, err := s.Add(ctx, me, &h1, true, []string{"Outlaw"}, "", "", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := s.Add(ctx, me, &h2, true, []string{"Rampage"}, "", "", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := s.Add(ctx, other, &h3, true, []string{"Outlaw"}, "", "", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	removed, err := s.DeleteAll(ctx, me)
	if err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}
	mine, _ := s.List(ctx, me)
	if len(mine) != 0 {
		t.Errorf("owner's targets remaining = %d, want 0", len(mine))
	}
	theirs, _ := s.List(ctx, other)
	if len(theirs) != 1 {
		t.Errorf("another user's target was deleted; remaining = %d, want 1", len(theirs))
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

	first, err := store.Add(ctx, userID, nil, true, []string{"Outlaw"}, "", "", "")
	if err != nil {
		t.Fatalf("Add wildcard: %v", err)
	}
	if first.ItemHash != nil {
		t.Errorf("item hash = %v, want nil", first.ItemHash)
	}
	if _, err := store.Add(ctx, userID, nil, true, []string{"Outlaw"}, "", "", ""); !IsDuplicate(err) {
		t.Errorf("identical wildcard: err = %v, want a duplicate", err)
	}
	// A different wildcard roll is a different target.
	if _, err := store.Add(ctx, userID, nil, true, []string{"Rampage"}, "", "", ""); err != nil {
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
