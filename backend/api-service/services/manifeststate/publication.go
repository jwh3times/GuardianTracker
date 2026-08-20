// Package manifeststate coordinates the reusable state a module derives from
// the Destiny manifest.
//
// ADR 0010 already tells observers when a new manifest was installed, and they
// invalidate on that signal. Invalidation alone does not prevent stale
// publication:
//
//  1. a request begins loading data from manifest generation A;
//  2. generation B is installed and the owning observer clears its state;
//  3. the old request finishes and republishes its generation-A result into the
//     now-current cache or index.
//
// A Publication closes that window. Work captures the generation it started
// under and can only install a result while that generation is still current.
// The losing request still returns the coherent result it loaded to whoever
// asked for it — it just may not leave it behind for anyone else.
//
// Each affected owner holds its own Publication and guards only its own
// manifest-derived state. This is deliberate: the general cache contract in
// ADR 0013 stays unaware of manifest generations, and the participant/observer
// split in ADR 0010 stays intact.
//
// See [ADR 0014](../../../../docs/adr/0014-own-manifest-derived-publication.md).
package manifeststate

import (
	"errors"
	"sync"
)

// ErrEmptyVersion is returned by [Publication.Advance] when handed an empty
// manifest version. An empty version means the caller could not determine what
// is installed, which is not a state any owner should invalidate against.
var ErrEmptyVersion = errors.New("manifeststate: manifest version is empty")

// Publication is one owner's generation fence over its manifest-derived state.
//
// Begin, Publish, and Advance are serialized against each other, so a
// generation cannot advance in the middle of a publish and an invalidation
// cannot interleave with one.
//
// The zero value is not usable; construct one with [New].
type Publication struct {
	mu         sync.Mutex
	generation uint64
	version    string
	invalidate func()
}

// New returns a Publication whose invalidate callback runs on every genuine
// version change, inside the same critical section that advances the
// generation. That atomicity is the point: a loader can never observe a
// generation that has moved but state that has not yet been cleared.
//
// invalidate must be bounded, non-blocking, and non-reentrant — it must not
// call back into this Publication. It cannot fail, because there is no coherent
// state to return to if clearing half-succeeds; an owner that needs to reject a
// version does so by validating before calling Advance.
//
// invalidate may be nil for an owner whose only manifest-derived state is
// installed through [Attempt.Publish] and therefore has nothing to clear.
func New(invalidate func()) *Publication {
	return &Publication{generation: 1, invalidate: invalidate}
}

// Begin captures the current generation for a unit of work.
//
// Call it before reading any manifest-derived state, not just before writing:
// the captured generation has to predate the read for the fence to cover the
// whole read-load-publish sequence.
func (p *Publication) Begin() Attempt {
	p.mu.Lock()
	defer p.mu.Unlock()
	return Attempt{p: p, generation: p.generation}
}

// Advance records that a new manifest version is installed, retiring every
// outstanding attempt and running the owner's invalidation.
//
// An empty version is rejected with [ErrEmptyVersion] and changes nothing.
// Repeating the current version is an idempotent no-op: it does not advance the
// generation, does not invalidate, and leaves outstanding attempts able to
// publish. Any other version — including one seen before — advances the
// generation, because work abandoned under an older generation must never
// become current again just by the version string coming back around.
//
// The signature matches bungie.ManifestObserver.OnVersionChanged, so an owner's
// observer method is usually one line.
func (p *Publication) Advance(version string) error {
	if version == "" {
		return ErrEmptyVersion
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if version == p.version {
		return nil
	}
	p.generation++
	p.version = version
	if p.invalidate != nil {
		p.invalidate()
	}
	return nil
}

// Attempt is a unit of work's claim on the generation it started under.
//
// The zero Attempt is inert: it publishes nothing. That makes an Attempt that
// was never obtained from [Publication.Begin] fail closed rather than install
// state against a generation it never captured.
type Attempt struct {
	p          *Publication
	generation uint64
}

// Publish runs commit and reports true only if this attempt's generation is
// still current. Otherwise commit does not run and Publish reports false.
//
// commit runs while the publication is held, so it must be as short as a cache
// write and must not call back into this Publication. Doing the work inside
// commit is what makes the check and the install atomic — testing the
// generation and then writing outside the lock would leave exactly the race
// this type exists to close.
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

	if a.generation != a.p.generation {
		return false
	}
	if commit != nil {
		commit()
	}
	return true
}
