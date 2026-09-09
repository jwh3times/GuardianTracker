package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/wishlist"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeWishlistStore records what reached storage and returns what the test
// tells it to. It embeds db.WishlistRepo so any method a test does not stub
// panics rather than quietly succeeding.
type fakeWishlistStore struct {
	db.WishlistRepo

	userID    int64
	userErr   error
	rows      []db.WishlistItem
	row       *db.WishlistItem
	err       error
	affected  int64
	found     bool
	gotUserID int64
	gotHash   uint32
	gotRank   int16
	gotNotes  string
	gotIDs    []int64
	gotRankPt *int16
}

func (f *fakeWishlistStore) GetUserID(context.Context, string) (int64, error) {
	return f.userID, f.userErr
}

func (f *fakeWishlistStore) List(_ context.Context, userID int64) ([]db.WishlistItem, error) {
	f.gotUserID = userID
	return f.rows, f.err
}

func (f *fakeWishlistStore) Add(_ context.Context, userID int64, hash uint32, rank int16, notes string) (*db.WishlistItem, error) {
	f.gotUserID, f.gotHash, f.gotRank, f.gotNotes = userID, hash, rank, notes
	return f.row, f.err
}

func (f *fakeWishlistStore) Update(_ context.Context, userID, id int64, rank *int16, notes *string) (*db.WishlistItem, error) {
	f.gotUserID, f.gotIDs, f.gotRankPt = userID, []int64{id}, rank
	if notes != nil {
		f.gotNotes = *notes
	}
	return f.row, f.err
}

func (f *fakeWishlistStore) Delete(_ context.Context, userID, id int64) (bool, error) {
	f.gotUserID, f.gotIDs = userID, []int64{id}
	return f.found, f.err
}

func (f *fakeWishlistStore) BulkDelete(_ context.Context, userID int64, ids []int64) (int64, error) {
	f.gotUserID, f.gotIDs = userID, ids
	return f.affected, f.err
}

func (f *fakeWishlistStore) BulkSetPriority(_ context.Context, userID int64, ids []int64, rank int16) (int64, error) {
	f.gotUserID, f.gotIDs, f.gotRank = userID, ids, rank
	return f.affected, f.err
}

func duplicateKeyError() error {
	return &pgconn.PgError{Code: "23505"}
}

func TestWishlistRepository_ListProjectsRowsInRepositoryOrder(t *testing.T) {
	created := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeWishlistStore{
		userID: 42,
		rows: []db.WishlistItem{
			{ID: 1, ItemHash: 100, Priority: 3, Notes: "Gjallarhorn", CreatedAt: created},
			{ID: 2, ItemHash: 200, Priority: 0, Notes: "", CreatedAt: created},
		},
	}

	entries, err := NewWishlistRepository(store).List(context.Background(), "member-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if store.gotUserID != 42 {
		t.Errorf("store queried user %d, want the membership's resolved id 42", store.gotUserID)
	}
	want := []wishlist.StoredEntry{
		{ID: 1, ItemHash: 100, Priority: wishlist.PriorityUrgent, Notes: "Gjallarhorn", CreatedAt: created},
		{ID: 2, ItemHash: 200, Priority: wishlist.PriorityLow, CreatedAt: created},
	}
	if len(entries) != len(want) {
		t.Fatalf("entries = %+v, want %+v", entries, want)
	}
	for i := range want {
		if entries[i] != want[i] {
			t.Errorf("entries[%d] = %+v, want %+v", i, entries[i], want[i])
		}
	}
}

// The domain names a priority; only storage ranks it. A wrong mapping here
// silently reshuffles everyone's wish list, so both directions are pinned.
func TestWishlistRepository_TranslatesPriorityBothWays(t *testing.T) {
	ranks := map[wishlist.Priority]int16{
		wishlist.PriorityLow:    0,
		wishlist.PriorityMedium: 1,
		wishlist.PriorityHigh:   2,
		wishlist.PriorityUrgent: 3,
	}
	for priority, rank := range ranks {
		t.Run(string(priority), func(t *testing.T) {
			store := &fakeWishlistStore{row: &db.WishlistItem{ID: 7, ItemHash: 1, Priority: rank}}
			entry, err := NewWishlistRepository(store).Add(context.Background(), "member-1",
				wishlist.AddCommand{ItemHash: 1, Priority: priority})
			if err != nil {
				t.Fatalf("Add: %v", err)
			}
			if store.gotRank != rank {
				t.Errorf("stored rank = %d, want %d", store.gotRank, rank)
			}
			if entry.Priority != priority {
				t.Errorf("returned priority = %q, want %q", entry.Priority, priority)
			}
		})
	}
}

// A unique-constraint violation is the storage spelling of "already saved".
// Left untranslated it would surface as a 500 instead of the conflict it is.
func TestWishlistRepository_AddTranslatesDuplicate(t *testing.T) {
	store := &fakeWishlistStore{err: duplicateKeyError()}

	_, err := NewWishlistRepository(store).Add(context.Background(), "member-1", wishlist.AddCommand{ItemHash: 1})

	if !errors.Is(err, wishlist.ErrDuplicate) {
		t.Fatalf("Add error = %v, want wishlist.ErrDuplicate", err)
	}
}

func TestWishlistRepository_UpdateTranslatesMissingRow(t *testing.T) {
	store := &fakeWishlistStore{err: pgx.ErrNoRows}

	_, err := NewWishlistRepository(store).Update(context.Background(), "member-1", 5, wishlist.UpdateCommand{})

	if !errors.Is(err, wishlist.ErrNotFound) {
		t.Fatalf("Update error = %v, want wishlist.ErrNotFound", err)
	}
}

// Update leaves an unset field alone. Sending a rank of zero for "no change"
// would silently demote every patched entry to Low.
func TestWishlistRepository_UpdateSendsNoRankWhenPriorityIsAbsent(t *testing.T) {
	store := &fakeWishlistStore{row: &db.WishlistItem{ID: 5, Priority: 2}}

	if _, err := NewWishlistRepository(store).Update(context.Background(), "member-1", 5,
		wishlist.UpdateCommand{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if store.gotRankPt != nil {
		t.Errorf("stored rank = %d, want no rank at all for an absent priority", *store.gotRankPt)
	}
}

// A delete that affected nothing is the entry being missing or foreign — the
// storage layer reports it as a count, and the domain needs it as an error.
func TestWishlistRepository_RemoveTranslatesAMissedDelete(t *testing.T) {
	store := &fakeWishlistStore{found: false}

	err := NewWishlistRepository(store).Remove(context.Background(), "member-1", 9)

	if !errors.Is(err, wishlist.ErrNotFound) {
		t.Fatalf("Remove error = %v, want wishlist.ErrNotFound", err)
	}
}

func TestWishlistRepository_BulkOperationsCarryIDsAndCounts(t *testing.T) {
	t.Run("remove", func(t *testing.T) {
		store := &fakeWishlistStore{affected: 2}
		removed, err := NewWishlistRepository(store).RemoveMany(context.Background(), "member-1",
			[]wishlist.EntryID{1, 2, 3})
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

	t.Run("set priority", func(t *testing.T) {
		store := &fakeWishlistStore{affected: 1}
		updated, err := NewWishlistRepository(store).SetPriorityMany(context.Background(), "member-1",
			[]wishlist.EntryID{4}, wishlist.PriorityHigh)
		if err != nil {
			t.Fatalf("SetPriorityMany: %v", err)
		}
		if updated != 1 {
			t.Errorf("updated = %d, want 1", updated)
		}
		if store.gotRank != 2 {
			t.Errorf("stored rank = %d, want 2 for HIGH", store.gotRank)
		}
	})
}

// A degraded store reports ErrUnavailable on every method. The domain must see
// its own sentinel — an empty wish list would read as "wants nothing".
func TestWishlistRepository_TranslatesUnavailableOnEveryMethod(t *testing.T) {
	repo := NewWishlistRepository(&fakeWishlistStore{userErr: db.ErrUnavailable})
	ctx := context.Background()

	calls := map[string]func() error{
		"List": func() error { _, err := repo.List(ctx, "member-1"); return err },
		"Add": func() error {
			_, err := repo.Add(ctx, "member-1", wishlist.AddCommand{ItemHash: 1, Priority: wishlist.PriorityLow})
			return err
		},
		"Update": func() error {
			_, err := repo.Update(ctx, "member-1", 1, wishlist.UpdateCommand{})
			return err
		},
		"Remove":     func() error { return repo.Remove(ctx, "member-1", 1) },
		"RemoveMany": func() error { _, err := repo.RemoveMany(ctx, "member-1", []wishlist.EntryID{1}); return err },
		"SetPriorityMany": func() error {
			_, err := repo.SetPriorityMany(ctx, "member-1", []wishlist.EntryID{1}, wishlist.PriorityLow)
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, wishlist.ErrUnavailable) {
				t.Errorf("%s error = %v, want wishlist.ErrUnavailable", name, err)
			}
		})
	}
}

// Everything else passes through untranslated: a broken read reported as "no
// database" would tell the user their wish list is unconfigured when it exists.
func TestWishlistRepository_PassesRealFailuresThrough(t *testing.T) {
	boom := errors.New("connection reset")
	store := &fakeWishlistStore{userID: 1, err: boom}

	_, err := NewWishlistRepository(store).List(context.Background(), "member-1")

	if !errors.Is(err, boom) {
		t.Fatalf("List error = %v, want the original failure", err)
	}
	if errors.Is(err, wishlist.ErrUnavailable) {
		t.Fatal("a real failure was reported as an absent database")
	}
}
