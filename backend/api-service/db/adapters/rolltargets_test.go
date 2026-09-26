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

	userID   int64
	userErr  error
	rows     []db.RollTarget
	row      *db.RollTarget
	err      error
	found    bool
	affected int64

	gotUserID      int64
	gotID          int64
	gotHash        *uint32
	gotWanted      bool
	gotPerks       []string
	gotNotes       string
	gotPatch       *[]string
	gotIDs         []int64
	gotImportID    string
	gotImportTitle string
}

func (f *fakeRollTargetStore) GetUserID(context.Context, string) (int64, error) {
	return f.userID, f.userErr
}

func (f *fakeRollTargetStore) List(_ context.Context, userID int64) ([]db.RollTarget, error) {
	f.gotUserID = userID
	return f.rows, f.err
}

func (f *fakeRollTargetStore) Add(_ context.Context, userID int64, hash *uint32, wanted bool, perks []string, notes, importID, importTitle string) (*db.RollTarget, error) {
	f.gotUserID, f.gotHash, f.gotWanted, f.gotPerks, f.gotNotes = userID, hash, wanted, perks, notes
	f.gotImportID, f.gotImportTitle = importID, importTitle
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

func (f *fakeRollTargetStore) BulkDelete(_ context.Context, userID int64, ids []int64) (int64, error) {
	f.gotUserID, f.gotIDs = userID, ids
	return f.affected, f.err
}

func (f *fakeRollTargetStore) DeleteAll(_ context.Context, userID int64) (int64, error) {
	f.gotUserID = userID
	return f.affected, f.err
}

func (f *fakeRollTargetStore) DeleteImport(_ context.Context, userID int64, importID string) (int64, error) {
	f.gotUserID, f.gotImportID = userID, importID
	return f.affected, f.err
}

func TestRollTargetRepository_ImportProvenance(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	store := &fakeRollTargetStore{userID: 42, affected: 101, row: &db.RollTarget{ID: 1, ImportID: id, ImportTitle: "DIM title"}}
	repo := NewRollTargetRepository(store)
	got, err := repo.Add(context.Background(), "m", rolltargets.AddCommand{ImportID: id, ImportTitle: "DIM title"})
	if err != nil || got.ImportID != id || got.ImportTitle != "DIM title" || store.gotImportID != id || store.gotImportTitle != "DIM title" || store.gotUserID != 42 {
		t.Fatalf("Add = %+v, %v; store = %+v", got, err, store)
	}
	deleted, err := repo.RemoveImport(context.Background(), "m", id)
	if err != nil || deleted != 101 || store.gotImportID != id || store.gotUserID != 42 {
		t.Fatalf("RemoveImport = %d, %v; store = %+v", deleted, err, store)
	}
	store.err = db.ErrUnavailable
	if _, err := repo.RemoveImport(context.Background(), "m", id); !errors.Is(err, rolltargets.ErrUnavailable) {
		t.Fatalf("store error = %v", err)
	}
	store.userErr = db.ErrUnavailable
	if _, err := repo.RemoveImport(context.Background(), "m", id); !errors.Is(err, rolltargets.ErrUnavailable) {
		t.Fatalf("lookup error = %v", err)
	}
	if _, err := NewRollTargetRepository(db.NewStores(nil).RollTargets).RemoveImport(context.Background(), "m", id); !errors.Is(err, rolltargets.ErrUnavailable) {
		t.Fatalf("degraded error = %v", err)
	}
}

func TestRollTargetRepository_ResolvesMembershipToUserID(t *testing.T) {
	hash := uint32(1000)
	store := &fakeRollTargetStore{userID: 42, rows: []db.RollTarget{
		{ID: 1, ItemHash: &hash, Wanted: true, Perks: []string{"Outlaw"}, Notes: "n", CreatedAt: time.Unix(5, 0)},
	}}
	got, err := NewRollTargetRepository(store).List(context.Background(), "membership-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if store.gotUserID != 42 {
		t.Errorf("store saw user id %d, want 42", store.gotUserID)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].ItemHash == nil || *got[0].ItemHash != 1000 {
		t.Errorf("targets = %+v", got)
	}
	if !got[0].Wanted || got[0].Perks[0] != "Outlaw" {
		t.Errorf("target = %+v", got[0])
	}
}

// A unique-constraint violation is a domain answer, not a driver detail.
func TestRollTargetRepository_DuplicateBecomesDomainError(t *testing.T) {
	store := &fakeRollTargetStore{userID: 1, err: &pgconn.PgError{Code: "23505"}}
	hash := uint32(1000)
	_, err := NewRollTargetRepository(store).Add(context.Background(), "m", rolltargets.AddCommand{
		ItemHash: &hash, Wanted: true, Perks: []string{"Outlaw"},
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
	if _, err := repo.RemoveMany(context.Background(), "m", []rolltargets.TargetID{1}); !errors.Is(err, rolltargets.ErrUnavailable) {
		t.Errorf("RemoveMany err = %v, want ErrUnavailable", err)
	}
	if _, err := repo.RemoveAll(context.Background(), "m"); !errors.Is(err, rolltargets.ErrUnavailable) {
		t.Errorf("RemoveAll err = %v, want ErrUnavailable", err)
	}
}

// The bulk paths carry every requested id to storage and hand back exactly
// what storage reports touching, the same contract wishlist's bulk adapter
// keeps.
func TestRollTargetRepository_BulkOperationsCarryIDsAndCounts(t *testing.T) {
	t.Run("remove many", func(t *testing.T) {
		store := &fakeRollTargetStore{userID: 1, affected: 2}
		removed, err := NewRollTargetRepository(store).RemoveMany(context.Background(), "m",
			[]rolltargets.TargetID{1, 2, 3})
		if err != nil {
			t.Fatalf("RemoveMany: %v", err)
		}
		if removed != 2 {
			t.Errorf("removed = %d, want 2", removed)
		}
		if len(store.gotIDs) != 3 || store.gotIDs[0] != 1 || store.gotIDs[2] != 3 {
			t.Errorf("stored ids = %v, want [1 2 3]", store.gotIDs)
		}
	})

	t.Run("remove all", func(t *testing.T) {
		store := &fakeRollTargetStore{userID: 1, affected: 5}
		removed, err := NewRollTargetRepository(store).RemoveAll(context.Background(), "m")
		if err != nil {
			t.Fatalf("RemoveAll: %v", err)
		}
		if removed != 5 {
			t.Errorf("removed = %d, want 5", removed)
		}
		if store.gotUserID != 1 {
			t.Errorf("store saw user id %d, want 1", store.gotUserID)
		}
	})
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
