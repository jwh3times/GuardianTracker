package efficiency

import (
	"errors"
	"sync"
	"testing"
	"time"

	"guardian-tracker/api-service/services/bungie"
	"guardian-tracker/api-service/services/manifest"
)

type mutableVersion struct {
	mu sync.RWMutex
	v  string
}

type coalescingSource struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
}

func (s *coalescingSource) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()
	if call == 1 {
		close(s.started)
	}
	<-s.release
	return generationRows(1), nil
}

func (s *coalescingSource) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestBuildIndexCoalescesConcurrentSameGenerationCalls(t *testing.T) {
	source := &coalescingSource{started: make(chan struct{}), release: make(chan struct{})}
	engine := NewEngine(source, fakeVersion{v: "v1"})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(source.release) }) }
	t.Cleanup(release)

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		engine.BuildIndex()
	}()
	select {
	case <-source.started:
	case <-time.After(time.Second):
		release()
		<-firstDone
		t.Fatal("first build did not reach the source")
	}

	const concurrentCalls = 8
	peersDone := make(chan struct{}, concurrentCalls)
	for range concurrentCalls {
		go func() {
			engine.BuildIndex()
			peersDone <- struct{}{}
		}()
	}

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for received := 0; received < concurrentCalls; received++ {
		select {
		case <-peersDone:
		case <-timer.C:
			release()
			for remaining := received; remaining < concurrentCalls; remaining++ {
				<-peersDone
			}
			<-firstDone
			t.Fatal("same-generation BuildIndex call did not coalesce")
		}
	}
	if got := source.callCount(); got != 1 {
		release()
		<-firstDone
		t.Fatalf("source reads while first build blocked = %d, want 1", got)
	}

	release()
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("published build did not finish after release")
	}
	if got := source.callCount(); got != 1 {
		t.Fatalf("source reads = %d, want 1", got)
	}
}

type recoveringSource struct {
	mu          sync.Mutex
	calls       int
	healStarted chan struct{}
	releaseHeal chan struct{}
}

func (s *recoveringSource) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()
	switch call {
	case 1:
		return generationRows(1), nil
	case 2:
		return nil, errors.New("current generation lookup failed")
	case 3:
		close(s.healStarted)
		<-s.releaseHeal
		return generationRows(2), nil
	default:
		return nil, errors.New("unexpected extra source read")
	}
}

func (s *recoveringSource) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestCurrentGenerationFailureRetainsSnapshotAndRankSelfHeals(t *testing.T) {
	versions := &mutableVersion{v: "v1"}
	source := &recoveringSource{healStarted: make(chan struct{}), releaseHeal: make(chan struct{})}
	var releaseOnce sync.Once
	releaseHeal := func() { releaseOnce.Do(func() { close(source.releaseHeal) }) }
	t.Cleanup(releaseHeal)
	engine := NewEngine(source, versions)
	engine.BuildIndex()

	versions.set("v2")
	if err := engine.publication.Advance("v2"); err != nil {
		t.Fatalf("Advance(v2): %v", err)
	}
	engine.BuildIndex() // deterministic current-generation failure

	engine.mu.RLock()
	builtAfterFailure := engine.builtVersion
	readyAfterFailure := engine.ready
	engine.mu.RUnlock()
	if builtAfterFailure != "v1" || !readyAfterFailure {
		t.Fatalf("failed v2 build changed prior snapshot: version=%q ready=%v", builtAfterFailure, readyAfterFailure)
	}

	prior := engine.Rank(RankInput{MissingItemHashes: map[uint32]struct{}{101: {}}})
	if prior.State != RankReady || len(prior.Candidates) != 1 || prior.Candidates[0].SourceHash != 1 {
		t.Fatalf("Rank during self-heal did not serve v1 snapshot: %+v", prior)
	}
	select {
	case <-source.healStarted:
	case <-time.After(time.Second):
		t.Fatal("Rank did not re-kick the failed current-generation build")
	}

	releaseHeal()
	waitForBuiltVersion(t, engine, "v2")

	healed := engine.Rank(RankInput{MissingItemHashes: map[uint32]struct{}{102: {}}})
	if healed.State != RankReady || len(healed.Candidates) != 1 || healed.Candidates[0].SourceHash != 2 {
		t.Fatalf("Rank did not adopt healed v2 snapshot: %+v", healed)
	}
	if got := source.callCount(); got != 3 {
		t.Fatalf("source reads = %d, want initial + failed + healed", got)
	}
}

func generationRows(generation int) []manifest.CollectibleWithItem {
	return []manifest.CollectibleWithItem{{
		Collectible: bungie.CollectibleDefinition{
			Hash:         uint32(generation),
			ItemHash:     uint32(100 + generation),
			SourceHash:   uint32(generation),
			SourceString: "Vault of Glass raid",
		},
		Item: item(100+generation, 5),
	}}
}

func waitForBuiltVersion(t *testing.T, engine *Engine, version string) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		engine.mu.RLock()
		built := engine.builtVersion
		engine.mu.RUnlock()
		if built == version {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline.C:
			t.Fatalf("snapshot %q did not publish", version)
		}
	}
}

func (v *mutableVersion) Version() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.v
}

func (v *mutableVersion) set(value string) {
	v.mu.Lock()
	v.v = value
	v.mu.Unlock()
}

type generationSource struct {
	mu        sync.Mutex
	calls     int
	v2Started chan struct{}
	releaseV2 chan struct{}
	v3Built   chan struct{}
}

func (s *generationSource) GetAllCollectiblesWithItems() ([]manifest.CollectibleWithItem, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()
	if call == 2 {
		close(s.v2Started)
		<-s.releaseV2
	}
	if call == 3 {
		close(s.v3Built)
	}
	return []manifest.CollectibleWithItem{{
		Collectible: bungie.CollectibleDefinition{
			Hash:         uint32(call),
			ItemHash:     uint32(100 + call),
			SourceHash:   uint32(call),
			SourceString: "Vault of Glass raid",
		},
		Item: item(100+call, 5),
	}}, nil
}

func TestBuildIndexFencesOldGenerationAndAllowsNewBuild(t *testing.T) {
	versions := &mutableVersion{v: "v1"}
	source := &generationSource{
		v2Started: make(chan struct{}),
		releaseV2: make(chan struct{}),
		v3Built:   make(chan struct{}),
	}
	engine := NewEngine(source, versions)
	engine.BuildIndex()

	versions.set("v2")
	if err := engine.OnVersionChanged("v2"); err != nil {
		t.Fatalf("OnVersionChanged(v2): %v", err)
	}
	select {
	case <-source.v2Started:
	case <-time.After(time.Second):
		t.Fatal("v2 build did not start")
	}

	old := engine.Rank(RankInput{MissingItemHashes: map[uint32]struct{}{101: {}}})
	if old.State != RankReady || len(old.Candidates) != 1 || old.Candidates[0].SourceHash != 1 {
		t.Fatalf("replacement build did not retain v1 snapshot: %+v", old)
	}

	versions.set("v3")
	if err := engine.OnVersionChanged("v3"); err != nil {
		t.Fatalf("OnVersionChanged(v3): %v", err)
	}
	select {
	case <-source.v3Built:
	case <-time.After(time.Second):
		t.Fatal("v3 build was coalesced behind obsolete v2 work")
	}

	deadline := time.Now().Add(time.Second)
	for {
		engine.mu.RLock()
		built := engine.builtVersion
		engine.mu.RUnlock()
		if built == "v3" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("v3 snapshot did not publish")
		}
		time.Sleep(time.Millisecond)
	}

	close(source.releaseV2)
	deadline = time.Now().Add(time.Second)
	for {
		source.mu.Lock()
		calls := source.calls
		source.mu.Unlock()
		engine.mu.RLock()
		building := len(engine.building)
		engine.mu.RUnlock()
		if calls >= 3 && building == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("obsolete v2 build did not finish")
		}
		time.Sleep(time.Millisecond)
	}

	got := engine.Rank(RankInput{MissingItemHashes: map[uint32]struct{}{103: {}}})
	if got.State != RankReady || len(got.Candidates) != 1 || got.Candidates[0].SourceHash != 3 {
		t.Fatalf("obsolete v2 work replaced v3 snapshot: %+v", got)
	}
}
