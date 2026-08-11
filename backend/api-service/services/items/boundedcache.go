package items

import "sync"

// boundedCache memoizes one manifest projection keyed by item hash.
//
// It exists because GetWeaponPerks, GetItem and GetCatalysts were three copies
// of the same twenty-five lines — read lock, miss, load, write lock, evict at
// capacity, store — differing only in the map they read and the repository call
// they made. The one place they genuinely differed was invisible: two of them
// cache a nil result and one deliberately does not, which was decided by where
// an early return happened to sit. That decision is now the storeIf argument,
// stated at each call site.
//
// Unlike cache.Cache this has no TTL. Entries are valid for exactly one manifest
// version and are dropped wholesale by InvalidateCache on a swap, so time is not
// what makes them stale.
type boundedCache[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]V
	max     int
}

func newBoundedCache[K comparable, V any](max int) *boundedCache[K, V] {
	return &boundedCache[K, V]{entries: map[K]V{}, max: max}
}

// load returns the cached value for key, calling load on a miss. An error is
// never cached. storeIf decides whether a freshly loaded value is worth keeping;
// a nil storeIf keeps everything, including a zero value.
func (c *boundedCache[K, V]) load(key K, load func(K) (V, error), storeIf func(V) bool) (V, error) {
	c.mu.RLock()
	v, ok := c.entries[key]
	c.mu.RUnlock()
	if ok {
		return v, nil
	}

	v, err := load(key)
	if err != nil {
		var zero V
		return zero, err
	}
	if storeIf != nil && !storeIf(v) {
		return v, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.max {
		// Bound memory: the key is a client-supplied item hash, so an
		// authenticated caller hitting many distinct hashes could otherwise grow
		// this without limit between manifest swaps. Evict an arbitrary entry at
		// capacity; legitimate traffic (a few thousand real weapons) never
		// reaches the cap.
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
	c.entries[key] = v
	return v, nil
}

// size reports how many entries are held. Only the cap tests read it.
func (c *boundedCache[K, V]) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// clear drops every entry, which is what a manifest swap does to all of them.
func (c *boundedCache[K, V]) clear() {
	c.mu.Lock()
	c.entries = map[K]V{}
	c.mu.Unlock()
}
