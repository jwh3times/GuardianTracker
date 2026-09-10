package wishlist

import (
	"context"
	"slices"
	"time"

	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/sources"
)

// LiveAvailabilityReader is the entire surface the complete service consumes
// from Weekly: item hash → the name of the vendor selling that item right now.
//
// Consumer-side and deliberately narrow, and the same capability Collections
// reads, so a wish-listed item and the same item in the collection grid cannot
// disagree about where it is on sale. Satisfied in production by
// *weekly.Service.
//
// This is the half of the dependency cycle ADR 0019 breaks by constructing the
// wish list in two stages: Weekly reads saved item hashes from [Entries], and
// the outer [Service] — built after Weekly — reads availability back out.
type LiveAvailabilityReader interface {
	LiveVendorItemHashes(ctx context.Context, membershipType int, membershipID, bungieToken string) map[uint32]string
}

// CredentialReader resolves a membership's current Bungie access token.
//
// Deliberately best effort. An authenticated vendor read sees the rotating
// vendors; an unauthenticated one still sees public Xûr availability. Losing
// the credential therefore narrows what availability can be found rather than
// failing the wish list, so this returning an error is an ordinary outcome and
// not a fault. Satisfied by *auth.TokenStore.
type CredentialReader interface {
	GetValidToken(membershipID string) (string, error)
}

// Membership is a Destiny membership: the platform and the id together.
//
// Persistence is keyed by id alone — a wish list belongs to the Guardian
// Tracker user, not to one platform — but resolving live availability needs the
// pair, because vendor inventory is read per platform. Taking the pair at this
// boundary keeps the HTTP adapter from having to know which operations need
// which half.
type Membership struct {
	MembershipType int
	MembershipID   string
}

// The visible projection of an item the Manifest no longer describes. These are
// the values the wish list has always shown for such an entry, and they live
// here rather than at the HTTP boundary because what a tombstone looks like is
// part of what a tombstone means.
const (
	tombstoneName     = "Unknown Item"
	tombstoneItemType = "Item"
	tombstoneRarity   = "Common"
)

// ItemState is what the wish list knows about a saved entry's item: either the
// canonical facts from Items, or a tombstone for an item a successful lookup
// did not contain.
//
// The two are a tagged pair rather than "facts that might be empty" because
// they mean different things and one of them is not a failure. An entry whose
// item has left the Manifest is still the user's entry — their priority and
// notes are intact — and dropping the row, or reporting it as an error, would
// discard work the user did. A lookup that could not be performed at all is a
// third outcome and never reaches here: it fails the operation instead.
//
// The accessors below always answer, so a caller renders an entry without
// branching on provenance and cannot read facts a tombstone does not have.
type ItemState struct {
	known bool
	facts items.AcquisitionFacts
}

// KnownItem is the state of an entry whose item the Manifest still describes.
func KnownItem(facts items.AcquisitionFacts) ItemState {
	return ItemState{known: true, facts: facts}
}

// UnknownItemTombstone is the state of an entry whose item a successful lookup
// did not contain.
func UnknownItemTombstone() ItemState {
	return ItemState{}
}

// Known reports whether the Manifest still describes this entry's item.
func (s ItemState) Known() bool { return s.known }

// Name is the item's name, or the tombstone's stand-in.
func (s ItemState) Name() string {
	if s.known {
		return s.facts.Name
	}
	return tombstoneName
}

// ItemType is the item's slot-specific type, or the tombstone's stand-in.
func (s ItemState) ItemType() string {
	if s.known {
		return s.facts.ItemType
	}
	return tombstoneItemType
}

// Rarity is the item's tier name, or the tombstone's stand-in.
func (s ItemState) Rarity() string {
	if s.known {
		return s.facts.Rarity
	}
	return tombstoneRarity
}

// Icon is the item's icon path, empty for a tombstone.
func (s ItemState) Icon() string {
	if s.known {
		return s.facts.Icon
	}
	return ""
}

// AcquisitionSources is the item's canonical source union (ADR 0015), or an
// empty collection for a tombstone.
//
// Never nil, so the wire carries `[]` rather than `null` exactly as it always
// has, and freshly allocated, so a caller cannot reach back into cached Items
// state through it.
func (s ItemState) AcquisitionSources() []sources.AcquisitionSource {
	if !s.known {
		return []sources.AcquisitionSource{}
	}
	return slices.Clone(s.facts.AcquisitionSources)
}

// Entry is one complete wish list entry: what the user saved, what the item is
// now, and whether it can be bought right now.
//
// AvailableFrom is empty when the item is not on sale, which is also what an
// unavailable vendor read produces. Availability is best effort, so "not on
// sale" and "we could not ask" are deliberately the same answer.
type Entry struct {
	ID            EntryID
	ItemHash      uint32
	Priority      Priority
	Notes         string
	CreatedAt     time.Time
	Item          ItemState
	AvailableFrom string
}

// Service is the complete, handler-facing wish list capability: the outer of
// the two construction stages in ADR 0019.
//
// It is constructed after Weekly and owns the operations a caller actually
// wants — a complete list, a complete save, a complete edit — rather than the
// ingredients of one. Entries and the availability reader are required; there
// is no setter, late-bound closure, or Gin-supplied completion callback.
type Service struct {
	entries     *Entries
	items       ItemLookup
	live        LiveAvailabilityReader
	credentials CredentialReader
}

// NewService constructs the complete service around its required core and
// availability reader.
//
// A nil core or availability reader is a composition error rather than a
// degraded mode, so it fails at startup instead of surfacing as a nil-pointer
// panic on the first request that needs it. The credential reader may be nil:
// without it, availability resolves against public vendor data only, which is a
// real degraded mode rather than a mistake.
// The Item lookup is passed in rather than borrowed from the core. Both stages
// consume it — Entries to refuse saving an item that does not exist, this
// service to say what a saved item is — and naming that here keeps the
// dependency visible instead of reaching through Entries for a port it holds
// for its own reasons.
func NewService(entries *Entries, itemLookup ItemLookup, live LiveAvailabilityReader, credentials CredentialReader) *Service {
	if entries == nil {
		panic("wishlist: entries core is required")
	}
	if itemLookup == nil {
		panic("wishlist: item lookup is required")
	}
	if live == nil {
		panic("wishlist: live availability reader is required")
	}
	return &Service{entries: entries, items: itemLookup, live: live, credentials: credentials}
}

// List returns the membership's complete saved entries in repository order.
//
// Every stored row is returned. An item the Manifest no longer describes
// becomes a tombstone rather than a dropped row, because the row is the user's
// work and its absence from the Manifest is a fact about the item, not about
// them. An item lookup that fails outright is neither: it fails the whole
// operation, so a transient failure can never be mistaken for every item having
// vanished at once.
func (s *Service) List(ctx context.Context, m Membership) ([]Entry, error) {
	stored, err := s.entries.List(ctx, m.MembershipID)
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		// Nothing saved short-circuits the Item lookup, the credential
		// resolution, and the vendor call: three round trips that could only
		// ever describe an empty list.
		return []Entry{}, nil
	}
	return s.complete(ctx, m, stored)
}

// Add saves one item and returns it complete, in the same shape List returns.
//
// The item must exist: Entries refuses an item a successful lookup does not
// contain, and writes nothing when the lookup itself fails.
func (s *Service) Add(ctx context.Context, m Membership, cmd AddCommand) (Entry, error) {
	stored, err := s.entries.Add(ctx, m.MembershipID, cmd)
	if err != nil {
		return Entry{}, err
	}
	return s.completeOne(ctx, m, stored)
}

// Update applies a partial patch and returns the entry complete.
//
// The stored row and its current item state are resolved *before* the write.
// That ordering is what separates the two reasons an item might not be known:
// a confirmed tombstone may still have its priority and notes edited, because
// the entry is still the user's; a lookup that failed writes nothing, because
// the answer is unknown rather than negative. Doing the write first would make
// those indistinguishable after the fact.
func (s *Service) Update(ctx context.Context, m Membership, id EntryID, patch UpdateCommand) (Entry, error) {
	existing, err := s.storedEntry(ctx, m.MembershipID, id)
	if err != nil {
		return Entry{}, err
	}
	state, err := s.itemState(ctx, existing.ItemHash)
	if err != nil {
		return Entry{}, err
	}

	updated, err := s.entries.Update(ctx, m.MembershipID, id, patch)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		ID:            updated.ID,
		ItemHash:      updated.ItemHash,
		Priority:      updated.Priority,
		Notes:         updated.Notes,
		CreatedAt:     updated.CreatedAt,
		Item:          state,
		AvailableFrom: s.availability(ctx, m)[updated.ItemHash],
	}, nil
}

// Remove deletes one entry. It resolves no item facts, credentials, or vendors:
// nothing about a deletion needs them.
func (s *Service) Remove(ctx context.Context, m Membership, id EntryID) error {
	return s.entries.Remove(ctx, m.MembershipID, id)
}

// DeleteMany removes every listed entry the membership owns. Missing and
// foreign ids are skipped and counted rather than failing the command.
func (s *Service) DeleteMany(ctx context.Context, m Membership, ids []EntryID) (BulkResult, error) {
	return s.entries.RemoveMany(ctx, m.MembershipID, ids)
}

// SetPriorityMany applies one priority to every listed entry the membership
// owns, with the same skip semantics as DeleteMany.
func (s *Service) SetPriorityMany(ctx context.Context, m Membership, ids []EntryID, priority Priority) (BulkResult, error) {
	return s.entries.SetPriorityMany(ctx, m.MembershipID, ids, priority)
}

// complete turns stored rows into complete entries: one batched item lookup for
// the whole list, then one vendor read for the same set.
func (s *Service) complete(ctx context.Context, m Membership, stored []StoredEntry) ([]Entry, error) {
	hashes := make([]uint32, len(stored))
	for i, entry := range stored {
		hashes[i] = entry.ItemHash
	}

	// One lookup for the whole list rather than one per entry. A wish list of
	// forty items used to be forty Manifest round trips.
	facts, err := s.items.Lookup(ctx, hashes)
	if err != nil {
		return nil, ErrItemsUnavailable
	}
	live := s.availability(ctx, m)

	out := make([]Entry, len(stored))
	for i, entry := range stored {
		out[i] = Entry{
			ID:            entry.ID,
			ItemHash:      entry.ItemHash,
			Priority:      entry.Priority,
			Notes:         entry.Notes,
			CreatedAt:     entry.CreatedAt,
			Item:          stateFor(facts, entry.ItemHash),
			AvailableFrom: live[entry.ItemHash],
		}
	}
	return out, nil
}

func (s *Service) completeOne(ctx context.Context, m Membership, stored StoredEntry) (Entry, error) {
	completed, err := s.complete(ctx, m, []StoredEntry{stored})
	if err != nil {
		return Entry{}, err
	}
	return completed[0], nil
}

// itemState resolves one item's current state, distinguishing a tombstone from
// a lookup that could not be performed.
func (s *Service) itemState(ctx context.Context, itemHash uint32) (ItemState, error) {
	facts, err := s.items.Lookup(ctx, []uint32{itemHash})
	if err != nil {
		return ItemState{}, ErrItemsUnavailable
	}
	return stateFor(facts, itemHash), nil
}

// stateFor is the one place the three item outcomes are distinguished. A
// successful lookup that contains the hash is a known item; a successful lookup
// that does not is a tombstone. The third — a lookup that failed — never
// reaches here.
func stateFor(facts map[uint32]items.AcquisitionFacts, itemHash uint32) ItemState {
	if f, ok := facts[itemHash]; ok {
		return KnownItem(f)
	}
	return UnknownItemTombstone()
}

// storedEntry finds one of the membership's stored rows.
//
// It reads through the list rather than a dedicated repository lookup because
// the wish list is small and bounded, and adding a by-id read to the
// persistence port would widen it for one caller. A row that is missing or
// belongs to someone else is the same answer, for the same reason it is
// everywhere else in this package: distinguishing them would confirm another
// user's entry ids.
func (s *Service) storedEntry(ctx context.Context, membershipID string, id EntryID) (StoredEntry, error) {
	stored, err := s.entries.List(ctx, membershipID)
	if err != nil {
		return StoredEntry{}, err
	}
	i := slices.IndexFunc(stored, func(entry StoredEntry) bool { return entry.ID == id })
	if i < 0 {
		return StoredEntry{}, ErrNotFound
	}
	return stored[i], nil
}

// availability resolves which of these items are on sale right now.
//
// Best effort throughout: a credential that cannot be resolved falls back to an
// unauthenticated read, which still sees public Xûr inventory, and a vendor
// read that returns nothing simply stamps nothing. Neither fails the operation
// that asked.
func (s *Service) availability(ctx context.Context, m Membership) map[uint32]string {
	token := ""
	if s.credentials != nil {
		if resolved, err := s.credentials.GetValidToken(m.MembershipID); err == nil {
			token = resolved
		}
	}

	// Returned whole rather than intersected with the saved items. Completion
	// reads it by saved item hash, so a vendor selling something nobody here
	// saved simply never gets looked up — filtering first would be work that
	// cannot change any answer.
	return s.live.LiveVendorItemHashes(ctx, m.MembershipType, m.MembershipID, token)
}
