// Package wishlist owns a Guardian Tracker user's saved wish list: what a
// stored entry means, which mutations are legal, and how persistence failures
// are named.
//
// The package deliberately knows nothing about PostgreSQL, internal user IDs,
// HTTP, or the Destiny manifest's raw definitions. Storage identity and driver
// errors are translated by an adapter (see db/adapters); item meaning arrives
// through the canonical Items seam (ADR 0015). See
// [ADR 0019](../../../../docs/adr/0019-own-wish-list-and-preferences.md).
package wishlist

import (
	"context"
	"errors"
	"time"

	"guardian-tracker/api-service/services/items"
)

// EntryID is the saved wish list entry's identity. It is deliberately its own
// type: the wire carries it as a string, storage as a row id, and neither
// meaning should be assignable to an item hash or a user id by accident.
type EntryID int64

// Priority is how badly the user wants the item. The domain value is the
// uppercase wire spelling, so nothing has to map between two vocabularies —
// only storage, which orders by rank, translates.
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
	PriorityUrgent Priority = "URGENT"
)

// Valid reports whether p is one of the four priorities.
func (p Priority) Valid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	}
	return false
}

// MaxNoteRunes is the note limit, counted in Unicode code points.
//
// Counting bytes — which the previous handler check did — rejects valid notes
// that PostgreSQL's `char_length` constraint would have accepted, because a
// single emoji or accented character costs up to four bytes. The product limit
// has always been 500 characters; this is where that is now decided.
const MaxNoteRunes = 500

// MaxBulkEntries bounds one bulk command, counted after duplicate ids are
// removed.
const MaxBulkEntries = 100

// StoredEntry is one persisted, user-authored wish list entry. It carries no
// item facts and no availability: those belong to other owners and complete an
// entry later, and mixing them in here is what previously let the same item
// read differently in different places.
type StoredEntry struct {
	ID        EntryID
	ItemHash  uint32
	Priority  Priority
	Notes     string
	CreatedAt time.Time
}

// AddCommand is a request to save one item. An empty Priority means the caller
// did not choose one and takes the default.
type AddCommand struct {
	ItemHash uint32
	Priority Priority
	Notes    string
}

// UpdateCommand is a partial patch: a nil field is unchanged, which is what
// distinguishes "leave the notes alone" from "clear the notes".
type UpdateCommand struct {
	Priority *Priority
	Notes    *string
}

// BulkResult reports how a bulk command landed. Skipped counts the requested
// entries that were missing or belong to someone else — their absence is an
// answer about those entries, not a failure of the command.
type BulkResult struct {
	Updated int
	Skipped int
}

var (
	// ErrUnavailable means the wish list has no persistence behind it. It is
	// distinct from an empty wish list on purpose: "you have saved nothing"
	// and "we cannot tell you what you saved" must never render the same.
	ErrUnavailable = errors.New("wishlist: persistence unavailable")

	// ErrNotFound means the entry does not exist, or exists under another
	// membership. The two are deliberately indistinguishable to the caller:
	// telling them apart would confirm another user's entry ids.
	ErrNotFound = errors.New("wishlist: entry not found")

	// ErrDuplicate means the item is already on this membership's wish list.
	ErrDuplicate = errors.New("wishlist: item already saved")

	// ErrInvalidPriority reports a priority outside the four legal values.
	ErrInvalidPriority = errors.New("wishlist: priority must be LOW, MEDIUM, HIGH, or URGENT")

	// ErrNotesTooLong reports a note longer than MaxNoteRunes code points.
	ErrNotesTooLong = errors.New("wishlist: notes must be 500 characters or fewer")

	// ErrNoEntries reports a bulk command with nothing left to act on.
	ErrNoEntries = errors.New("wishlist: at least one entry id is required")

	// ErrTooManyEntries reports a bulk command over MaxBulkEntries.
	ErrTooManyEntries = errors.New("wishlist: too many entry ids in one request")

	// ErrUnknownItem means the manifest, read successfully, does not contain
	// the requested item. Saving it would persist metadata about nothing.
	ErrUnknownItem = errors.New("wishlist: unknown item hash")

	// ErrItemsUnavailable means item facts could not be read at all. It is
	// never conflated with ErrUnknownItem: an unreadable manifest must not be
	// allowed to look like a verdict about the item, in either direction.
	ErrItemsUnavailable = errors.New("wishlist: item facts unavailable")
)

// Repository is the membership-keyed persistence port. Implementations resolve
// the internal user identity themselves, translate storage failures into this
// package's error vocabulary, and return entries in repository order.
//
// Repository order today is priority first, then most recently added — the
// order the wish list has always been served in. It is the repository's to
// define and callers must not re-sort it.
//
// A repository with no database behind it returns ErrUnavailable, never an
// empty list.
type Repository interface {
	List(ctx context.Context, membershipID string) ([]StoredEntry, error)

	// Add persists a validated entry. A membership that already saved the item
	// gets ErrDuplicate.
	Add(ctx context.Context, membershipID string, entry AddCommand) (StoredEntry, error)

	// Update applies a validated patch to one entry the membership owns, and
	// returns the stored result. An entry that is missing or foreign gets
	// ErrNotFound.
	Update(ctx context.Context, membershipID string, id EntryID, patch UpdateCommand) (StoredEntry, error)

	// Remove deletes one entry the membership owns; missing or foreign gets
	// ErrNotFound.
	Remove(ctx context.Context, membershipID string, id EntryID) error

	// RemoveMany deletes every listed entry the membership owns and reports how
	// many were deleted. Missing and foreign ids are skipped, not refused.
	RemoveMany(ctx context.Context, membershipID string, ids []EntryID) (int, error)

	// SetPriorityMany sets one priority on every listed entry the membership
	// owns and reports how many were changed, with the same skip semantics.
	SetPriorityMany(ctx context.Context, membershipID string, ids []EntryID, priority Priority) (int, error)
}

// ItemLookup is the Items surface the wish list consumes: does this item exist,
// and what is it? Satisfied by *items.Service.
//
// The wish list asks only so it can refuse to save an item the manifest has
// never heard of. What an entry's item means to a reader is completed by the
// outer service.
type ItemLookup interface {
	Lookup(ctx context.Context, hashes []uint32) (map[uint32]items.AcquisitionFacts, error)
}
