package cache

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

// LRUCache implements an LRU (Least Recently Used) cache with statistics
type LRUCache struct {
	mu         sync.RWMutex
	capacity   int
	ttl        time.Duration
	items      map[string]*lruItem
	lruList    *list.List
	
	// Statistics (using atomic for thread-safe counters)
	hits       atomic.Uint64
	misses     atomic.Uint64
	evictions  atomic.Uint64
	expirations atomic.Uint64
}

type lruItem struct {
	key        string
	value      interface{}
	expiration time.Time
	element    *list.Element // pointer to position in LRU list
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(capacity int, ttl time.Duration) *LRUCache {
	c := &LRUCache{
		capacity: capacity,
		ttl:      ttl,
		items:    make(map[string]*lruItem),
		lruList:  list.New(),
	}

	// Start cleanup goroutine
	go c.cleanupExpired()

	return c
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		c.misses.Add(1)
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.expiration) {
		c.remove(key)
		c.expirations.Add(1)
		c.misses.Add(1)
		return nil, false
	}

	// Move to front (most recently used)
	c.lruList.MoveToFront(item.element)
	c.hits.Add(1)

	return item.value, true
}

// Set adds or updates a value in the cache
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if item already exists
	if item, found := c.items[key]; found {
		// Update existing item
		item.value = value
		item.expiration = time.Now().Add(c.ttl)
		c.lruList.MoveToFront(item.element)
		return
	}

	// Check capacity and evict if necessary
	if c.lruList.Len() >= c.capacity {
		c.evictOldest()
	}

	// Add new item
	item := &lruItem{
		key:        key,
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}

	// Add to front of LRU list
	item.element = c.lruList.PushFront(key)
	c.items[key] = item
}

// Delete removes a value from the cache
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.remove(key)
}

// Clear removes all values from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*lruItem)
	c.lruList = list.New()
}

// Size returns the current number of items in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Stats returns cache statistics
func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return CacheStats{
		Hits:        hits,
		Misses:      misses,
		HitRate:     hitRate,
		Evictions:   c.evictions.Load(),
		Expirations: c.expirations.Load(),
		Size:        len(c.items),
		Capacity:    c.capacity,
	}
}

// ResetStats resets all statistics counters
func (c *LRUCache) ResetStats() {
	c.hits.Store(0)
	c.misses.Store(0)
	c.evictions.Store(0)
	c.expirations.Store(0)
}

// remove removes an item from the cache (caller must hold lock)
func (c *LRUCache) remove(key string) {
	if item, found := c.items[key]; found {
		c.lruList.Remove(item.element)
		delete(c.items, key)
	}
}

// evictOldest removes the least recently used item
func (c *LRUCache) evictOldest() {
	element := c.lruList.Back()
	if element != nil {
		key := element.Value.(string)
		c.remove(key)
		c.evictions.Add(1)
	}
}

// cleanupExpired periodically removes expired items
func (c *LRUCache) cleanupExpired() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		expiredKeys := make([]string, 0)
		
		for key, item := range c.items {
			if now.After(item.expiration) {
				expiredKeys = append(expiredKeys, key)
			}
		}

		for _, key := range expiredKeys {
			c.remove(key)
			c.expirations.Add(1)
		}
		c.mu.Unlock()
	}
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	Hits        uint64  `json:"hits"`
	Misses      uint64  `json:"misses"`
	HitRate     float64 `json:"hit_rate_percent"`
	Evictions   uint64  `json:"evictions"`
	Expirations uint64  `json:"expirations"`
	Size        int     `json:"current_size"`
	Capacity    int     `json:"capacity"`
}
