package datastore

import (
	cache "github.com/patrickmn/go-cache"
	"time"
)

const (
	DefaultTTL = cache.DefaultExpiration
	NoTTL      = cache.NoExpiration
)

type Cacher interface {
	// Get retrieves a value by key.
	// Returns (nil, false) if not found.
	Get(key string) (any, bool)

	// Set stores a value with optional expiration.
	// If duration is 0, item never expires (if supported).
	Set(key string, value any, duration time.Duration)

	// Delete removes the key from the cache.
	Delete(key string)

	// Clear removes all entries (if
	// supported).
	Clear()
}

type GoCache struct {
	internal *cache.Cache
}

func NewGoCache(defaultTTL, cleanupInterval time.Duration) Cacher {
	return &GoCache{
		internal: cache.New(defaultTTL, cleanupInterval),
	}
}

func (c *GoCache) Get(key string) (any, bool) {
	return c.internal.Get(key)
}

func (c *GoCache) Set(key string, value any, duration time.Duration) {
	c.internal.Set(key, value, duration)
}

func (c *GoCache) Delete(key string) {
	c.internal.Delete(key)
}

func (c *GoCache) Clear() {
	c.internal.Flush()
}
