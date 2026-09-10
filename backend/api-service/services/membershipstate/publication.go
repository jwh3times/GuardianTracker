// Package membershipstate coordinates the cached data an owner holds for one
// Destiny membership across a user-requested membership data refresh.
//
// Deleting a cache entry is an event, not a freshness guarantee:
//
//  1. a request begins loading a membership's data;
//  2. the user asks for a refresh, and the owner deletes the cache entry;
//  3. the older request finishes and installs its pre-refresh result into the
//     now-empty entry.
//
// The refresh returned successfully and the next read still sees pre-refresh
// data. A Publication closes that window. Work captures the generation it
// started under and can only install a result while that generation is still
// current. The losing request still returns the coherent result it loaded to
// whoever asked for it — it just may not leave it behind for anyone else.
//
// This is [manifeststate.Publication]'s sibling, and deliberately the same
// shape: the two fence different axes of the same problem. A manifest
// publication has one generation for the whole owner and advances when a new
// Manifest is installed; a membership publication keeps a generation per
// membership and advances when that one membership is refreshed. An owner may
// hold both, in which case it takes them in a fixed order — see the note on
// [Attempt.Publish].
//
// Each participating owner holds its own Publication and guards only its own
// cached data. The general cache contract in ADR 0013 stays unaware of refresh
// generations, exactly as it stays unaware of manifest generations.
//
// See [ADR 0018](../../../../docs/adr/0018-own-complete-membership-collections.md).
package membershipstate

import "sync"

// membership is the map key: a Destiny membership is the platform and the id
// together, so the generation is kept per pair rather than per id.
type membership struct {
	membershipType int
	membershipID   string
}

// Publication is one owner's per-membership generation fence over the data it
// caches for a membership.
//
// Begin, Publish, and Advance are serialized against each other, so a
// generation cannot advance in the middle of a publish and an invalidation
// cannot interleave with one.
//
// The zero value is not usable; construct one with [New].
type Publication struct {
	mu          sync.Mutex
	generations map[membership]uint64
	invalidate  func(membershipType int, membershipID string)
}

// New returns a Publication whose invalidate callback runs on every refresh,
// inside the same critical section that advances that membership's generation.
// That atomicity is the point: a loader can never observe a generation that has
// moved but a cache entry that has not yet been cleared.
//
// invalidate must be bounded, non-blocking, and non-reentrant — it must not
// call back into this Publication. It cannot fail, because there is no coherent
// state to return to if clearing half-succeeds.
//
// invalidate may be nil for an owner with nothing to clear.
func New(invalidate func(membershipType int, membershipID string)) *Publication {
	return &Publication{
		generations: make(map[membership]uint64),
		invalidate:  invalidate,
	}
}

// Begin captures the current generation for a unit of work on one membership.
//
// Call it before reading the cache, not just before writing: the captured
// generation has to predate the read for the fence to cover the whole
// read-load-publish sequence.
func (p *Publication) Begin(membershipType int, membershipID string) Attempt {
	key := membership{membershipType: membershipType, membershipID: membershipID}

	p.mu.Lock()
	defer p.mu.Unlock()
	return Attempt{p: p, key: key, generation: p.generations[key]}
}

// Advance records that this membership was refreshed, retiring every
// outstanding attempt for it and running the owner's invalidation.
//
// Only the named membership is affected. One user refreshing their own data
// must not retire work in flight for anyone else, which is the whole reason the
// generation is per membership rather than per owner.
//
// A membership that has never been refreshed has no map entry and an implicit
// generation of zero, so this is also where an entry is first created. The map
// therefore holds one small entry per membership that has actually been
// refreshed in this process's lifetime, not one per membership ever seen.
func (p *Publication) Advance(membershipType int, membershipID string) {
	key := membership{membershipType: membershipType, membershipID: membershipID}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.generations[key]++
	if p.invalidate != nil {
		p.invalidate(membershipType, membershipID)
	}
}

// Attempt is a unit of work's claim on the generation its membership was at
// when it started.
//
// The zero Attempt is inert: it publishes nothing. That makes an Attempt that
// was never obtained from [Publication.Begin] fail closed rather than install
// state against a generation it never captured.
type Attempt struct {
	p          *Publication
	key        membership
	generation uint64
}

// Publish runs commit and reports true only if this attempt's generation is
// still current for its membership. Otherwise commit does not run and Publish
// reports false.
//
// commit runs while the publication is held, so it must be as short as a cache
// write and must not call back into this Publication. Doing the work inside
// commit is what makes the check and the install atomic — testing the
// generation and then writing outside the lock would leave exactly the race
// this type exists to close.
//
// An owner fenced on both axes nests the two publishes, manifest outside and
// membership inside, and must keep that order everywhere. The order is safe to
// fix because neither invalidation callback reaches into the other publication:
// a manifest invalidation only drops manifest-derived state, and a membership
// invalidation only deletes that membership's cache entry.
//
// A false result is not an error. The caller loaded a coherent result for a
// generation that has since been retired; it may still return that result to
// whoever initiated the work. It may not leave it behind as reusable state.
func (a Attempt) Publish(commit func()) bool {
	if a.p == nil {
		return false
	}

	a.p.mu.Lock()
	defer a.p.mu.Unlock()

	if a.generation != a.p.generations[a.key] {
		return false
	}
	if commit != nil {
		commit()
	}
	return true
}

// Current reports whether the generation this attempt captured is still the
// current one for its membership.
//
// Publish answers that question for work that is about to install a result.
// Current answers it for a result that was already installed and outlives its
// own load, the way a cached value stamped with the attempt it was built under
// can say later, and cheaply, whether a refresh has since retired it.
//
// The zero Attempt is never current, so a value carrying no attempt rebuilds
// rather than passing as fresh.
func (a Attempt) Current() bool {
	if a.p == nil {
		return false
	}

	a.p.mu.Lock()
	defer a.p.mu.Unlock()

	return a.generation == a.p.generations[a.key]
}
