package cache

import (
	"context"
	"fmt"
	"time"

	"guardian-tracker/api-service/observability"
)

// Load returns the value cached under key, calling load on a miss and caching
// what it returns.
//
// It exists because a dozen call sites wrote the same six lines — get, assert
// the type, fall through on either failure — and each then decided for itself
// what was worth storing. Two rules that had only ever been prose comments are
// now this function's behaviour:
//
//   - An error is never cached. The next caller retries instead of being served
//     a failure until the TTL runs out.
//   - A cached value whose type is not T is a miss, and says so. That case used
//     to be indistinguishable from a cold entry, so changing a cached type gave
//     a permanent 100% miss rate with nothing to notice it by.
//
// A nil Cache loads every time. Callers used to guard for that themselves, and
// the guards outlived the possibility — the running service always has a cache,
// NoOpCache when caching is disabled.
func Load[T any](ctx context.Context, c Cache, key string, ttl time.Duration, load func() (T, error)) (T, error) {
	return LoadIf(ctx, c, key, ttl, load, nil)
}

// LoadIf is Load with a predicate deciding whether a freshly loaded value is
// worth caching. A nil predicate caches everything load returns.
//
// The predicate is what "do not cache an empty result" looks like when it is
// enforced rather than described: a transient empty vendor response cached
// under a key that lives until the next daily reset would serve every user an
// empty week, and nothing but a comment stood between the two sites that knew
// this and the ten that did not.
func LoadIf[T any](ctx context.Context, c Cache, key string, ttl time.Duration, load func() (T, error), storeIf func(T) bool) (T, error) {
	var zero T
	if c != nil {
		if cached, ok := c.Get(key); ok {
			if typed, ok := cached.(T); ok {
				return typed, nil
			}
			// The key is deliberately absent: cache keys embed membership and
			// character ids, and application logs carry those only as
			// pseudonyms. The two types are what makes this actionable anyway.
			observability.Logger(ctx).WarnContext(ctx, "cached value has the wrong type; treating as a miss",
				"want_type", fmt.Sprintf("%T", zero), "cached_type", fmt.Sprintf("%T", cached))
		}
	}

	value, err := load()
	if err != nil {
		return zero, err
	}
	if c != nil && (storeIf == nil || storeIf(value)) {
		c.Set(key, value, ttl)
	}
	return value, nil
}

// NonEmptyMap and NonEmptySlice are the LoadIf predicate every current caller
// wants: an empty result from a manifest read or a vendor call is a failed read
// rather than an answer, and caching it serves every later request a blank
// result for the whole TTL. They live here so the rule has one spelling rather
// than one per service.
func NonEmptyMap[K comparable, V any](m map[K]V) bool { return len(m) > 0 }
func NonEmptySlice[T any](s []T) bool                 { return len(s) > 0 }
