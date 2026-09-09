package adapters

import (
	"context"
	"errors"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/wishlist"

	"github.com/jackc/pgx/v5"
)

// wishlistRepository adapts the internal user-keyed wish list store to the
// membership-keyed domain repository.
//
// Two things stop here and go no further: the internal Guardian Tracker user
// id, and PostgreSQL's vocabulary for failure. A unique-constraint violation
// becomes "already saved", a missing row becomes "not found", and a missing
// database becomes "unavailable" — because the domain has to act on those
// distinctions, and no domain code should have to know a SQLSTATE to find them.
type wishlistRepository struct{ store db.WishlistRepo }

// NewWishlistRepository wraps the wish list store for wishlist.Entries.
func NewWishlistRepository(store db.WishlistRepo) wishlist.Repository {
	return &wishlistRepository{store: store}
}

// priorityRank is storage's ordering vocabulary. The column is a rank so
// `ORDER BY priority DESC` can put urgent items first; the domain's vocabulary
// is the name.
var priorityRank = map[wishlist.Priority]int16{
	wishlist.PriorityLow:    0,
	wishlist.PriorityMedium: 1,
	wishlist.PriorityHigh:   2,
	wishlist.PriorityUrgent: 3,
}

var priorityName = map[int16]wishlist.Priority{
	0: wishlist.PriorityLow,
	1: wishlist.PriorityMedium,
	2: wishlist.PriorityHigh,
	3: wishlist.PriorityUrgent,
}

func (r *wishlistRepository) List(ctx context.Context, membershipID string) ([]wishlist.StoredEntry, error) {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	rows, err := r.store.List(ctx, userID)
	if err != nil {
		return nil, wishlistError(err)
	}
	entries := make([]wishlist.StoredEntry, len(rows))
	for i, row := range rows {
		entries[i] = storedEntry(&row)
	}
	return entries, nil
}

func (r *wishlistRepository) Add(ctx context.Context, membershipID string, entry wishlist.AddCommand) (wishlist.StoredEntry, error) {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return wishlist.StoredEntry{}, err
	}
	row, err := r.store.Add(ctx, userID, entry.ItemHash, priorityRank[entry.Priority], entry.Notes)
	if err != nil {
		if db.IsDuplicate(err) {
			return wishlist.StoredEntry{}, wishlist.ErrDuplicate
		}
		return wishlist.StoredEntry{}, wishlistError(err)
	}
	return storedEntry(row), nil
}

func (r *wishlistRepository) Update(ctx context.Context, membershipID string, id wishlist.EntryID, patch wishlist.UpdateCommand) (wishlist.StoredEntry, error) {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return wishlist.StoredEntry{}, err
	}
	var rank *int16
	if patch.Priority != nil {
		stored := priorityRank[*patch.Priority]
		rank = &stored
	}
	row, err := r.store.Update(ctx, userID, int64(id), rank, patch.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return wishlist.StoredEntry{}, wishlist.ErrNotFound
		}
		return wishlist.StoredEntry{}, wishlistError(err)
	}
	return storedEntry(row), nil
}

func (r *wishlistRepository) Remove(ctx context.Context, membershipID string, id wishlist.EntryID) error {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return err
	}
	found, err := r.store.Delete(ctx, userID, int64(id))
	if err != nil {
		return wishlistError(err)
	}
	if !found {
		return wishlist.ErrNotFound
	}
	return nil
}

func (r *wishlistRepository) RemoveMany(ctx context.Context, membershipID string, ids []wishlist.EntryID) (int, error) {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return 0, err
	}
	removed, err := r.store.BulkDelete(ctx, userID, rowIDs(ids))
	if err != nil {
		return 0, wishlistError(err)
	}
	return int(removed), nil
}

func (r *wishlistRepository) SetPriorityMany(ctx context.Context, membershipID string, ids []wishlist.EntryID, priority wishlist.Priority) (int, error) {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return 0, err
	}
	updated, err := r.store.BulkSetPriority(ctx, userID, rowIDs(ids), priorityRank[priority])
	if err != nil {
		return 0, wishlistError(err)
	}
	return int(updated), nil
}

// userID resolves the internal identity every store call needs from the
// membership the domain works in.
func (r *wishlistRepository) userID(ctx context.Context, membershipID string) (int64, error) {
	id, err := r.store.GetUserID(ctx, membershipID)
	if err != nil {
		return 0, wishlistError(err)
	}
	return id, nil
}

func rowIDs(ids []wishlist.EntryID) []int64 {
	out := make([]int64, len(ids))
	for i, id := range ids {
		out[i] = int64(id)
	}
	return out
}

func storedEntry(row *db.WishlistItem) wishlist.StoredEntry {
	return wishlist.StoredEntry{
		ID:        wishlist.EntryID(row.ID),
		ItemHash:  row.ItemHash,
		Priority:  priorityName[row.Priority],
		Notes:     row.Notes,
		CreatedAt: row.CreatedAt,
	}
}

// wishlistError translates the one storage condition the domain must act on.
// Everything else passes through: reporting a real failure as "no database"
// would tell a user their saved wish list is simply unavailable when in fact
// the read broke.
func wishlistError(err error) error {
	if errors.Is(err, db.ErrUnavailable) {
		return wishlist.ErrUnavailable
	}
	return err
}
