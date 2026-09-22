// Package rolltargets owns a Guardian Tracker user's saved roll targets: what a
// stored target means, which mutations are legal, and how persistence failures
// are named.
//
// A roll target is the player's own saved perk combination wanted on a specific
// weapon (CONTEXT.md). It is deliberately not a wish list entry: a wish list
// entry saves a wanted Item hash and has no perk dimension, and collapsing the
// two would make every row of one half-empty.
//
// The package knows nothing about PostgreSQL, internal user IDs, or HTTP —
// storage identity and driver errors are translated by an adapter (see
// db/adapters). It does depend on the manifest's PerkColumn, because a target
// is only meaningful against the pool the weapon can actually roll, and that
// pool is the manifest's to define.
//
// Follows the ownership pattern of
// [ADR 0019](../../../../docs/adr/0019-own-wish-list-and-preferences.md).
package rolltargets

import (
	"context"
	"errors"
	"time"

	"guardian-tracker/api-service/services/manifest"
)

// TargetID is the saved roll target's identity. Its own type on purpose: the
// wire carries it as a string and storage as a row id, and neither should be
// assignable to an item hash or a user id by accident.
type TargetID int64

// MaxPerks bounds one target's wanted perks. Weapons top out at six perk
// columns; ten leaves room without letting a target become a list.
const MaxPerks = 10

// MaxNoteRunes is the note limit, counted in Unicode code points — matching the
// wish list, and counted the same way, because a byte count rejects valid notes
// that the storage constraint would accept.
const MaxNoteRunes = 500

// StoredTarget is one persisted, user-authored roll target.
//
// Perks holds perk display names, not plug hashes, and they are AND-ed: a
// weapon matches only when every name is present. Names rather than hashes
// because a perk's base and enhanced variants are two different hashes sharing
// one name, linked by no manifest field (see manifest.PerkPlug) — a stored hash
// would match only the variant it was captured from, so an enhanced drop would
// silently stop matching a target written against the base.
//
// Perks are held sorted. Order carries no meaning because they are AND-ed, and
// normalising it is what lets one roll be recognised as already saved.
type StoredTarget struct {
	ID TargetID

	// ItemHash is nil for an any-weapon target, which names perks without
	// naming a weapon. A pointer rather than a zero value, because 0 must not
	// be readable as "no weapon".
	ItemHash *uint32

	// Wanted distinguishes a roll the player is looking for from one they want
	// to be told about so they can dismantle it.
	Wanted bool

	Perks     []string
	Notes     string
	CreatedAt time.Time
}

// AnyWeapon reports whether the target names perks without naming a weapon.
func (t StoredTarget) AnyWeapon() bool { return t.ItemHash == nil }

// AddCommand is a request to save one roll. A nil ItemHash saves an any-weapon
// target.
type AddCommand struct {
	ItemHash *uint32
	Wanted   bool
	Perks    []string
	Notes    string
}

// UpdateCommand is a partial patch: a nil field is unchanged, which is what
// distinguishes "leave the notes alone" from "clear the notes". Perks are
// replaced wholesale rather than merged — a wanted roll is one combination, and
// merging two combinations produces a third nobody asked for.
type UpdateCommand struct {
	Perks *[]string
	Notes *string
}

var (
	// ErrUnavailable means roll targets have no persistence behind them. It is
	// distinct from having none saved on purpose: "you have saved nothing" and
	// "we cannot tell you what you saved" must never render the same.
	ErrUnavailable = errors.New("rolltargets: persistence unavailable")

	// ErrNotFound means the target does not exist, or exists under another
	// membership. The two are deliberately indistinguishable to the caller:
	// telling them apart would confirm another user's target ids.
	ErrNotFound = errors.New("rolltargets: target not found")

	// ErrDuplicate means this membership has already saved this exact roll —
	// the same weapon, stance and perks. Several different rolls on one weapon
	// are ordinary; the same one twice is not.
	ErrDuplicate = errors.New("rolltargets: this roll is already saved")

	// ErrNoPerks reports a target naming no perks. It would match every copy of
	// the weapon, which is what a wish list entry already does.
	ErrNoPerks = errors.New("rolltargets: at least one perk is required")

	// ErrTooManyPerks reports a target over MaxPerks.
	ErrTooManyPerks = errors.New("rolltargets: too many perks in one target")

	// ErrDuplicatePerk reports the same perk named twice in one target. AND-ing
	// a perk with itself is always a typo.
	ErrDuplicatePerk = errors.New("rolltargets: the same perk is named twice")

	// ErrNotesTooLong reports a note longer than MaxNoteRunes code points.
	ErrNotesTooLong = errors.New("rolltargets: notes must be 500 characters or fewer")

	// ErrNotAWeapon means the hash resolves to something with no perk columns —
	// armor, a consumable, or nothing at all. A roll target on it could never
	// match.
	ErrNotAWeapon = errors.New("rolltargets: item is not a weapon with perk columns")

	// ErrUnknownPerkName means a perk named on an any-weapon target matches no
	// plug in the manifest. Such a target has no weapon pool to check against,
	// so its names are checked against the manifest's plugs instead.
	ErrUnknownPerkName = errors.New("rolltargets: no perk has that name")

	// ErrUnknownPerk means the weapon's pool, read successfully, does not
	// contain a named perk. Saving it would persist a target that can never
	// match, and silently never matching is indistinguishable from a bug.
	ErrUnknownPerk = errors.New("rolltargets: weapon cannot roll a named perk")

	// ErrOwnedRollsUnavailable means this deployment has no way to read what
	// the player owns, so matching cannot be attempted. Distinct from a read
	// that found nothing: one is "we cannot look", the other is "nothing you
	// own matches".
	ErrOwnedRollsUnavailable = errors.New("rolltargets: owned rolls cannot be read")

	// ErrPerksUnavailable means the weapon's perk pool could not be read at
	// all. It is never conflated with ErrNotAWeapon or ErrUnknownPerk: an
	// unreadable manifest must not look like a verdict about the weapon, in
	// either direction.
	ErrPerksUnavailable = errors.New("rolltargets: weapon perk pool unavailable")
)

// Repository is the membership-keyed persistence port. Implementations resolve
// the internal user identity themselves, translate storage failures into this
// package's error vocabulary, and return targets in repository order.
//
// Repository order is most recently added first. It is the repository's to
// define and callers must not re-sort it.
//
// A repository with no database behind it returns ErrUnavailable, never an
// empty list.
type Repository interface {
	List(ctx context.Context, membershipID string) ([]StoredTarget, error)

	// Add persists a validated target. A membership that already has one for
	// the weapon gets ErrDuplicate.
	Add(ctx context.Context, membershipID string, target AddCommand) (StoredTarget, error)

	// Update applies a validated patch to one target the membership owns. A
	// missing or foreign target gets ErrNotFound.
	Update(ctx context.Context, membershipID string, id TargetID, patch UpdateCommand) (StoredTarget, error)

	// Remove deletes one target the membership owns; missing or foreign gets
	// ErrNotFound.
	Remove(ctx context.Context, membershipID string, id TargetID) error
}

// PerkPool is the manifest surface roll targets consume: which perks can this
// weapon roll? Satisfied by *items.Service.
//
// Roll targets ask only so they can refuse to save a target the weapon could
// never satisfy. Columns arrive already deduped across a perk's base and
// enhanced variants, which is exactly the identity a target is written in.
type PerkPool interface {
	GetWeaponPerks(itemHash uint32) ([]manifest.PerkColumn, error)

	// PlugNames resolves plug item hashes to their display names. An any-weapon
	// target has no pool to resolve against, and a DIM wildcard line still
	// carries plug hashes, so they are resolved directly. Safe where a name
	// lookup would not be: many plugs share one name, but each hash has exactly
	// one, and base and enhanced variants share theirs — so a hash resolves to
	// the same normalised name either way.
	PlugNames(hashes []uint32) (map[uint32]string, error)
}
