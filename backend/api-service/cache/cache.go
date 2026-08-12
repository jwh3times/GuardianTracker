package cache

import (
	"sync"
	"time"
)

// Cache defines the interface for caching operations.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

type cacheClock interface {
	Now() time.Time
	NewTicker(interval time.Duration) cacheTicker
}

type cacheTicker interface {
	C() <-chan time.Time
	Stop()
}

type realCacheClock struct{}

func (realCacheClock) Now() time.Time { return time.Now() }
func (realCacheClock) NewTicker(interval time.Duration) cacheTicker {
	return realCacheTicker{Ticker: time.NewTicker(interval)}
}

type realCacheTicker struct {
	*time.Ticker
}

func (t realCacheTicker) C() <-chan time.Time { return t.Ticker.C }

// MemoryCache implements Cache using unbounded in-memory storage.
type MemoryCache struct {
	mu                sync.RWMutex
	items             map[string]cacheItem
	defaultExpiration time.Duration
	clock             cacheClock
	stopCleanup       chan struct{}
	cleanupStopped    chan struct{}
	closeOnce         sync.Once
}

func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	return newMemoryCacheWithClock(defaultExpiration, cleanupInterval, realCacheClock{}, nil)
}

func newMemoryCacheWithClock(
	defaultExpiration time.Duration,
	cleanupInterval time.Duration,
	clock cacheClock,
	afterCleanup func(),
) *MemoryCache {
	c := &MemoryCache{
		items:             make(map[string]cacheItem),
		defaultExpiration: defaultExpiration,
		clock:             clock,
	}
	if cleanupInterval > 0 {
		c.stopCleanup = make(chan struct{})
		c.cleanupStopped = make(chan struct{})
		go c.runCleanup(cleanupInterval, afterCleanup)
	}
	return c
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	if !ok {
		c.mu.RUnlock()
		return nil, false
	}
	if item.expiresAt.IsZero() || c.clock.Now().Before(item.expiresAt) {
		c.mu.RUnlock()
		return item.value, true
	}
	c.mu.RUnlock()

	// Cleanup may be disabled, so Get must enforce expiry itself. Recheck under
	// the write lock so a concurrent Set cannot be removed after an overwrite.
	c.mu.Lock()
	item, ok = c.items[key]
	if ok && !item.expiresAt.IsZero() && !c.clock.Now().Before(item.expiresAt) {
		delete(c.items, key)
	}
	c.mu.Unlock()
	return nil, false
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	if ttl == 0 {
		ttl = c.defaultExpiration
	}

	item := cacheItem{value: value}
	if ttl > 0 {
		item.expiresAt = c.clock.Now().Add(ttl)
	}

	c.mu.Lock()
	c.items[key] = item
	c.mu.Unlock()
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *MemoryCache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]cacheItem)
	c.mu.Unlock()
}

// Close stops the cleanup worker, if one was started, and waits for it to exit.
// It is safe to call Close more than once.
func (c *MemoryCache) Close() {
	c.closeOnce.Do(func() {
		if c.stopCleanup == nil {
			return
		}
		close(c.stopCleanup)
		<-c.cleanupStopped
	})
}

func (c *MemoryCache) runCleanup(interval time.Duration, afterCleanup func()) {
	ticker := c.clock.NewTicker(interval)
	defer close(c.cleanupStopped)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C():
			c.deleteExpired()
			if afterCleanup != nil {
				afterCleanup()
			}
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *MemoryCache) deleteExpired() {
	now := c.clock.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, item := range c.items {
		if !item.expiresAt.IsZero() && !now.Before(item.expiresAt) {
			delete(c.items, key)
		}
	}
}

// NoOpCache is a no-op implementation (for when caching is disabled).
type NoOpCache struct{}

func NewNoOpCache() *NoOpCache                                            { return &NoOpCache{} }
func (c *NoOpCache) Get(key string) (interface{}, bool)                   { return nil, false }
func (c *NoOpCache) Set(key string, value interface{}, ttl time.Duration) {}
func (c *NoOpCache) Delete(key string)                                    {}
func (c *NoOpCache) Clear()                                               {}
