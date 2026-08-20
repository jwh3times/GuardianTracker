package manifeststate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"guardian-tracker/api-service/cache"
	"guardian-tracker/api-service/observability"
)

// ErrNilPublication is returned when [Load] or [LoadIf] is called without a
// publication. Unlike a nil cache, which simply means "load every time", a nil
// publication would silently drop the fence and reintroduce stale publication —
// so it fails loudly instead.
var ErrNilPublication = errors.New("manifeststate: nil publication")

// Load returns the value cached under key, calling load on a miss and caching
// what it returns only if the manifest generation has not moved meanwhile.
//
// It is [cache.Load] plus the ADR 0014 fence. The rules that function
// established still hold — an error is never cached, and a cached value of the
// wrong type is a miss that says so — with one added: a value loaded under a
// retired generation is returned to its caller but not installed.
func Load[T any](ctx context.Context, p *Publication, c cache.Cache, key string, ttl time.Duration, load func() (T, error)) (T, error) {
	return LoadIf(ctx, p, c, key, ttl, load, nil)
}

// LoadIf is [Load] with a predicate deciding whether a freshly loaded value is
// worth caching, mirroring [cache.LoadIf]. A nil predicate caches everything
// load returns, subject to the generation still being current.
//
// The predicate and the fence answer different questions and both must pass:
// storeIf asks whether the value is worth keeping, the fence asks whether it is
// still about the current manifest.
//
// The sequence is deliberately not expressed as a [cache.LoadIf] call with a
// clever predicate. The generation check and the cache write have to happen
// together under the publication, and a predicate runs before the write with
// the lock released in between — which is the window this exists to close.
func LoadIf[T any](
	ctx context.Context,
	p *Publication,
	c cache.Cache,
	key string,
	ttl time.Duration,
	load func() (T, error),
	storeIf func(T) bool,
) (T, error) {
	var zero T
	if p == nil {
		return zero, ErrNilPublication
	}

	// Captured before the read, not just before the write: a hit served from an
	// entry that a concurrent Advance is about to clear is still coherent, but
	// anything loaded from here on belongs to this generation.
	attempt := p.Begin()

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
		attempt.Publish(func() { c.Set(key, value, ttl) })
	}

	// Returned whether or not it published. The caller asked for a coherent
	// result and got one; losing the publish race only means nobody else
	// inherits it.
	return value, nil
}
