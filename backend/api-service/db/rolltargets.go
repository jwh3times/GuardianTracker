package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RollTarget is the DB representation of one saved roll target: a perk
// combination, the weapon it applies to, and whether it is wanted or unwanted.
//
// ItemHash is nil for an any-weapon target, which names perks without naming a
// weapon. A pointer rather than a zero value because 0 is a hash the column
// would accept, and "no weapon" must not be spelled the same as a weapon.
type RollTarget struct {
	ID        int64
	UserID    int64
	ItemHash  *uint32
	Wanted    bool
	Perks     []string
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RollTargetStore handles roll-target DB operations.
type RollTargetStore struct{ pool *pgxpool.Pool }

func NewRollTargetStore(pool *pgxpool.Pool) *RollTargetStore { return &RollTargetStore{pool: pool} }

// GetUserID returns the users.id for a given membership_id.
func (s *RollTargetStore) GetUserID(ctx context.Context, membershipID string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM users WHERE membership_id = $1`, membershipID).Scan(&id)
	return id, err
}

const rollTargetCols = `id, user_id, item_hash, wanted, perks, notes, created_at, updated_at`

// List returns every roll target for the user, most recently added first.
func (s *RollTargetStore) List(ctx context.Context, userID int64) ([]RollTarget, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+rollTargetCols+`
		 FROM roll_targets WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RollTarget
	for rows.Next() {
		t, err := scanRollTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Add inserts a roll target. A nil hash stores an any-weapon target. Callers
// should check IsDuplicate on the returned error: the same roll cannot be saved
// twice, which is what stops a re-imported file from stacking duplicates.
func (s *RollTargetStore) Add(ctx context.Context, userID int64, hash *uint32, wanted bool, perks []string, notes string) (*RollTarget, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO roll_targets (user_id, item_hash, wanted, perks, notes)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+rollTargetCols,
		userID, itemHashArg(hash), wanted, perks, notes)
	t, err := scanRollTarget(row)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Update replaces perks and/or notes on a target the user owns. A nil argument
// leaves that column alone. Returns pgx.ErrNoRows when the row is missing or
// belongs to someone else.
func (s *RollTargetStore) Update(ctx context.Context, userID, id int64, perks *[]string, notes *string) (*RollTarget, error) {
	// COALESCE cannot take a nil []string through pgx as "leave it alone", so
	// the perks patch is applied with an explicit null check instead.
	row := s.pool.QueryRow(ctx,
		`UPDATE roll_targets
		 SET perks      = CASE WHEN $3::text[] IS NULL THEN perks ELSE $3::text[] END,
		     notes      = COALESCE($4, notes),
		     updated_at = now()
		 WHERE id = $1 AND user_id = $2
		 RETURNING `+rollTargetCols,
		id, userID, perks, notes)
	t, err := scanRollTarget(row)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete removes a roll target the user owns. Reports false when the row is
// missing or belongs to someone else.
func (s *RollTargetStore) Delete(ctx context.Context, userID, id int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM roll_targets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows, so one scan helper
// serves the single-row writes and the list read.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRollTarget(r rowScanner) (RollTarget, error) {
	var t RollTarget
	var hashInt *int64
	if err := r.Scan(&t.ID, &t.UserID, &hashInt, &t.Wanted, &t.Perks, &t.Notes, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return RollTarget{}, err
	}
	if hashInt != nil {
		h := uint32(*hashInt)
		t.ItemHash = &h
	}
	return t, nil
}

// itemHashArg widens a hash for the BIGINT column while keeping "no weapon" as
// a real NULL rather than a zero.
func itemHashArg(hash *uint32) *int64 {
	if hash == nil {
		return nil
	}
	v := int64(*hash)
	return &v
}
