package cache

import (
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Cache defines the interface for caching operations.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

// MemoryCache implements Cache using in-memory storage.
type MemoryCache struct {
	cache *gocache.Cache
}

func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	return &MemoryCache{cache: gocache.New(defaultExpiration, cleanupInterval)}
}

func (c *MemoryCache) Get(key string) (interface{}, bool)              { return c.cache.Get(key) }
func (c *MemoryCache) Set(key string, v interface{}, ttl time.Duration) { c.cache.Set(key, v, ttl) }
func (c *MemoryCache) Delete(key string)                               { c.cache.Delete(key) }
func (c *MemoryCache) Clear()                                          { c.cache.Flush() }

// NoOpCache is a no-op implementation (for when caching is disabled).
type NoOpCache struct{}

func NewNoOpCache() *NoOpCache                                           { return &NoOpCache{} }
func (c *NoOpCache) Get(key string) (interface{}, bool)                 { return nil, false }
func (c *NoOpCache) Set(key string, value interface{}, ttl time.Duration) {}
func (c *NoOpCache) Delete(key string)                                  {}
func (c *NoOpCache) Clear()                                             {}
