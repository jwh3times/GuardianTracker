package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

// countingLoader records how often it ran, which is how every test here tells a
// hit from a miss.
type countingLoader[T any] struct {
	value T
	err   error
	calls int
}

func (l *countingLoader[T]) load() (T, error) {
	l.calls++
	return l.value, l.err
}

func TestLoad_CachesAcrossCalls(t *testing.T) {
	c := NewMemoryCache(time.Minute, 0)
	loader := &countingLoader[[]string]{value: []string{"a", "b"}}

	for i := range 3 {
		got, err := Load(context.Background(), c, "k", time.Minute, loader.load)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if len(got) != 2 || got[0] != "a" {
			t.Fatalf("call %d returned %v", i, got)
		}
	}
	if loader.calls != 1 {
		t.Errorf("loader ran %d times, want 1 — later calls should have been served from the cache", loader.calls)
	}
}

func TestLoad_NeverCachesAnError(t *testing.T) {
	c := NewMemoryCache(time.Minute, 0)
	boom := errors.New("manifest not ready")
	loader := &countingLoader[[]string]{err: boom}

	for range 2 {
		if _, err := Load(context.Background(), c, "k", time.Minute, loader.load); !errors.Is(err, boom) {
			t.Fatalf("error = %v, want %v", err, boom)
		}
	}
	if loader.calls != 2 {
		t.Errorf("loader ran %d times, want 2 — a failure must not be served from the cache until the TTL expires", loader.calls)
	}

	// And the failure must not have displaced anything: a later success caches.
	loader.err = nil
	loader.value = []string{"recovered"}
	if _, err := Load(context.Background(), c, "k", time.Minute, loader.load); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(context.Background(), c, "k", time.Minute, loader.load); err != nil {
		t.Fatal(err)
	}
	if loader.calls != 3 {
		t.Errorf("loader ran %d times, want 3 — the recovered value should be cached", loader.calls)
	}
}

func TestLoad_ErrorReturnsTheZeroValue(t *testing.T) {
	c := NewMemoryCache(time.Minute, 0)
	loader := &countingLoader[map[string]string]{value: map[string]string{"leaked": "yes"}, err: errors.New("boom")}

	got, err := Load(context.Background(), c, "k", time.Minute, loader.load)
	if err == nil {
		t.Fatal("expected an error")
	}
	if got != nil {
		t.Errorf("value = %v alongside an error, want the zero value — a half-built result read as real data is the failure this package keeps hitting", got)
	}
}

func TestLoadIf_RejectedValueIsReturnedButNotCached(t *testing.T) {
	c := NewMemoryCache(time.Minute, 0)
	loader := &countingLoader[[]int]{value: nil} // a transient empty response
	nonEmpty := func(v []int) bool { return len(v) > 0 }

	got, err := LoadIf(context.Background(), c, "k", time.Minute, loader.load, nonEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("value = %v, want the loaded empty result returned to this caller", got)
	}

	// The next caller must reach the loader rather than inherit the empty result.
	loader.value = []int{1, 2, 3}
	got, err = LoadIf(context.Background(), c, "k", time.Minute, loader.load, nonEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("value = %v, want the reloaded non-empty result", got)
	}
	if loader.calls != 2 {
		t.Fatalf("loader ran %d times, want 2 — the empty result must not have been cached", loader.calls)
	}

	// The accepted value is cached, so a third call does not reload.
	if _, err := LoadIf(context.Background(), c, "k", time.Minute, loader.load, nonEmpty); err != nil {
		t.Fatal(err)
	}
	if loader.calls != 2 {
		t.Errorf("loader ran %d times, want 2 — the accepted value should have been cached", loader.calls)
	}
}

func TestLoad_WrongTypedEntryIsAMissAndIsReplaced(t *testing.T) {
	c := NewMemoryCache(time.Minute, 0)
	c.Set("k", "a string from an older build", time.Minute)
	loader := &countingLoader[[]int]{value: []int{7}}

	got, err := Load(context.Background(), c, "k", time.Minute, loader.load)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("value = %v, want the freshly loaded one", got)
	}

	// Replaced, not merely bypassed: leaving the wrong-typed entry in place
	// would make every later call reload for as long as the TTL held it.
	if _, err := Load(context.Background(), c, "k", time.Minute, loader.load); err != nil {
		t.Fatal(err)
	}
	if loader.calls != 1 {
		t.Errorf("loader ran %d times, want 1 — the wrong-typed entry should have been overwritten", loader.calls)
	}
}

func TestLoad_NilCacheAlwaysLoads(t *testing.T) {
	loader := &countingLoader[int]{value: 42}

	for range 2 {
		got, err := Load(context.Background(), nil, "k", time.Minute, loader.load)
		if err != nil {
			t.Fatal(err)
		}
		if got != 42 {
			t.Fatalf("value = %d, want 42", got)
		}
	}
	if loader.calls != 2 {
		t.Errorf("loader ran %d times, want 2", loader.calls)
	}
}

func TestLoad_NoOpCacheAlwaysLoads(t *testing.T) {
	loader := &countingLoader[int]{value: 1}
	for range 2 {
		if _, err := Load(context.Background(), NewNoOpCache(), "k", time.Minute, loader.load); err != nil {
			t.Fatal(err)
		}
	}
	if loader.calls != 2 {
		t.Errorf("loader ran %d times, want 2 — NoOpCache stores nothing", loader.calls)
	}
}

func TestLoad_HonoursTheTTL(t *testing.T) {
	c := NewMemoryCache(time.Minute, 0)
	loader := &countingLoader[int]{value: 1}

	if _, err := Load(context.Background(), c, "k", 20*time.Millisecond, loader.load); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	if _, err := Load(context.Background(), c, "k", 20*time.Millisecond, loader.load); err != nil {
		t.Fatal(err)
	}
	if loader.calls != 2 {
		t.Errorf("loader ran %d times, want 2 — the entry should have expired", loader.calls)
	}
}

// A nil map and a nil slice survive the round trip as themselves rather than
// coming back as a wrong-typed miss, which is what lets a caller cache a
// legitimately empty answer when it wants to.
func TestLoad_CachesTypedNilResults(t *testing.T) {
	c := NewMemoryCache(time.Minute, 0)
	loader := &countingLoader[[]string]{value: nil}

	for range 2 {
		got, err := Load(context.Background(), c, "k", time.Minute, loader.load)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Fatalf("value = %v, want nil", got)
		}
	}
	if loader.calls != 1 {
		t.Errorf("loader ran %d times, want 1 — a typed nil is a real cached answer", loader.calls)
	}
}

// The two shipped predicates, checked here rather than only through their
// callers: a NonEmpty that answered true for an empty value would silently
// disable the don't-cache-empty rule at every site that passes it.
func TestNonEmptyPredicates(t *testing.T) {
	if NonEmptyMap(map[string]int{}) || NonEmptyMap[string, int](nil) {
		t.Error("NonEmptyMap accepted an empty map")
	}
	if !NonEmptyMap(map[string]int{"a": 1}) {
		t.Error("NonEmptyMap rejected a populated map")
	}
	if NonEmptySlice([]int{}) || NonEmptySlice[int](nil) {
		t.Error("NonEmptySlice accepted an empty slice")
	}
	if !NonEmptySlice([]int{1}) {
		t.Error("NonEmptySlice rejected a populated slice")
	}
}
