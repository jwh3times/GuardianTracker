package wishlist

import (
	"context"
	"errors"
	"strings"
	"testing"

	"guardian-tracker/api-service/services/items"
)

// fakeRepository records every write it is asked to make, so a test can prove
// a refusal wrote nothing rather than merely returning an error.
type fakeRepository struct {
	entries []StoredEntry
	err     error
	writes  int
	reads   int

	addedCommand   AddCommand
	updatedID      EntryID
	updatedPatch   UpdateCommand
	removedID      EntryID
	bulkIDs        []EntryID
	bulkPriority   Priority
	affected       int
	returnedStored StoredEntry
}

func (f *fakeRepository) List(context.Context, string) ([]StoredEntry, error) {
	f.reads++
	return f.entries, f.err
}

func (f *fakeRepository) Add(_ context.Context, _ string, entry AddCommand) (StoredEntry, error) {
	f.writes++
	f.addedCommand = entry
	if f.err != nil {
		return StoredEntry{}, f.err
	}
	return f.returnedStored, nil
}

func (f *fakeRepository) Update(_ context.Context, _ string, id EntryID, patch UpdateCommand) (StoredEntry, error) {
	f.writes++
	f.updatedID, f.updatedPatch = id, patch
	if f.err != nil {
		return StoredEntry{}, f.err
	}
	return f.returnedStored, nil
}

func (f *fakeRepository) Remove(_ context.Context, _ string, id EntryID) error {
	f.writes++
	f.removedID = id
	return f.err
}

func (f *fakeRepository) RemoveMany(_ context.Context, _ string, ids []EntryID) (int, error) {
	f.writes++
	f.bulkIDs = ids
	return f.affected, f.err
}

func (f *fakeRepository) SetPriorityMany(_ context.Context, _ string, ids []EntryID, priority Priority) (int, error) {
	f.writes++
	f.bulkIDs, f.bulkPriority = ids, priority
	return f.affected, f.err
}

// fakeItems stands in for the Items seam: known hashes resolve, everything else
// is absent, and err makes the lookup itself fail.
type fakeItems struct {
	known []uint32
	err   error
	calls int
}

func (f *fakeItems) Lookup(_ context.Context, hashes []uint32) (map[uint32]items.AcquisitionFacts, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	out := map[uint32]items.AcquisitionFacts{}
	for _, hash := range hashes {
		for _, known := range f.known {
			if hash == known {
				out[hash] = items.AcquisitionFacts{ItemHash: hash, Name: "Known"}
			}
		}
	}
	return out, nil
}

const member = "member-1"

func TestAdd_DefaultsOmittedPriorityToMedium(t *testing.T) {
	repo := &fakeRepository{}
	entries := NewEntries(repo, &fakeItems{known: []uint32{100}})

	if _, err := entries.Add(context.Background(), member, AddCommand{ItemHash: 100}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if repo.addedCommand.Priority != PriorityMedium {
		t.Errorf("persisted priority = %q, want %q", repo.addedCommand.Priority, PriorityMedium)
	}
}

func TestAdd_RejectsAnInvalidPriorityWithoutWriting(t *testing.T) {
	repo := &fakeRepository{}
	itemLookup := &fakeItems{known: []uint32{100}}
	entries := NewEntries(repo, itemLookup)

	_, err := entries.Add(context.Background(), member, AddCommand{ItemHash: 100, Priority: "SOON"})

	if !errors.Is(err, ErrInvalidPriority) {
		t.Fatalf("Add error = %v, want ErrInvalidPriority", err)
	}
	if repo.writes != 0 || itemLookup.calls != 0 {
		t.Errorf("a malformed request cost %d writes and %d lookups", repo.writes, itemLookup.calls)
	}
}

// The limit is 500 characters, and it has to be counted the way PostgreSQL's
// constraint counts it. Measuring bytes rejected notes the database would have
// accepted — every non-ASCII character costs two to four of them.
func TestAdd_CountsNotesInCodePointsNotBytes(t *testing.T) {
	cases := []struct {
		name    string
		notes   string
		wantErr bool
	}{
		{"500 ascii", strings.Repeat("a", MaxNoteRunes), false},
		{"501 ascii", strings.Repeat("a", MaxNoteRunes+1), true},
		{"500 four-byte runes", strings.Repeat("🔫", MaxNoteRunes), false},
		{"501 four-byte runes", strings.Repeat("🔫", MaxNoteRunes+1), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			entries := NewEntries(repo, &fakeItems{known: []uint32{100}})

			_, err := entries.Add(context.Background(), member, AddCommand{ItemHash: 100, Notes: tc.notes})

			if tc.wantErr {
				if !errors.Is(err, ErrNotesTooLong) {
					t.Fatalf("Add error = %v, want ErrNotesTooLong", err)
				}
				if repo.writes != 0 {
					t.Error("an over-long note was persisted")
				}
				return
			}
			if err != nil {
				t.Fatalf("Add error = %v, want a note of %d characters to be accepted", err, MaxNoteRunes)
			}
		})
	}
}

// Three item outcomes must stay distinguishable: confirmed present, confirmed
// absent, and no answer at all. Collapsing the last two would let an unreadable
// manifest look like a verdict about the item.
func TestAdd_SeparatesAnUnknownItemFromAnUnreadableLookup(t *testing.T) {
	t.Run("unknown item", func(t *testing.T) {
		repo := &fakeRepository{}
		entries := NewEntries(repo, &fakeItems{known: []uint32{999}})

		_, err := entries.Add(context.Background(), member, AddCommand{ItemHash: 100})

		if !errors.Is(err, ErrUnknownItem) {
			t.Fatalf("Add error = %v, want ErrUnknownItem", err)
		}
		if repo.writes != 0 {
			t.Error("an unknown item was persisted")
		}
	})

	t.Run("lookup failure", func(t *testing.T) {
		boom := errors.New("manifest not ready")
		repo := &fakeRepository{}
		entries := NewEntries(repo, &fakeItems{err: boom})

		_, err := entries.Add(context.Background(), member, AddCommand{ItemHash: 100})

		if !errors.Is(err, ErrItemsUnavailable) {
			t.Fatalf("Add error = %v, want ErrItemsUnavailable", err)
		}
		if errors.Is(err, ErrUnknownItem) {
			t.Error("an unreadable lookup was reported as a verdict about the item")
		}
		if !errors.Is(err, boom) {
			t.Error("the underlying cause was dropped")
		}
		if repo.writes != 0 {
			t.Error("an unverified item was persisted")
		}
	})
}

// An entry already saved stays editable after its item leaves the manifest —
// otherwise a user could not fix or remove their own note about it.
func TestUpdate_DoesNotReCheckTheItem(t *testing.T) {
	repo := &fakeRepository{}
	itemLookup := &fakeItems{}
	entries := NewEntries(repo, itemLookup)

	priority := PriorityHigh
	if _, err := entries.Update(context.Background(), member, 7, UpdateCommand{Priority: &priority}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if itemLookup.calls != 0 {
		t.Error("Update consulted the manifest; a stored entry's item is not in question")
	}
	if repo.updatedID != 7 || repo.updatedPatch.Priority == nil || *repo.updatedPatch.Priority != PriorityHigh {
		t.Errorf("patch reached storage as id=%d %+v", repo.updatedID, repo.updatedPatch)
	}
}

func TestUpdate_RejectsInvalidPatchesWithoutWriting(t *testing.T) {
	bad := Priority("SOON")
	long := strings.Repeat("a", MaxNoteRunes+1)
	cases := []struct {
		name  string
		patch UpdateCommand
		want  error
	}{
		{"invalid priority", UpdateCommand{Priority: &bad}, ErrInvalidPriority},
		{"over-long notes", UpdateCommand{Notes: &long}, ErrNotesTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			entries := NewEntries(repo, &fakeItems{})

			_, err := entries.Update(context.Background(), member, 1, tc.patch)

			if !errors.Is(err, tc.want) {
				t.Fatalf("Update error = %v, want %v", err, tc.want)
			}
			if repo.writes != 0 {
				t.Error("an invalid patch was persisted")
			}
		})
	}
}

// An empty patch is legal: it means "change nothing", and storage leaves both
// fields alone.
func TestUpdate_EmptyPatchCarriesNoFields(t *testing.T) {
	repo := &fakeRepository{}
	entries := NewEntries(repo, &fakeItems{})

	if _, err := entries.Update(context.Background(), member, 3, UpdateCommand{}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if repo.updatedPatch.Priority != nil || repo.updatedPatch.Notes != nil {
		t.Errorf("empty patch reached storage as %+v", repo.updatedPatch)
	}
}

func TestListItemHashes_ProjectsInRepositoryOrder(t *testing.T) {
	repo := &fakeRepository{entries: []StoredEntry{
		{ID: 1, ItemHash: 300},
		{ID: 2, ItemHash: 100},
	}}
	entries := NewEntries(repo, &fakeItems{})

	hashes, err := entries.ListItemHashes(context.Background(), member)
	if err != nil {
		t.Fatalf("ListItemHashes: %v", err)
	}
	if len(hashes) != 2 || hashes[0] != 300 || hashes[1] != 100 {
		t.Errorf("hashes = %v, want [300 100] — the repository's order, not a sorted one", hashes)
	}
}

// "We cannot read your wish list" must not arrive as "you saved nothing".
// Weekly chooses to degrade; that choice belongs to the consumer, not here.
func TestListItemHashes_ReportsFailureRatherThanAnEmptyList(t *testing.T) {
	entries := NewEntries(&fakeRepository{err: ErrUnavailable}, &fakeItems{})

	hashes, err := entries.ListItemHashes(context.Background(), member)

	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListItemHashes error = %v, want ErrUnavailable", err)
	}
	if hashes != nil {
		t.Errorf("hashes = %v, want none alongside an error", hashes)
	}
}

// A client that sends the same entry twice is asking about one entry — not
// two, and not a larger command than it thinks.
func TestBulkCommands_DeduplicateBeforeValidatingAndCounting(t *testing.T) {
	repo := &fakeRepository{affected: 2}
	entries := NewEntries(repo, &fakeItems{})

	result, err := entries.RemoveMany(context.Background(), member, []EntryID{1, 2, 1, 2, 3})
	if err != nil {
		t.Fatalf("RemoveMany: %v", err)
	}

	if len(repo.bulkIDs) != 3 {
		t.Errorf("storage received %v, want three unique ids", repo.bulkIDs)
	}
	if result.Updated != 2 || result.Skipped != 1 {
		t.Errorf("result = %+v, want {Updated:2 Skipped:1}", result)
	}
}

func TestBulkCommands_RejectEmptyAndOversizedRequests(t *testing.T) {
	tooMany := make([]EntryID, MaxBulkEntries+1)
	for i := range tooMany {
		tooMany[i] = EntryID(i + 1)
	}
	cases := []struct {
		name string
		ids  []EntryID
		want error
	}{
		{"none", nil, ErrNoEntries},
		{"all duplicates of nothing", []EntryID{}, ErrNoEntries},
		{"over the cap", tooMany, ErrTooManyEntries},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepository{}
			entries := NewEntries(repo, &fakeItems{})

			if _, err := entries.RemoveMany(context.Background(), member, tc.ids); !errors.Is(err, tc.want) {
				t.Fatalf("RemoveMany error = %v, want %v", err, tc.want)
			}
			if repo.writes != 0 {
				t.Error("a refused bulk command still reached storage")
			}
		})
	}
}

// A bulk request whose ids are all duplicates of each other still names one
// entry, so it is a legal command rather than an empty one.
func TestBulkCommands_RepeatedIDCollapsesToOneEntry(t *testing.T) {
	repo := &fakeRepository{affected: 1}
	entries := NewEntries(repo, &fakeItems{})

	result, err := entries.RemoveMany(context.Background(), member, []EntryID{5, 5, 5})
	if err != nil {
		t.Fatalf("RemoveMany: %v", err)
	}
	if len(repo.bulkIDs) != 1 || result.Updated != 1 || result.Skipped != 0 {
		t.Errorf("ids = %v, result = %+v, want one entry updated", repo.bulkIDs, result)
	}
}

func TestSetPriorityMany_ValidatesThePriorityBeforeTouchingStorage(t *testing.T) {
	repo := &fakeRepository{}
	entries := NewEntries(repo, &fakeItems{})

	_, err := entries.SetPriorityMany(context.Background(), member, []EntryID{1}, "SOON")

	if !errors.Is(err, ErrInvalidPriority) {
		t.Fatalf("SetPriorityMany error = %v, want ErrInvalidPriority", err)
	}
	if repo.writes != 0 {
		t.Error("an invalid priority reached storage")
	}
}

func TestSetPriorityMany_CarriesThePriorityThrough(t *testing.T) {
	repo := &fakeRepository{affected: 1}
	entries := NewEntries(repo, &fakeItems{})

	if _, err := entries.SetPriorityMany(context.Background(), member, []EntryID{1}, PriorityUrgent); err != nil {
		t.Fatalf("SetPriorityMany: %v", err)
	}
	if repo.bulkPriority != PriorityUrgent {
		t.Errorf("storage received priority %q, want %q", repo.bulkPriority, PriorityUrgent)
	}
}

// A storage failure is not a partial success: no count is invented for it.
func TestBulkCommands_ReportStorageFailuresWithoutACount(t *testing.T) {
	repo := &fakeRepository{err: ErrUnavailable, affected: 5}
	entries := NewEntries(repo, &fakeItems{})

	result, err := entries.RemoveMany(context.Background(), member, []EntryID{1, 2})

	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("RemoveMany error = %v, want ErrUnavailable", err)
	}
	if result != (BulkResult{}) {
		t.Errorf("result = %+v, want no counts alongside a failure", result)
	}
}

// The handler maps request-shaped refusals to 400 and everything else to a
// retry-or-report status, so the split has to be stated somewhere testable.
func TestIsValidationError_CoversRequestShapedRefusalsOnly(t *testing.T) {
	validation := []error{ErrInvalidPriority, ErrNotesTooLong, ErrNoEntries, ErrTooManyEntries, ErrUnknownItem}
	for _, err := range validation {
		if !IsValidationError(err) {
			t.Errorf("%v is a request-shaped refusal", err)
		}
	}
	other := []error{ErrUnavailable, ErrNotFound, ErrDuplicate, ErrItemsUnavailable, errors.New("boom")}
	for _, err := range other {
		if IsValidationError(err) {
			t.Errorf("%v is not a validation error; mapping it to 400 would blame the client", err)
		}
	}
}
