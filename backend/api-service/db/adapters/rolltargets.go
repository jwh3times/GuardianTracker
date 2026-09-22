package adapters

import (
	"context"
	"errors"

	"guardian-tracker/api-service/db"
	"guardian-tracker/api-service/services/rolltargets"

	"github.com/jackc/pgx/v5"
)

// rollTargetRepository adapts the internal user-keyed roll-target store to the
// membership-keyed domain repository.
//
// Two things stop here and go no further: the internal Guardian Tracker user
// id, and PostgreSQL's vocabulary for failure. A unique-constraint violation
// becomes "this roll is already saved", a missing row becomes "not
// found", and a missing database becomes "unavailable" — because the domain
// acts on those distinctions and no domain code should need a SQLSTATE to find
// them.
type rollTargetRepository struct{ store db.RollTargetRepo }

// NewRollTargetRepository wraps the roll-target store for rolltargets.Service.
func NewRollTargetRepository(store db.RollTargetRepo) rolltargets.Repository {
	return &rollTargetRepository{store: store}
}

func (r *rollTargetRepository) List(ctx context.Context, membershipID string) ([]rolltargets.StoredTarget, error) {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	rows, err := r.store.List(ctx, userID)
	if err != nil {
		return nil, rollTargetError(err)
	}
	out := make([]rolltargets.StoredTarget, len(rows))
	for i, row := range rows {
		out[i] = storedTarget(&row)
	}
	return out, nil
}

func (r *rollTargetRepository) Add(ctx context.Context, membershipID string, target rolltargets.AddCommand) (rolltargets.StoredTarget, error) {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return rolltargets.StoredTarget{}, err
	}
	row, err := r.store.Add(ctx, userID, target.ItemHash, target.Wanted, target.Perks, target.Notes)
	if err != nil {
		if db.IsDuplicate(err) {
			return rolltargets.StoredTarget{}, rolltargets.ErrDuplicate
		}
		return rolltargets.StoredTarget{}, rollTargetError(err)
	}
	return storedTarget(row), nil
}

func (r *rollTargetRepository) Update(ctx context.Context, membershipID string, id rolltargets.TargetID, patch rolltargets.UpdateCommand) (rolltargets.StoredTarget, error) {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return rolltargets.StoredTarget{}, err
	}
	row, err := r.store.Update(ctx, userID, int64(id), patch.Perks, patch.Notes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return rolltargets.StoredTarget{}, rolltargets.ErrNotFound
		}
		return rolltargets.StoredTarget{}, rollTargetError(err)
	}
	return storedTarget(row), nil
}

func (r *rollTargetRepository) Remove(ctx context.Context, membershipID string, id rolltargets.TargetID) error {
	userID, err := r.userID(ctx, membershipID)
	if err != nil {
		return err
	}
	found, err := r.store.Delete(ctx, userID, int64(id))
	if err != nil {
		return rollTargetError(err)
	}
	if !found {
		return rolltargets.ErrNotFound
	}
	return nil
}

// userID resolves the internal identity every store call needs from the
// membership the domain works in.
func (r *rollTargetRepository) userID(ctx context.Context, membershipID string) (int64, error) {
	id, err := r.store.GetUserID(ctx, membershipID)
	if err != nil {
		return 0, rollTargetError(err)
	}
	return id, nil
}

func storedTarget(row *db.RollTarget) rolltargets.StoredTarget {
	return rolltargets.StoredTarget{
		ID:        rolltargets.TargetID(row.ID),
		ItemHash:  row.ItemHash,
		Wanted:    row.Wanted,
		Perks:     row.Perks,
		Notes:     row.Notes,
		CreatedAt: row.CreatedAt,
	}
}

// rollTargetError translates the one storage condition the domain must act on.
// Everything else passes through: reporting a real failure as "no database"
// would tell a user their saved targets are simply unavailable when in fact the
// read broke.
func rollTargetError(err error) error {
	if errors.Is(err, db.ErrUnavailable) {
		return rolltargets.ErrUnavailable
	}
	return err
}
