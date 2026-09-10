package collections

import (
	"context"
	"time"
)

// LiveAvailabilityReader is the entire surface the complete service consumes
// from Weekly: item hash → the name of the vendor selling that item right now.
//
// Consumer-side and deliberately narrow. Weekly owns rotating-vendor policy —
// character resolution, the empty-character fallback, exact vendor names, and
// Xûr's tie precedence — and Collections learns none of it. Satisfied in
// production by *weekly.Service.
//
// This is the half of the dependency cycle ADR 0018 breaks by constructing
// Collections in two stages: Weekly reads missing items from
// [MembershipAnalysis], and the outer [Service] — built after Weekly — reads
// availability back out of Weekly.
type LiveAvailabilityReader interface {
	LiveVendorItemHashes(ctx context.Context, membershipType int, membershipID, bungieToken string) map[uint32]string
}

// RefreshParticipant is one owner of membership-scoped cached data that a
// manual refresh must invalidate.
//
// The participant owns its own cache keys and its own publication fence;
// Collections names the membership and never learns how that owner stores it or
// how it keeps its own in-flight work from republishing. Satisfied by
// *characters.Service and *records.Service.
//
// Invalidating is required to be one transition, not a bare cache delete: after
// it returns, work that began before it may still answer its own request but
// may no longer become reusable. Without that, a refresh could delete an entry
// only for an older in-flight load to refill it moments later.
type RefreshParticipant interface {
	InvalidateCache(membershipType int, membershipID string)
}

// Membership is a Destiny membership: the platform and the id together.
//
// The pair is the identity. The same numeric id can name memberships on
// different platforms, which is why every operation here takes both and why the
// HTTP boundary authorizes both before calling in.
type Membership struct {
	MembershipType int
	MembershipID   string
}

// MembershipRequest is one already-authorized read for a membership.
//
// The service receives a membership whose ownership the HTTP boundary has
// already established and a Bungie access token it has already resolved. It
// owns collected-state and projection semantics, not JWT, token-store, or HTTP
// policy.
type MembershipRequest struct {
	MembershipType int
	MembershipID   string
	AccessToken    string
}

// Summary is the cheap complete outcome: the presentation tree with rolled-up
// counts, the four-category totals, and when the ownership profile was fetched.
//
// It carries no item surface at all and performs no availability work. That is
// the point of it being its own type rather than a fuller result with fields
// left empty — "no items" here is the answer, not a missing part of one.
type Summary struct {
	Tree      []CollectionNode
	Totals    CategorySummary
	FetchedAt time.Time
}

// Full is the complete outcome: everything [Summary] carries, plus each node's
// item hashes and one [CollectionItem] per catalogued item.
//
// FetchedAt remains the ownership-profile fetch time. It is not restamped by
// the availability join, which is a separate, best-effort, per-request fact.
type Full struct {
	Tree      []CollectionNode
	Items     []CollectionItem
	Totals    CategorySummary
	FetchedAt time.Time
}

// CollectionItem is one item in a complete collection: what the item is,
// whether this membership has it, and where it can be bought right now.
//
// The three facts have three different owners and three different lifetimes —
// Items owns the canonical facts, the membership's profile owns Collected, and
// Weekly's vendor read owns AvailableFrom for the length of one request — and
// combining them here is what stops the grid, the wish list, and item detail
// from disagreeing about the same item.
//
// AvailableFrom is empty when the item is not on sale, which is also what an
// unavailable vendor read produces. Availability is best effort: it never fails
// an otherwise valid result, so "not on sale" and "we could not ask" are
// deliberately the same answer.
type CollectionItem struct {
	Item          DestinyItem
	Collected     bool
	AvailableFrom string
}

// Service is the complete, handler-facing Collections capability: the outer of
// the two construction stages in ADR 0018.
//
// It is constructed after Weekly and owns the operations a caller actually
// wants — a summary, a complete collection, and a membership data refresh —
// rather than the ingredients of one. Everything it depends on is required:
// there is no setter, no late-bound closure, no per-request availability
// callback, and no nil-degraded production path.
type Service struct {
	analysis   *MembershipAnalysis
	live       LiveAvailabilityReader
	characters RefreshParticipant
	records    RefreshParticipant
}

// NewService constructs the complete service around its required dependencies.
//
// A nil dependency is a composition error rather than a degraded mode, so it
// fails at startup instead of surfacing as a nil-pointer panic on the first
// request that needs it.
func NewService(analysis *MembershipAnalysis, live LiveAvailabilityReader, characters, records RefreshParticipant) *Service {
	if analysis == nil {
		panic("collections: membership analysis is required")
	}
	if live == nil {
		panic("collections: live availability reader is required")
	}
	if characters == nil {
		panic("collections: characters refresh participant is required")
	}
	if records == nil {
		panic("collections: records refresh participant is required")
	}
	return &Service{analysis: analysis, live: live, characters: characters, records: records}
}

// GetSummary returns the counted tree and category totals for one membership.
//
// It does no item or availability work, which is what makes it the cheap read
// the Dashboard hero can take on every visit.
func (s *Service) GetSummary(ctx context.Context, req MembershipRequest) (Summary, error) {
	a, err := s.analysis.getAnalysis(ctx, req.MembershipType, req.MembershipID, req.AccessToken)
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		Tree:      a.tree.overlayCounts(a.owned),
		Totals:    buildCategorySummary(a.catalog, a.owned),
		FetchedAt: a.fetchedAt,
	}, nil
}

// GetFull returns the complete collection for one membership.
//
// The core result is resolved first; only then is live availability asked for,
// so a vendor read never delays or endangers ownership data. The overlay is
// applied to this request's fresh items and is never written back into the
// longer-lived cached analysis.
//
// Items come out in ascending item-hash order, inherited from the Items catalog
// (ADR 0015), and that order is contract: the HTTP adapter walks this slice in
// place to build every item-derived wire field.
func (s *Service) GetFull(ctx context.Context, req MembershipRequest) (Full, error) {
	a, err := s.analysis.getAnalysis(ctx, req.MembershipType, req.MembershipID, req.AccessToken)
	if err != nil {
		return Full{}, err
	}

	// Best effort by construction: a nil or empty map from an unavailable
	// vendor read simply stamps nothing, and the result below is unaffected.
	live := s.live.LiveVendorItemHashes(ctx, req.MembershipType, req.MembershipID, req.AccessToken)

	// Reading availability out of the catalog loop is what intersects it with
	// tracked items: an item hash the vendor sells but the catalog does not
	// carry has nowhere to land, so it is dropped rather than announced.
	collected := make([]CollectionItem, 0, len(a.catalog))
	for _, f := range a.catalog {
		collected = append(collected, CollectionItem{
			Item:          destinyItem(f),
			Collected:     a.owned[f.ItemHash],
			AvailableFrom: live[f.ItemHash],
		})
	}

	return Full{
		Tree:      a.tree.overlayWithItems(a.owned),
		Items:     collected,
		Totals:    buildCategorySummary(a.catalog, a.owned),
		FetchedAt: a.fetchedAt,
	}, nil
}

// RefreshMembership invalidates every backend owner of this membership's cached
// data, so the next read of any of them fetches fresh data.
//
// It is an invalidation command, not an eager Bungie fetch. The participant set
// is exactly the three owners that hold membership-scoped upstream data:
// Collections' own analysis, Characters, and Records. Weekly's vendor caches,
// the wish list, preferences, shared manifest state, and Items are deliberately
// not participants — they are either not membership-scoped or govern their own
// rotation.
//
// All three are advanced before this returns, and each advance is one
// owner-local transition of a generation plus an eviction. That is what makes
// the promise enforceable rather than merely stated: once this returns, no
// pre-refresh work anywhere in the participant set can install itself, so the
// next read of any of the three genuinely fetches fresh data. Work already in
// flight still answers the request that started it — it just is not left
// behind. Deleting the three cache entries without the fence would let an older
// load refill one immediately after the refresh reported success.
//
// The error return is the contract's room for a participant that can fail.
// None of the current three can, so today this only ever returns nil.
func (s *Service) RefreshMembership(ctx context.Context, membership Membership) error {
	s.analysis.InvalidateCache(membership.MembershipType, membership.MembershipID)
	s.characters.InvalidateCache(membership.MembershipType, membership.MembershipID)
	s.records.InvalidateCache(membership.MembershipType, membership.MembershipID)
	return nil
}
