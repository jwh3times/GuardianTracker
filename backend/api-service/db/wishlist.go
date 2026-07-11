package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WishlistItem is the DB representation of a user's wishlist entry.
type WishlistItem struct {
	ID        int64
	UserID    int64
	ItemHash  uint32
	Priority  int16
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WishlistStore handles wishlist DB operations.
type WishlistStore struct{ pool *pgxpool.Pool }

func NewWishlistStore(pool *pgxpool.Pool) *WishlistStore { return &WishlistStore{pool: pool} }

// GetUserID returns the users.id for a given membership_id.
func (s *WishlistStore) GetUserID(ctx context.Context, membershipID string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE membership_id = $1`, membershipID).Scan(&id)
	return id, err
}

// List returns all wishlist items for the user, ordered by priority DESC, created_at DESC.
func (s *WishlistStore) List(ctx context.Context, userID int64) ([]WishlistItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, item_hash, priority, notes, created_at, updated_at
		 FROM wishlist_items WHERE user_id = $1
		 ORDER BY priority DESC, created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []WishlistItem
	for rows.Next() {
		var it WishlistItem
		var hashInt int64
		if err := rows.Scan(&it.ID, &it.UserID, &hashInt, &it.Priority, &it.Notes, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		it.ItemHash = uint32(hashInt)
		items = append(items, it)
	}
	return items, rows.Err()
}

// Add inserts a new wishlist item. Callers should check IsDuplicate on the returned error.
func (s *WishlistStore) Add(ctx context.Context, userID int64, hash uint32, prio int16, notes string) (*WishlistItem, error) {
	var it WishlistItem
	var hashInt int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO wishlist_items (user_id, item_hash, priority, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, item_hash, priority, notes, created_at, updated_at`,
		userID, int64(hash), prio, notes,
	).Scan(&it.ID, &it.UserID, &hashInt, &it.Priority, &it.Notes, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		return nil, err
	}
	it.ItemHash = uint32(hashInt)
	return &it, nil
}

// Update modifies priority and/or notes for an item the user owns.
// Returns pgx.ErrNoRows if not found or not owned by this user.
func (s *WishlistStore) Update(ctx context.Context, userID, id int64, prio *int16, notes *string) (*WishlistItem, error) {
	var it WishlistItem
	var hashInt int64
	err := s.pool.QueryRow(ctx,
		`UPDATE wishlist_items
		 SET priority   = COALESCE($3, priority),
		     notes      = COALESCE($4, notes),
		     updated_at = now()
		 WHERE id = $1 AND user_id = $2
		 RETURNING id, user_id, item_hash, priority, notes, created_at, updated_at`,
		id, userID, prio, notes,
	).Scan(&it.ID, &it.UserID, &hashInt, &it.Priority, &it.Notes, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		return nil, err
	}
	it.ItemHash = uint32(hashInt)
	return &it, nil
}

// Delete removes a wishlist item the user owns. Returns false if not found or not owned.
func (s *WishlistStore) Delete(ctx context.Context, userID, id int64) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM wishlist_items WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// BulkDelete removes every wishlist item in ids that the user owns. Foreign or
// missing ids are silently skipped. Returns the number of rows deleted.
func (s *WishlistStore) BulkDelete(ctx context.Context, userID int64, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM wishlist_items WHERE id = ANY($1) AND user_id = $2`, ids, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// BulkSetPriority sets priority on every wishlist item in ids that the user owns.
// Foreign or missing ids are silently skipped. Returns the number of rows updated.
func (s *WishlistStore) BulkSetPriority(ctx context.Context, userID int64, ids []int64, prio int16) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE wishlist_items SET priority = $3, updated_at = now()
		 WHERE id = ANY($1) AND user_id = $2`, ids, userID, prio)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// IsDuplicate returns true when the error is a unique-constraint violation on (user_id, item_hash).
func IsDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
