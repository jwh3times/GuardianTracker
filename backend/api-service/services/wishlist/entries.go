package wishlist

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"guardian-tracker/api-service/services/items"
)

// Entries is the reusable wish list core: persistence, mutation validation,
// item-existence rules, and the membership-scoped item-hash view.
//
// It is the inner of the two construction stages in ADR 0019. Weekly reads
// saved item hashes through it, which is what lets the complete wish list
// service depend on Weekly's live availability without a construction cycle —
// and it is why nothing else may reach the repository directly. Two consumers
// of the same store would each get to decide what an unavailable wish list
// means, and they would not agree.
type Entries struct {
	repository Repository
	items      ItemLookup
}

// NewEntries constructs the core around its two required dependencies.
func NewEntries(repository Repository, items ItemLookup) *Entries {
	return &Entries{repository: repository, items: items}
}

// List returns the membership's saved entries in repository order.
func (e *Entries) List(ctx context.Context, membershipID string) ([]StoredEntry, error) {
	return e.repository.List(ctx, membershipID)
}

// ListItemHashes is the narrow view Weekly consumes: which items this
// membership has saved, in repository order.
//
// Weekly treats a failure as "no personalization" rather than an error, so this
// deliberately reports one instead of hiding it — the decision to degrade
// belongs to the consumer that knows the signal is optional.
func (e *Entries) ListItemHashes(ctx context.Context, membershipID string) ([]uint32, error) {
	stored, err := e.repository.List(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	hashes := make([]uint32, len(stored))
	for i, entry := range stored {
		hashes[i] = entry.ItemHash
	}
	return hashes, nil
}

// Add validates and saves one item. An omitted priority defaults to Medium.
//
// The item must exist in a successful lookup. A lookup that fails outright
// writes nothing and says so: allowing the save would record a wish for an
// item nobody has confirmed exists, and refusing it as "unknown" would blame
// the user's request for the manifest being unreadable.
// The facts it validated against are returned alongside the stored row. The
// caller needs them to describe what it just saved, and handing back the answer
// already in hand is not a convenience: looking the same item up again after
// the write opens a window where the row is committed but the response cannot
// be built, which would report a failure for an operation that succeeded.
func (e *Entries) Add(ctx context.Context, membershipID string, cmd AddCommand) (StoredEntry, items.AcquisitionFacts, error) {
	if cmd.Priority == "" {
		cmd.Priority = PriorityMedium
	}
	if !cmd.Priority.Valid() {
		return StoredEntry{}, items.AcquisitionFacts{}, ErrInvalidPriority
	}
	if err := validateNotes(cmd.Notes); err != nil {
		return StoredEntry{}, items.AcquisitionFacts{}, err
	}
	facts, err := e.requireKnownItem(ctx, cmd.ItemHash)
	if err != nil {
		return StoredEntry{}, items.AcquisitionFacts{}, err
	}
	stored, err := e.repository.Add(ctx, membershipID, cmd)
	if err != nil {
		return StoredEntry{}, items.AcquisitionFacts{}, err
	}
	return stored, facts, nil
}

// Update applies a partial patch to one entry the membership owns.
//
// It deliberately does not re-check the item: an entry already saved stays
// editable even after its item leaves the manifest, so a user can still fix or
// remove their own note about it.
func (e *Entries) Update(ctx context.Context, membershipID string, id EntryID, patch UpdateCommand) (StoredEntry, error) {
	if patch.Priority != nil && !patch.Priority.Valid() {
		return StoredEntry{}, ErrInvalidPriority
	}
	if patch.Notes != nil {
		if err := validateNotes(*patch.Notes); err != nil {
			return StoredEntry{}, err
		}
	}
	return e.repository.Update(ctx, membershipID, id, patch)
}

// Remove deletes one entry the membership owns.
func (e *Entries) Remove(ctx context.Context, membershipID string, id EntryID) error {
	return e.repository.Remove(ctx, membershipID, id)
}

// RemoveMany deletes the listed entries the membership owns.
func (e *Entries) RemoveMany(ctx context.Context, membershipID string, ids []EntryID) (BulkResult, error) {
	unique, err := bulkIDs(ids)
	if err != nil {
		return BulkResult{}, err
	}
	removed, err := e.repository.RemoveMany(ctx, membershipID, unique)
	if err != nil {
		return BulkResult{}, err
	}
	return bulkResult(unique, removed), nil
}

// SetPriorityMany sets one priority across the listed entries the membership
// owns.
func (e *Entries) SetPriorityMany(ctx context.Context, membershipID string, ids []EntryID, priority Priority) (BulkResult, error) {
	if !priority.Valid() {
		return BulkResult{}, ErrInvalidPriority
	}
	unique, err := bulkIDs(ids)
	if err != nil {
		return BulkResult{}, err
	}
	updated, err := e.repository.SetPriorityMany(ctx, membershipID, unique, priority)
	if err != nil {
		return BulkResult{}, err
	}
	return bulkResult(unique, updated), nil
}

// requireKnownItem separates the three item outcomes that must never collapse
// into each other: a confirmed item, a confirmed absence, and no answer at all.
// A confirmed item's facts come back with it, so nobody has to ask twice.
func (e *Entries) requireKnownItem(ctx context.Context, itemHash uint32) (items.AcquisitionFacts, error) {
	facts, err := e.items.Lookup(ctx, []uint32{itemHash})
	if err != nil {
		return items.AcquisitionFacts{}, fmt.Errorf("%w: %w", ErrItemsUnavailable, err)
	}
	known, ok := facts[itemHash]
	if !ok {
		return items.AcquisitionFacts{}, ErrUnknownItem
	}
	return known, nil
}

// validateNotes counts code points, matching the PostgreSQL `char_length`
// constraint the value has to survive.
func validateNotes(notes string) error {
	if utf8.RuneCountInString(notes) > MaxNoteRunes {
		return ErrNotesTooLong
	}
	return nil
}

// bulkIDs removes duplicate ids before the size rules apply, so a client that
// sends the same entry twice is asking about one entry — not two, and not a
// larger command than it thinks.
func bulkIDs(ids []EntryID) ([]EntryID, error) {
	seen := make(map[EntryID]struct{}, len(ids))
	unique := make([]EntryID, 0, len(ids))
	for _, id := range ids {
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil, ErrNoEntries
	}
	if len(unique) > MaxBulkEntries {
		return nil, fmt.Errorf("%w: at most %d", ErrTooManyEntries, MaxBulkEntries)
	}
	return unique, nil
}

// bulkResult reports what the storage layer actually touched. Entries that were
// missing or belong to another membership are skipped rather than failing the
// command, so one stale id in a selection does not discard the rest.
func bulkResult(requested []EntryID, affected int) BulkResult {
	if affected < 0 {
		affected = 0
	}
	skipped := len(requested) - affected
	if skipped < 0 {
		skipped = 0
	}
	return BulkResult{Updated: affected, Skipped: skipped}
}

// IsValidationError reports whether err is one of the request-shaped refusals —
// the ones a caller fixes by sending something different, rather than by
// retrying later.
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidPriority) ||
		errors.Is(err, ErrNotesTooLong) ||
		errors.Is(err, ErrNoEntries) ||
		errors.Is(err, ErrTooManyEntries) ||
		errors.Is(err, ErrUnknownItem)
}
