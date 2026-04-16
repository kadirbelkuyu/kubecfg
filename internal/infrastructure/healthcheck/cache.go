package healthcheck

import (
	"sync"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain/health"
)

type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	result    health.Result
	expiresAt time.Time
}

func NewCache() *Cache {
	return &Cache{entries: make(map[string]cacheEntry)}
}

func (c *Cache) Get(contextName string) (health.Result, bool) {
	c.mu.RLock()
	entry, ok := c.entries[contextName]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return health.Result{}, false
	}
	return entry.result, true
}

func (c *Cache) Set(result health.Result) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[result.ContextName] = cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(health.CacheExpiry),
	}
}

func (c *Cache) Invalidate(contextName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, contextName)
}

func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]cacheEntry)
}
