package open5e

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type Cache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
	ttl     time.Duration
	clock   Clock
	stop    chan struct{}
	done    chan struct{}
}

func NewCache(ttl, reapEvery time.Duration) *Cache {
	return newCache(ttl, reapEvery, realClock{})
}

func newCache(ttl, reapEvery time.Duration, clock Clock) *Cache {
	c := &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		clock:   clock,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go c.reaper(reapEvery)
	return c
}

func (c *Cache) Close() {
	select {
	case <-c.stop:
		return
	default:
		close(c.stop)
		<-c.done
	}
}

func (c *Cache) reaper(every time.Duration) {
	defer close(c.done)
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.PurgeExpired()
		}
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if !e.expiresAt.After(c.clock.Now()) {
		delete(c.entries, key)
		return nil, false
	}
	out := make([]byte, len(e.data))
	copy(out, e.data)
	return out, true
}

func (c *Cache) Set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	c.entries[key] = cacheEntry{
		data:      cp,
		expiresAt: c.clock.Now().Add(c.ttl),
	}
}

func (c *Cache) PurgeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.clock.Now()
	for k, e := range c.entries {
		if !e.expiresAt.After(now) {
			delete(c.entries, k)
		}
	}
}

func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
