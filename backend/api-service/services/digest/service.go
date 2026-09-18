// Package digest owns the since-last-visit digest (ADR 0023): the visit
// boundary, the additive collected-item diff, and the privacy gate on the
// snapshot write. It holds no presentation, wording, or ordering — the wire
// shape belongs to its HTTP adapter.
package digest

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"guardian-tracker/api-service/observability"
	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/items"
)

// VisitGap is how long a request can be silent before the next one starts a
// new visit (ADR 0023).
const VisitGap = 2 * time.Hour

// Status is Digest's typed outcome, in the style of Characters'
// current-activity states (ready/idle/unavailable/unknown): which branch
// produced the result is visible to the caller rather than inferred from an
// empty list.
type Status string

const (
	// StatusReady means the digest was computed for this visit and may
	// legitimately report zero newly acquired items.
	StatusReady Status = "ready"
	// StatusFirstVisit means no prior snapshot exists for this membership —
	// there is nothing to diff against. The caller must say "tracking starts
	// now", never an empty digest and never a dump of the whole collection.
	StatusFirstVisit Status = "first-visit"
	// StatusUnavailable means no digest could be produced for this visit:
	// there is no database, the Bungie read failed, or the profile is
	// private. Any previous snapshot is left standing untouched.
	StatusUnavailable Status = "unavailable"
)

// AcquiredItem is one collectible the membership has acquired since its
// previous visit, projected the way other item surfaces already are.
type AcquiredItem struct {
	ItemHash uint32
	Name     string
	Icon     string
	ItemType string
}

// Result is the complete outcome of one digest read.
type Result struct {
	Status Status
	// VisitStartedAt is always populated: a visit is under way for this
	// membership as of this call regardless of whether a digest could be
	// computed for it.
	VisitStartedAt time.Time
	// PreviousVisitAt is when the membership was last seen active — the end
	// of the previous visit. Nil on a first visit and whenever no previous
	// snapshot exists yet to report a boundary against.
	PreviousVisitAt *time.Time
	// Acquired is always non-nil so a caller never has to distinguish a nil
	// slice from an explicitly empty one.
	Acquired []AcquiredItem
}

// Snapshot is the durable visit-clock-plus-baseline state Repository stores.
type Snapshot struct {
	LastActivityAt  time.Time
	VisitStartedAt  time.Time
	OwnedItemHashes []uint32
	SnapshotTakenAt time.Time
}

// ErrUnavailable distinguishes unavailable persistence from another failure,
// mirroring preferences.ErrUnavailable.
var ErrUnavailable = errors.New("digest persistence unavailable")

// Repository is the membership-keyed persistence port used by Service.
type Repository interface {
	// Get returns the stored visit/snapshot state. found is false only when
	// no row exists yet for this membership — a genuinely new account.
	Get(ctx context.Context, membershipID string) (snap Snapshot, found bool, err error)

	// TouchActivity updates only the visit-clock columns. It never touches
	// the snapshot, so it is safe to call after a failed or private read —
	// ADR 0023's privacy gate covers the snapshot alone. visitStartedAt is
	// nil when this request continues the existing visit rather than
	// starting a new one.
	TouchActivity(ctx context.Context, membershipID string, lastActivityAt time.Time, visitStartedAt *time.Time) error

	// Save atomically replaces the whole row: the visit clock and the
	// snapshot together. Called only when a new visit begins and the read is
	// public and successful.
	Save(ctx context.Context, membershipID string, snap Snapshot) error
}

// CollectionsReader is the narrow Collections surface Digest depends on: the
// membership's currently owned item hashes plus the Bungie
// profile-collectibles privacy value the same read carried
// (bungie.CollectiblesPrivacy*) — from Collections' own cached analysis
// (ADR 0023), never a forced refresh. Satisfied by
// *collections.MembershipAnalysis.
type CollectionsReader interface {
	CollectedState(ctx context.Context, membershipType int, membershipID, accessToken string) (owned map[uint32]bool, privacy int, fetchedAt time.Time, err error)
}

// ItemLookup is the narrow Items surface Digest depends on: canonical facts
// for the acquired item hashes, used only for display. Satisfied by
// *items.Service.
type ItemLookup interface {
	Lookup(ctx context.Context, hashes []uint32) (map[uint32]items.AcquisitionFacts, error)
}

// Service owns the since-last-visit digest.
//
// It keeps one process-local, per-membership record of the digest computed
// for the membership's current visit. That record — not a database column —
// is what makes a repeated request inside the same visit return an identical
// result without a second Bungie read: the pre-visit baseline is consumed the
// moment the new visit's snapshot replaces it in storage, so nothing durable
// is left to recompute the same diff from afterward. This mirrors the rest of
// the codebase's in-process, non-durable caches (Collections' and Weekly's
// analysis caches, the revocation cache) — acceptable because Guardian
// Tracker is a single local-only process. If the record is missing (e.g. a
// restart mid-visit), the visit degrades to reporting zero acquisitions
// rather than fabricating a diff it no longer has the baseline for.
type Service struct {
	repo        Repository
	collections CollectionsReader
	items       ItemLookup
	now         func() time.Time

	mu     sync.Mutex
	frozen map[string]Result
}

// NewService constructs Digest around its required dependencies. A nil
// dependency is a composition error, so it fails at startup rather than
// surfacing as a nil-pointer panic on the first request that needs it.
func NewService(repo Repository, collections CollectionsReader, items ItemLookup) *Service {
	if repo == nil {
		panic("digest: repository is required")
	}
	if collections == nil {
		panic("digest: collections reader is required")
	}
	if items == nil {
		panic("digest: item lookup is required")
	}
	return &Service{
		repo:        repo,
		collections: collections,
		items:       items,
		now:         func() time.Time { return time.Now().UTC() },
		frozen:      make(map[string]Result),
	}
}

// NewServiceWithClock is NewService with an injectable clock, for tests and
// the E2E fixed-time harness other services already support.
func NewServiceWithClock(repo Repository, collections CollectionsReader, items ItemLookup, now func() time.Time) *Service {
	s := NewService(repo, collections, items)
	s.now = now
	return s
}

func membershipKey(membershipType int, membershipID string) string {
	return fmt.Sprintf("%d:%s", membershipType, membershipID)
}

// GetDigest returns the since-last-visit digest for one membership, advancing
// the visit clock as a side effect exactly as ADR 0023 describes.
func (s *Service) GetDigest(ctx context.Context, membershipType int, membershipID, accessToken string) (Result, error) {
	now := s.now()
	key := membershipKey(membershipType, membershipID)

	existing, found, err := s.repo.Get(ctx, membershipID)
	if errors.Is(err, ErrUnavailable) {
		return Result{Status: StatusUnavailable, VisitStartedAt: now, Acquired: []AcquiredItem{}}, nil
	}
	if err != nil {
		return Result{}, fmt.Errorf("digest: read state: %w", err)
	}

	newVisit := !found || now.Sub(existing.LastActivityAt) > VisitGap

	if !newVisit {
		return s.continueVisit(ctx, membershipID, existing, now, key), nil
	}
	return s.startVisit(ctx, membershipType, membershipID, accessToken, existing, found, now, key)
}

// continueVisit handles a request that falls inside an already-started visit:
// the visit clock still advances, but the digest itself is frozen and no
// Bungie call is made.
func (s *Service) continueVisit(ctx context.Context, membershipID string, existing Snapshot, now time.Time, key string) Result {
	if err := s.repo.TouchActivity(ctx, membershipID, now, nil); err != nil && !errors.Is(err, ErrUnavailable) {
		observability.Logger(ctx).WarnContext(ctx, "digest: visit-clock touch failed", observability.Err(err))
	}

	s.mu.Lock()
	frozen, ok := s.frozen[key]
	s.mu.Unlock()
	if ok {
		return frozen
	}

	// The process-local record is gone (most likely a restart mid-visit).
	// The pre-visit baseline this visit's diff needs was already consumed
	// when the new snapshot replaced it in storage, so the true diff cannot
	// be reconstructed. Degrading to zero acquisitions never invents data;
	// it only under-reports for the remainder of this one visit.
	observability.Logger(ctx).WarnContext(ctx, "digest: frozen visit result unavailable; degrading to zero acquisitions")
	return Result{Status: StatusReady, VisitStartedAt: existing.VisitStartedAt, Acquired: []AcquiredItem{}}
}

// startVisit handles a request that begins a new visit: the Bungie read
// happens here, gated by privacy, and (on success) the snapshot is replaced.
func (s *Service) startVisit(ctx context.Context, membershipType int, membershipID, accessToken string, existing Snapshot, found bool, now time.Time, key string) (Result, error) {
	owned, privacy, _, err := s.collections.CollectedState(ctx, membershipType, membershipID, accessToken)
	if err != nil {
		observability.Logger(ctx).WarnContext(ctx, "digest: collections read failed; snapshot left standing", observability.Err(err))
		return s.unavailableVisit(ctx, membershipID, existing, found, now, key), nil
	}
	if privacy != bungie.CollectiblesPrivacyPublic {
		return s.unavailableVisit(ctx, membershipID, existing, found, now, key), nil
	}

	newSnapshot := sortedOwnedHashes(owned)

	var result Result
	if !found {
		result = Result{Status: StatusFirstVisit, VisitStartedAt: now, Acquired: []AcquiredItem{}}
	} else {
		acquiredHashes := setDifference(newSnapshot, existing.OwnedItemHashes)
		logIfShrunk(ctx, membershipType, existing.OwnedItemHashes, newSnapshot)

		acquired, err := s.projectAcquired(ctx, acquiredHashes)
		if err != nil {
			return Result{}, fmt.Errorf("digest: project acquired items: %w", err)
		}
		prev := existing.LastActivityAt
		result = Result{
			Status:          StatusReady,
			VisitStartedAt:  now,
			PreviousVisitAt: &prev,
			Acquired:        acquired,
		}
	}

	if err := s.repo.Save(ctx, membershipID, Snapshot{
		LastActivityAt:  now,
		VisitStartedAt:  now,
		OwnedItemHashes: newSnapshot,
		SnapshotTakenAt: now,
	}); err != nil {
		if errors.Is(err, ErrUnavailable) {
			return Result{Status: StatusUnavailable, VisitStartedAt: now, Acquired: []AcquiredItem{}}, nil
		}
		return Result{}, fmt.Errorf("digest: save snapshot: %w", err)
	}

	s.freeze(key, result)
	return result, nil
}

// unavailableVisit is the shared tail of startVisit's two failure branches
// (a failed read and a private profile): the visit clock advances when there
// is a row to advance it on, but the snapshot is never touched.
func (s *Service) unavailableVisit(ctx context.Context, membershipID string, existing Snapshot, found bool, now time.Time, key string) Result {
	if found {
		// There is a previous row: advance the visit clock onto it, but
		// never touch the snapshot columns (the privacy gate covers exactly
		// those). Best-effort — a touch failure must not fail the request,
		// since there is already a definite answer (unavailable) to return.
		if err := s.repo.TouchActivity(ctx, membershipID, now, &now); err != nil && !errors.Is(err, ErrUnavailable) {
			observability.Logger(ctx).WarnContext(ctx, "digest: visit-clock update failed", observability.Err(err))
		}
	}
	// !found: no row exists, and one cannot be created without a snapshot
	// value (NOT NULL) — so this request leaves no trace. The next request,
	// however soon, retries the read from scratch, which is exactly what
	// lets the account recover as soon as the profile is public again.

	result := Result{Status: StatusUnavailable, VisitStartedAt: now, Acquired: []AcquiredItem{}}
	if found {
		prev := existing.LastActivityAt
		result.PreviousVisitAt = &prev
	}
	if found {
		s.freeze(key, result)
	}
	return result
}

func (s *Service) freeze(key string, result Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.frozen[key] = result
}

func (s *Service) projectAcquired(ctx context.Context, hashes []uint32) ([]AcquiredItem, error) {
	if len(hashes) == 0 {
		return []AcquiredItem{}, nil
	}
	facts, err := s.items.Lookup(ctx, hashes)
	if err != nil {
		return nil, err
	}
	out := make([]AcquiredItem, 0, len(hashes))
	for _, h := range hashes { // hashes is already ascending
		f, ok := facts[h]
		if !ok {
			// Acquired but the manifest no longer (or not yet) describes it:
			// drop it rather than render a guess.
			continue
		}
		out = append(out, AcquiredItem{ItemHash: f.ItemHash, Name: f.Name, Icon: f.Icon, ItemType: f.ItemType})
	}
	return out, nil
}

// sortedOwnedHashes projects the owned map into a deterministic ascending
// slice, the shape Repository persists.
func sortedOwnedHashes(owned map[uint32]bool) []uint32 {
	out := make([]uint32, 0, len(owned))
	for h, isOwned := range owned {
		if isOwned {
			out = append(out, h)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// setDifference returns the elements of a not present in b, preserving a's
// order. Both are treated as sets.
func setDifference(a, b []uint32) []uint32 {
	inB := make(map[uint32]struct{}, len(b))
	for _, h := range b {
		inB[h] = struct{}{}
	}
	out := make([]uint32, 0)
	for _, h := range a {
		if _, ok := inB[h]; !ok {
			out = append(out, h)
		}
	}
	return out
}

// logIfShrunk reports the anomalous-but-allowed case where a successful
// public read collects fewer items than the previous snapshot (ADR 0023: the
// owner-identified case is deleting all characters). The new baseline is
// still accepted; this is observability only.
func logIfShrunk(ctx context.Context, membershipType int, previous, current []uint32) {
	removed := setDifference(previous, current)
	if len(removed) == 0 {
		return
	}
	observability.Logger(ctx).WarnContext(ctx, "digest: collected set shrank; accepting new baseline",
		"membership_type", membershipType, "removed_count", len(removed))
}
