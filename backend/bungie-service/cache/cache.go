package cache

import (
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Cache defines the interface for caching operations
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

// MemoryCache implements Cache using in-memory storage
type MemoryCache struct {
	cache *gocache.Cache
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	return &MemoryCache{
		cache: gocache.New(defaultExpiration, cleanupInterval),
	}
}

// Get retrieves a value from the cache
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

// Set stores a value in the cache with the specified TTL
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.cache.Set(key, value, ttl)
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(key string) {
	c.cache.Delete(key)
}

// Clear removes all items from the cache
func (c *MemoryCache) Clear() {
	c.cache.Flush()
}

// NoOpCache is a cache that does nothing (for when caching is disabled)
type NoOpCache struct{}

// NewNoOpCache creates a no-op cache
func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

// Get always returns not found
func (c *NoOpCache) Get(key string) (interface{}, bool) {
	return nil, false
}

// Set does nothing
func (c *NoOpCache) Set(key string, value interface{}, ttl time.Duration) {}

// Delete does nothing
func (c *NoOpCache) Delete(key string) {}

// Clear does nothing
func (c *NoOpCache) Clear() {}
