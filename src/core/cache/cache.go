/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/zefronxd/TGMUSIC
 */

package cache

import (
	"container/list"
	"sync"
	"time"
)

// Item holds a cached value and its expiration time.
type Item[T any] struct {
	Value      T
	Expiration time.Time
}

// entry is the payload stored in the LRU list; key is duplicated here so an
// evicted list element can remove itself from the lookup map.
type entry[T any] struct {
	key  string
	item Item[T]
}

// Cache is a generic, thread-safe TTL cache with automatic background
// eviction of expired entries and, when maxEntries > 0, bounded size via
// least-recently-used eviction. Bounded caches are what keep large or
// numerous payloads (thumbnails, album art, downloaded files, search
// results) from growing RAM usage without limit.
type Cache[T any] struct {
	mu         sync.RWMutex
	data       map[string]*list.Element // key -> *entry[T] node in order
	order      *list.List               // front = most recently used
	ttl        time.Duration
	maxEntries int
}

// NewCache creates a Cache with the given default TTL and no size limit
// (only TTL-based expiration). Use NewBoundedCache when the cached values
// can be large or numerous enough to threaten memory usage.
func NewCache[T any](ttl time.Duration) *Cache[T] {
	return newTTLCache[T](ttl, 0)
}

// NewBoundedCache creates a Cache with the given default TTL and a hard cap
// on the number of entries. Once the cap is reached, the least-recently-used
// entry is evicted before inserting a new one - this bounds worst-case RAM
// usage independent of how many distinct keys (tracks, queries, chats) are
// ever seen.
func NewBoundedCache[T any](ttl time.Duration, maxEntries int) *Cache[T] {
	return newTTLCache[T](ttl, maxEntries)
}

func newTTLCache[T any](ttl time.Duration, maxEntries int) *Cache[T] {
	c := &Cache[T]{
		data:       make(map[string]*list.Element),
		order:      list.New(),
		ttl:        ttl,
		maxEntries: maxEntries,
	}
	registerCache(c)
	return c
}

// Get returns the value for key and true if it exists and has not expired.
// A successful lookup marks the key as most-recently-used.
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.data[key]
	if !ok {
		var zero T
		return zero, false
	}

	e := el.Value.(*entry[T])
	if time.Now().After(e.item.Expiration) {
		c.removeElement(el)
		var zero T
		return zero, false
	}

	c.order.MoveToFront(el)
	return e.item.Value, true
}

// Set stores value under key using the default TTL.
func (c *Cache[T]) Set(key string, value T) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores value under key with a custom TTL. If the cache is
// bounded and already at capacity, the least-recently-used entry is evicted
// first.
func (c *Cache[T]) SetWithTTL(key string, value T, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item := Item[T]{Value: value, Expiration: time.Now().Add(ttl)}

	if el, ok := c.data[key]; ok {
		el.Value.(*entry[T]).item = item
		c.order.MoveToFront(el)
		return
	}

	if c.maxEntries > 0 && len(c.data) >= c.maxEntries {
		c.evictOldest()
	}

	el := c.order.PushFront(&entry[T]{key: key, item: item})
	c.data[key] = el
}

// Delete removes key from the cache immediately.
func (c *Cache[T]) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.data[key]; ok {
		c.removeElement(el)
	}
}

// Clear evicts all items at once.
func (c *Cache[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*list.Element)
	c.order.Init()
}

// Size returns the number of entries currently in the cache (including not-yet-evicted expired ones).
func (c *Cache[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Close unregisters the cache from the background cleanup janitor.
func (c *Cache[T]) Close() {
	unregisterCache(c)
}

// removeElement deletes a list element and its map entry. Caller must hold c.mu.
func (c *Cache[T]) removeElement(el *list.Element) {
	c.order.Remove(el)
	delete(c.data, el.Value.(*entry[T]).key)
}

// evictOldest drops the least-recently-used entry. Caller must hold c.mu.
func (c *Cache[T]) evictOldest() {
	oldest := c.order.Back()
	if oldest != nil {
		c.removeElement(oldest)
	}
}

// evictExpired removes all entries whose TTL has elapsed.
func (c *Cache[T]) evictExpired() {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for el := c.order.Back(); el != nil; {
		prev := el.Prev()
		if now.After(el.Value.(*entry[T]).item.Expiration) {
			c.removeElement(el)
		}
		el = prev
	}
}

// cleaner is an interface for caches that can evict expired items.
type cleaner interface {
	evictExpired()
}

// janitor manages a single goroutine that cleans up multiple caches.
type janitor struct {
	mu       sync.Mutex
	caches   []cleaner
	interval time.Duration
	stop     chan struct{}
	running  bool
}

var (
	sharedJanitor   *janitor
	janitorOnce     sync.Once
	janitorInterval = time.Minute
)

func getJanitor() *janitor {
	janitorOnce.Do(func() {
		sharedJanitor = &janitor{
			interval: janitorInterval,
		}
	})
	return sharedJanitor
}

func (j *janitor) runWith(interval time.Duration, stop chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			j.mu.Lock()
			if len(j.caches) == 0 {
				j.mu.Unlock()
				continue
			}
			caches := make([]cleaner, len(j.caches))
			copy(caches, j.caches)
			j.mu.Unlock()

			for _, c := range caches {
				c.evictExpired()
			}
		case <-stop:
			return
		}
	}
}

func registerCache(c cleaner) {
	getJanitor().register(c)
}

func (j *janitor) register(c cleaner) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.caches = append(j.caches, c)
	if !j.running {
		j.interval = janitorInterval
		j.stop = make(chan struct{})
		j.running = true
		go j.runWith(j.interval, j.stop)
	}
}

func unregisterCache(c cleaner) {
	getJanitor().unregister(c)
}

func (j *janitor) unregister(c cleaner) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i, v := range j.caches {
		if v == c {
			j.caches = append(j.caches[:i], j.caches[i+1:]...)
			break
		}
	}
	if len(j.caches) == 0 && j.running {
		close(j.stop)
		j.running = false
	}
}

func (j *janitor) has(c cleaner) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, v := range j.caches {
		if v == c {
			return true
		}
	}
	return false
}

func (j *janitor) count() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return len(j.caches)
}
