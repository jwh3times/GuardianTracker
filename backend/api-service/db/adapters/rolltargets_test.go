package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/rolltargets"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeRollTargetStore records what reached storage and returns what the test
// tells it to. It embeds db.RollTargetRepo so any method a test does not stub
// panics rather than quietly succeeding.
type fakeRollTargetStore struct {
	db.RollTargetRepo

	userID  int64
	userErr error
	rows    []db.RollTarget
	row     *db.RollTarget
	err     error
	found   bool

	gotUserID int64
	gotID     int64
	gotHash   uint32
	gotPerks  []string
	gotNotes  string
	gotPatch  *[]string
}

func (f *fakeRollTargetStore) GetUserID(context.Context, string) (int64, error) {
	return f.userID, f.userErr
}

func (f *fakeRollTargetStore) List(_ context.Context, userID int64) ([]db.RollTarget, error) {
	f.gotUserID = userID
	return f.rows, f.err
}

func (f *fakeRollTargetStore) Add(_ context.Context, userID int64, hash uint32, perks []string, notes string) (*db.RollTarget, error) {
	f.gotUserID, f.gotHash, f.gotPerks, f.gotNotes = userID, hash, perks, notes
	return f.row, f.err
}

func (f *fakeRollTargetStore) Update(_ context.Context, userID, id int64, perks *[]string, _ *string) (*db.RollTarget, error) {
	f.gotUserID, f.gotID, f.gotPatch = userID, id, perks
	return f.row, f.err
}

func (f *fakeRollTargetStore) Delete(_ context.Context, userID, id int64) (bool, error) {
	f.gotUserID, f.gotID = userID, id
	return f.found, f.err
}

func TestRollTargetRepository_ResolvesMembershipToUserID(t *testing.T) {
	store := &fakeRollTargetStore{userID: 42, rows: []db.RollTarget{
		{ID: 1, ItemHash: 1000, Perks: []string{"Outlaw"}, Notes: "n", CreatedAt: time.Unix(5, 0)},
	}}
	got, err := NewRollTargetRepository(store).List(context.Background(), "membership-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if store.gotUserID != 42 {
		t.Errorf("store saw user id %d, want 42", store.gotUserID)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].ItemHash != 1000 || got[0].Perks[0] != "Outlaw" {
		t.Errorf("targets = %+v", got)
	}
}

// A unique-constraint violation is a domain answer, not a driver detail.
func TestRollTargetRepository_DuplicateBecomesDomainError(t *testing.T) {
	store := &fakeRollTargetStore{userID: 1, err: &pgconn.PgError{Code: "23505"}}
	_, err := NewRollTargetRepository(store).Add(context.Background(), "m", rolltargets.AddCommand{
		ItemHash: 1000, Perks: []string{"Outlaw"},
	})
	if !errors.Is(err, rolltargets.ErrDuplicate) {
		t.Fatalf("err = %v, want ErrDuplicate", err)
	}
}

func TestRollTargetRepository_MissingRowBecomesNotFound(t *testing.T) {
	store := &fakeRollTargetStore{userID: 1, err: pgx.ErrNoRows}
	perks := []string{"Outlaw"}
	_, err := NewRollTargetRepository(store).Update(context.Background(), "m", 7,
		rolltargets.UpdateCommand{Perks: &perks})
	if !errors.Is(err, rolltargets.ErrNotFound) {
		t.Fatalf("update err = %v, want ErrNotFound", err)
	}

	store = &fakeRollTargetStore{userID: 1, found: false}
	if err := NewRollTargetRepository(store).Remove(context.Background(), "m", 7); !errors.Is(err, rolltargets.ErrNotFound) {
		t.Fatalf("remove err = %v, want ErrNotFound", err)
	}
}

func TestRollTargetRepository_NoDatabaseBecomesUnavailable(t *testing.T) {
	// Every entry point must translate it, including the identity lookup that
	// runs before the store call.
	store := &fakeRollTargetStore{userErr: db.ErrUnavailable}
	repo := NewRollTargetRepository(store)
	if _, err := repo.List(context.Background(), "m"); !errors.Is(err, rolltargets.ErrUnavailable) {
		t.Errorf("List err = %v, want ErrUnavailable", err)
	}
	if _, err := repo.Add(context.Background(), "m", rolltargets.AddCommand{}); !errors.Is(err, rolltargets.ErrUnavailable) {
		t.Errorf("Add err = %v, want ErrUnavailable", err)
	}
	if err := repo.Remove(context.Background(), "m", 1); !errors.Is(err, rolltargets.ErrUnavailable) {
		t.Errorf("Remove err = %v, want ErrUnavailable", err)
	}
}

// A real read failure must not be dressed up as "no database" — that would tell
// a user their saved targets are simply unavailable when the read broke.
func TestRollTargetRepository_RealFailuresPassThrough(t *testing.T) {
	boom := errors.New("connection reset")
	store := &fakeRollTargetStore{userID: 1, err: boom}
	_, err := NewRollTargetRepository(store).List(context.Background(), "m")
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the original failure", err)
	}
	if errors.Is(err, rolltargets.ErrUnavailable) {
		t.Error("a real failure was reported as unavailable")
	}
}
