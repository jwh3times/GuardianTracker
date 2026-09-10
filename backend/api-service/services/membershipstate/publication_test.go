package membershipstate

import (
	"sync"
	"testing"
)

func TestPublish_InstallsWorkStartedUnderTheCurrentGeneration(t *testing.T) {
	p := New(nil)

	attempt := p.Begin(3, "member-1")

	installed := false
	if !attempt.Publish(func() { installed = true }) {
		t.Error("Publish reported false for an attempt whose generation is current")
	}
	if !installed {
		t.Error("commit did not run")
	}
}

// The whole point: work that began before a refresh may not install itself
// after one.
func TestPublish_RefusesWorkStartedBeforeARefresh(t *testing.T) {
	p := New(nil)
	attempt := p.Begin(3, "member-1")

	p.Advance(3, "member-1")

	installed := false
	if attempt.Publish(func() { installed = true }) {
		t.Error("Publish reported true for an attempt retired by a refresh")
	}
	if installed {
		t.Error("commit ran for an attempt retired by a refresh")
	}
}

// A refresh is one user's request about one membership. It must not retire work
// in flight for anybody else — that is the reason the generation is kept per
// membership rather than per owner.
func TestAdvance_RetiresOnlyTheNamedMembership(t *testing.T) {
	p := New(nil)

	cases := []struct {
		name           string
		membershipType int
		membershipID   string
		wantPublished  bool
	}{
		{"the refreshed membership", 3, "member-1", false},
		{"a different id on the same platform", 3, "member-2", true},
		{"the same id on a different platform", 2, "member-1", true},
	}

	attempts := make([]Attempt, len(cases))
	for i, tc := range cases {
		attempts[i] = p.Begin(tc.membershipType, tc.membershipID)
	}

	p.Advance(3, "member-1")

	for i, tc := range cases {
		if got := attempts[i].Publish(nil); got != tc.wantPublished {
			t.Errorf("%s: Publish = %v, want %v", tc.name, got, tc.wantPublished)
		}
	}
}

// Invalidation runs inside the same critical section that advances the
// generation, so no loader can ever see a moved generation with the old entry
// still in place.
func TestAdvance_InvalidatesInsideTheGenerationChange(t *testing.T) {
	var (
		p                *Publication
		sawMovedAtInvoke bool
		gotType          int
		gotID            string
	)
	p = New(func(membershipType int, membershipID string) {
		gotType, gotID = membershipType, membershipID
		// Reading the generation through the map directly: calling back into
		// the publication would deadlock, which is itself the contract.
		sawMovedAtInvoke = p.generations[membership{membershipType, membershipID}] == 1
	})

	p.Advance(3, "member-1")

	if gotType != 3 || gotID != "member-1" {
		t.Errorf("invalidate got (%d, %q), want (3, \"member-1\")", gotType, gotID)
	}
	if !sawMovedAtInvoke {
		t.Error("the generation had not advanced when invalidate ran; the two are not one transition")
	}
}

// Repeated refreshes keep retiring older work rather than cycling back to a
// generation an abandoned attempt still holds.
func TestAdvance_IsMonotonic(t *testing.T) {
	p := New(nil)
	first := p.Begin(3, "member-1")

	p.Advance(3, "member-1")
	second := p.Begin(3, "member-1")
	p.Advance(3, "member-1")

	if first.Publish(nil) {
		t.Error("an attempt from two refreshes ago published")
	}
	if second.Publish(nil) {
		t.Error("an attempt from one refresh ago published")
	}
	if !p.Begin(3, "member-1").Publish(nil) {
		t.Error("a fresh attempt must publish")
	}
}

// A zero Attempt was never obtained from Begin, so it has no generation to
// claim and must fail closed rather than install state.
func TestZeroAttemptIsInert(t *testing.T) {
	var zero Attempt

	installed := false
	if zero.Publish(func() { installed = true }) {
		t.Error("the zero Attempt published")
	}
	if installed {
		t.Error("the zero Attempt ran commit")
	}
	if zero.Current() {
		t.Error("the zero Attempt reported itself current")
	}
}

func TestCurrent_TracksTheMembershipsOwnGeneration(t *testing.T) {
	p := New(nil)
	attempt := p.Begin(3, "member-1")

	if !attempt.Current() {
		t.Error("a fresh attempt is not current")
	}

	p.Advance(3, "member-2") // somebody else's refresh
	if !attempt.Current() {
		t.Error("another membership's refresh retired this attempt")
	}

	p.Advance(3, "member-1")
	if attempt.Current() {
		t.Error("the attempt survived its own membership's refresh")
	}
}

// Begin, Advance, and Publish are serialized against each other; this is here
// so -race has something to say if that ever stops being true.
func TestConcurrentUseIsSerialized(t *testing.T) {
	p := New(func(int, string) {})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); p.Begin(3, "member-1").Publish(func() {}) }()
		go func() { defer wg.Done(); p.Advance(3, "member-1") }()
		go func() { defer wg.Done(); p.Begin(3, "member-2").Current() }()
	}
	wg.Wait()
}
