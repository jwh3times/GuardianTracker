package wishlist

import (
	"context"
	"errors"
	"testing"
	"time"

	"guardian-tracker/api-service/services/items"
	"guardian-tracker/api-service/services/sources"
)

// fakeVendors stands in for Weekly's live-availability capability. calls counts
// invocations so a test can prove a read never asked at all, which is a
// different claim from asking and getting nothing back.
type fakeVendors struct {
	live      map[uint32]string
	calls     int
	gotType   int
	gotID     string
	gotToken  string
	gotCalled bool
}

func (f *fakeVendors) LiveVendorItemHashes(_ context.Context, membershipType int, membershipID, token string) map[uint32]string {
	f.calls++
	f.gotType, f.gotID, f.gotToken, f.gotCalled = membershipType, membershipID, token, true
	return f.live
}

// fakeCredentials stands in for the token store.
type fakeCredentials struct {
	token string
	err   error
	calls int
}

func (f *fakeCredentials) GetValidToken(string) (string, error) {
	f.calls++
	return f.token, f.err
}

// richItems resolves the given hashes to fully-populated facts, so a test can
// tell a real projection from a tombstone's stand-in.
type richItems struct {
	facts map[uint32]items.AcquisitionFacts
	err   error
	calls int
	batch [][]uint32
}

func (r *richItems) Lookup(_ context.Context, hashes []uint32) (map[uint32]items.AcquisitionFacts, error) {
	r.calls++
	r.batch = append(r.batch, hashes)
	if r.err != nil {
		return nil, r.err
	}
	out := map[uint32]items.AcquisitionFacts{}
	for _, hash := range hashes {
		if f, ok := r.facts[hash]; ok {
			out[hash] = f
		}
	}
	return out, nil
}

func fatebringer() items.AcquisitionFacts {
	return items.AcquisitionFacts{
		ItemHash: 100,
		Name:     "Fatebringer",
		ItemType: "Hand Cannon",
		Rarity:   "Legendary",
		Icon:     "/i/fatebringer.png",
		AcquisitionSources: []sources.AcquisitionSource{
			{Text: "Vault of Glass raid", Difficulty: sources.Challenging, RaidDungeon: true},
		},
	}
}

func membership() Membership {
	return Membership{MembershipType: 3, MembershipID: member}
}

func storedAt(id EntryID, hash uint32) StoredEntry {
	return StoredEntry{
		ID:        id,
		ItemHash:  hash,
		Priority:  PriorityHigh,
		Notes:     "nice roll",
		CreatedAt: time.Date(2026, 7, 18, 18, 0, 0, 0, time.UTC),
	}
}

// newFixtureService wires a complete service whose repository holds the given
// rows and whose Items seam knows Fatebringer.
func newFixtureService(t *testing.T, stored []StoredEntry, vendors *fakeVendors, creds CredentialReader) (*Service, *richItems, *fakeRepository) {
	t.Helper()
	repo := &fakeRepository{entries: stored}
	lookup := &richItems{facts: map[uint32]items.AcquisitionFacts{100: fatebringer()}}
	return NewService(NewEntries(repo, lookup), lookup, vendors, creds), lookup, repo
}

// A complete entry carries the canonical Items facts, not a second opinion
// derived here. This is what stops the wish list and the collection grid from
// describing the same item differently.
func TestList_CompletesEntriesFromCanonicalItemFacts(t *testing.T) {
	svc, _, _ := newFixtureService(t, []StoredEntry{storedAt(1, 100)}, &fakeVendors{}, nil)

	got, err := svc.List(context.Background(), membership())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("entries = %d, want 1", len(got))
	}

	entry := got[0]
	if !entry.Item.Known() {
		t.Fatal("a known item must not be a tombstone")
	}
	if entry.Item.Name() != "Fatebringer" || entry.Item.ItemType() != "Hand Cannon" || entry.Item.Rarity() != "Legendary" {
		t.Errorf("projection = %s/%s/%s", entry.Item.Name(), entry.Item.ItemType(), entry.Item.Rarity())
	}
	if entry.Item.Icon() != "/i/fatebringer.png" {
		t.Errorf("icon = %q", entry.Item.Icon())
	}
	if srcs := entry.Item.AcquisitionSources(); len(srcs) != 1 || srcs[0].Text != "Vault of Glass raid" || !srcs[0].RaidDungeon {
		t.Errorf("sources = %+v; the canonical union and its facets must survive", srcs)
	}
	// The user's own data is untouched by completion.
	if entry.Priority != PriorityHigh || entry.Notes != "nice roll" || entry.ID != 1 {
		t.Errorf("stored fields = %+v", entry)
	}
}

// An item the Manifest no longer describes is a tombstone: the row survives
// with the user's metadata and the visible stand-in projection. Dropping it
// would discard work the user did.
func TestList_UnknownItemBecomesATombstone(t *testing.T) {
	svc, _, _ := newFixtureService(t, []StoredEntry{storedAt(1, 999)}, &fakeVendors{}, nil)

	got, err := svc.List(context.Background(), membership())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("entries = %d, want the row to survive", len(got))
	}

	entry := got[0]
	if entry.Item.Known() {
		t.Fatal("an item absent from a successful lookup must be a tombstone")
	}
	if entry.Item.Name() != "Unknown Item" || entry.Item.ItemType() != "Item" || entry.Item.Rarity() != "Common" {
		t.Errorf("tombstone projection = %s/%s/%s", entry.Item.Name(), entry.Item.ItemType(), entry.Item.Rarity())
	}
	if entry.Item.Icon() != "" {
		t.Errorf("tombstone icon = %q, want empty", entry.Item.Icon())
	}
	if srcs := entry.Item.AcquisitionSources(); srcs == nil || len(srcs) != 0 {
		t.Errorf("tombstone sources = %+v, want an empty non-nil collection", srcs)
	}
	if entry.Priority != PriorityHigh || entry.Notes != "nice roll" {
		t.Error("a tombstone must retain the user's own metadata")
	}
}

// The third outcome, and the one that used to be invisible: a lookup that could
// not be performed is not a verdict about anybody's items. Reporting it as a
// list of tombstones would tell every user their whole wish list had vanished.
func TestList_ItemLookupFailureFailsTheOperation(t *testing.T) {
	repo := &fakeRepository{entries: []StoredEntry{storedAt(1, 100)}}
	lookup := &richItems{err: errors.New("manifest is mid-swap")}
	svc := NewService(NewEntries(repo, lookup), lookup, &fakeVendors{}, nil)

	_, err := svc.List(context.Background(), membership())

	if !errors.Is(err, ErrItemsUnavailable) {
		t.Fatalf("err = %v, want ErrItemsUnavailable", err)
	}
}

// Every stored hash is resolved in one batched lookup. A wish list of forty
// items is one Manifest round trip, not forty.
func TestList_ResolvesEveryItemInOneBatch(t *testing.T) {
	stored := []StoredEntry{storedAt(1, 100), storedAt(2, 999), storedAt(3, 100)}
	svc, lookup, _ := newFixtureService(t, stored, &fakeVendors{}, nil)

	if _, err := svc.List(context.Background(), membership()); err != nil {
		t.Fatalf("List: %v", err)
	}

	if lookup.calls != 1 {
		t.Fatalf("item lookups = %d, want exactly 1 for the whole list", lookup.calls)
	}
	if len(lookup.batch[0]) != len(stored) {
		t.Errorf("batch = %v, want every stored hash", lookup.batch[0])
	}
}

// An empty wish list has nothing to complete, so it must not spend an Item
// lookup, a credential resolution, or a vendor call describing nothing.
func TestList_EmptyShortCircuitsCompletion(t *testing.T) {
	vendors := &fakeVendors{}
	creds := &fakeCredentials{token: "tok"}
	svc, lookup, _ := newFixtureService(t, nil, vendors, creds)

	got, err := svc.List(context.Background(), membership())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if got == nil || len(got) != 0 {
		t.Errorf("entries = %+v, want an empty non-nil slice", got)
	}
	if lookup.calls != 0 || creds.calls != 0 || vendors.calls != 0 {
		t.Errorf("empty list spent lookups=%d credentials=%d vendors=%d, want none",
			lookup.calls, creds.calls, vendors.calls)
	}
}

// Availability is asked for once, with the caller's own membership pair and
// credential, and lands only on the items it actually names.
//
// The vendor selling something nobody saved is here to pin that it cannot
// appear anywhere in the result — completion reads availability by saved item
// hash, so an unsaved hash has nothing to attach to.
func TestList_StampsAvailabilityOntoTheItemsItNames(t *testing.T) {
	vendors := &fakeVendors{live: map[uint32]string{
		100: "Xûr",
		777: "Banshee-44", // nobody saved this
	}}
	svc, _, _ := newFixtureService(t, []StoredEntry{storedAt(1, 100)}, vendors, &fakeCredentials{token: "tok"})

	got, err := svc.List(context.Background(), membership())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if got[0].AvailableFrom != "Xûr" {
		t.Errorf("availableFrom = %q, want Xûr", got[0].AvailableFrom)
	}
	if vendors.calls != 1 {
		t.Errorf("vendor reads = %d, want exactly 1", vendors.calls)
	}
	if vendors.gotType != 3 || vendors.gotID != member || vendors.gotToken != "tok" {
		t.Errorf("vendor read used (%d, %q, %q); the caller's own pair and credential must be passed through",
			vendors.gotType, vendors.gotID, vendors.gotToken)
	}
	for _, entry := range got {
		if entry.ItemHash == 777 || entry.AvailableFrom == "Banshee-44" {
			t.Errorf("an item nobody saved reached the result: %+v", entry)
		}
	}
}

// Credential resolution is best effort: without one, the read still happens
// unauthenticated so public Xûr availability survives.
func TestList_CredentialFailureFallsBackToAnUnauthenticatedRead(t *testing.T) {
	vendors := &fakeVendors{live: map[uint32]string{100: "Xûr"}}
	creds := &fakeCredentials{err: errors.New("reauthorization required")}
	svc, _, _ := newFixtureService(t, []StoredEntry{storedAt(1, 100)}, vendors, creds)

	got, err := svc.List(context.Background(), membership())
	if err != nil {
		t.Fatalf("a credential failure must not fail the wish list: %v", err)
	}
	if !vendors.gotCalled || vendors.gotToken != "" {
		t.Errorf("vendor read token = %q, want an empty token rather than a skipped read", vendors.gotToken)
	}
	if got[0].AvailableFrom != "Xûr" {
		t.Errorf("public availability = %q, want it to survive a credential failure", got[0].AvailableFrom)
	}
}

// Availability failing entirely stamps nothing and fails nothing.
func TestList_UnavailableVendorsDoNotFailTheOperation(t *testing.T) {
	svc, _, _ := newFixtureService(t, []StoredEntry{storedAt(1, 100)}, &fakeVendors{live: nil}, nil)

	got, err := svc.List(context.Background(), membership())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got[0].AvailableFrom != "" {
		t.Errorf("availableFrom = %q, want empty", got[0].AvailableFrom)
	}
	if !got[0].Item.Known() {
		t.Error("the item must still be complete when only availability was unavailable")
	}
}

// Add returns the same complete shape List does, so a freshly saved item
// renders identically to a reloaded one.
func TestAdd_ReturnsACompleteEntry(t *testing.T) {
	repo := &fakeRepository{returnedStored: storedAt(7, 100)}
	lookup := &richItems{facts: map[uint32]items.AcquisitionFacts{100: fatebringer()}}
	svc := NewService(NewEntries(repo, lookup), lookup, &fakeVendors{live: map[uint32]string{100: "Xûr"}}, nil)

	got, err := svc.Add(context.Background(), membership(), AddCommand{ItemHash: 100, Priority: PriorityHigh})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if got.ID != 7 || !got.Item.Known() || got.Item.Name() != "Fatebringer" {
		t.Errorf("entry = %+v", got)
	}
	if got.AvailableFrom != "Xûr" {
		t.Errorf("availableFrom = %q, want the same completion List performs", got.AvailableFrom)
	}
}

// Update resolves the stored row's item state before writing. A confirmed
// tombstone is still the user's entry, so its metadata may be edited.
func TestUpdate_AllowsEditingAConfirmedTombstone(t *testing.T) {
	updated := storedAt(1, 999)
	updated.Notes = "still want it"
	repo := &fakeRepository{entries: []StoredEntry{storedAt(1, 999)}, returnedStored: updated}
	lookup := &richItems{facts: map[uint32]items.AcquisitionFacts{100: fatebringer()}}
	svc := NewService(NewEntries(repo, lookup), lookup, &fakeVendors{}, nil)

	notes := "still want it"
	got, err := svc.Update(context.Background(), membership(), 1, UpdateCommand{Notes: &notes})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Item.Known() {
		t.Error("the item is absent from the Manifest; it must stay a tombstone")
	}
	if got.Notes != "still want it" {
		t.Errorf("notes = %q, want the edit to have been applied", got.Notes)
	}
	if repo.writes == 0 {
		t.Error("editing a tombstone must still write")
	}
}

// A lookup that failed is not a tombstone. Update must write nothing, so a
// transient failure cannot be mistaken for a confirmed one afterwards.
func TestUpdate_ItemLookupFailureWritesNothing(t *testing.T) {
	repo := &fakeRepository{entries: []StoredEntry{storedAt(1, 100)}}
	lookup := &richItems{facts: map[uint32]items.AcquisitionFacts{100: fatebringer()}}
	svc := NewService(NewEntries(repo, lookup), lookup, &fakeVendors{}, nil)
	lookup.err = errors.New("manifest is mid-swap")

	notes := "edited"
	_, err := svc.Update(context.Background(), membership(), 1, UpdateCommand{Notes: &notes})

	if !errors.Is(err, ErrItemsUnavailable) {
		t.Fatalf("err = %v, want ErrItemsUnavailable", err)
	}
	if repo.writes != 0 {
		t.Errorf("writes = %d, want none — an unreadable Manifest must not mutate the wish list", repo.writes)
	}
}

func TestUpdate_MissingEntryIsNotFound(t *testing.T) {
	svc, _, repo := newFixtureService(t, []StoredEntry{storedAt(1, 100)}, &fakeVendors{}, nil)

	_, err := svc.Update(context.Background(), membership(), 42, UpdateCommand{})

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if repo.writes != 0 {
		t.Errorf("writes = %d, want none", repo.writes)
	}
}

// Deletions and priority changes describe no item, so they must not spend an
// Item lookup, a credential, or a vendor call.
func TestMutationsWithoutAResultSkipCompletion(t *testing.T) {
	cases := map[string]func(*Service) error{
		"Remove": func(s *Service) error {
			return s.Remove(context.Background(), membership(), 1)
		},
		"DeleteMany": func(s *Service) error {
			_, err := s.DeleteMany(context.Background(), membership(), []EntryID{1})
			return err
		},
		"SetPriorityMany": func(s *Service) error {
			_, err := s.SetPriorityMany(context.Background(), membership(), []EntryID{1}, PriorityLow)
			return err
		},
	}
	for name, op := range cases {
		t.Run(name, func(t *testing.T) {
			vendors := &fakeVendors{}
			creds := &fakeCredentials{}
			svc, lookup, _ := newFixtureService(t, []StoredEntry{storedAt(1, 100)}, vendors, creds)

			if err := op(svc); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if lookup.calls != 0 || creds.calls != 0 || vendors.calls != 0 {
				t.Errorf("%s spent lookups=%d credentials=%d vendors=%d, want none",
					name, lookup.calls, creds.calls, vendors.calls)
			}
		})
	}
}

// A caller must not be able to reach into cached Items state through a returned
// entry and change what everyone else reads.
func TestAcquisitionSources_AreNotSharedWithTheItemsCache(t *testing.T) {
	facts := fatebringer()
	state := KnownItem(facts)

	first := state.AcquisitionSources()
	first[0].Text = "mutated"

	if state.AcquisitionSources()[0].Text != "Vault of Glass raid" {
		t.Error("mutating a returned source collection reached the underlying facts")
	}
	if facts.AcquisitionSources[0].Text != "Vault of Glass raid" {
		t.Error("mutating a returned source collection reached the caller's own facts")
	}
}

// The core and the availability reader are required. A missing one is a
// composition error that must stop the process at startup.
func TestNewService_RequiresItsCoreDependencies(t *testing.T) {
	lookup := &richItems{}
	entries := NewEntries(&fakeRepository{}, lookup)

	cases := []struct {
		name    string
		entries *Entries
		items   ItemLookup
		live    LiveAvailabilityReader
	}{
		{"no entries core", nil, lookup, &fakeVendors{}},
		{"no item lookup", entries, nil, &fakeVendors{}},
		{"no live availability", entries, lookup, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("NewService must panic on a missing required dependency")
				}
			}()
			NewService(tc.entries, tc.items, tc.live, nil)
		})
	}
}

// A missing credential reader is a real degraded mode, not a composition error:
// availability still resolves against public vendor data.
func TestNewService_AcceptsAMissingCredentialReader(t *testing.T) {
	svc, _, _ := newFixtureService(t, []StoredEntry{storedAt(1, 100)}, &fakeVendors{live: map[uint32]string{100: "Xûr"}}, nil)

	got, err := svc.List(context.Background(), membership())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got[0].AvailableFrom != "Xûr" {
		t.Errorf("availableFrom = %q, want public availability without a credential reader", got[0].AvailableFrom)
	}
}
