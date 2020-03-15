package cache

import (
	"runtime"
	"sync"
	"time"
)

type Item struct {
	value      interface{}
	expiration int64
}

type Cache struct {
	items             sync.Map
	defaultExpiration time.Duration
	gcInterval        time.Duration
	stop              chan struct{}
}

// Delete all expired items from the cache.
func (c *Cache) DeleteExpired() {
	now := time.Now().UnixNano()

	c.items.Range(func(key, value interface{}) bool {
		if value := value.(*Item); value.expiration > 0 && now > value.expiration {
			c.items.Delete(key)
		}
		return true
	})
}

func (c *Cache) gcLoop() {
	ticker := time.NewTicker(c.gcInterval)
	for {
		select {
		case <-ticker.C:
			c.DeleteExpired()
		case <-c.stop:
			ticker.Stop()
			return
		}
	}
}

func New(defaultExpiration, gcInterval time.Duration) *Cache {
	c := &Cache{
		defaultExpiration: defaultExpiration,
		gcInterval:        gcInterval,
		stop:              make(chan struct{}),
	}

	go c.gcLoop()

	runtime.SetFinalizer(c, func(c *Cache) {
		c.stop <- struct{}{}
	})
	return c
}

func (c *Cache) Set(key, value interface{}, d time.Duration) {
	var e int64

	if d == 0 {
		d = c.defaultExpiration
	}

	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}

	c.items.Store(key, &Item{value, e})
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	return c.items.Load(key)
}

func (c *Cache) Del(key interface{}) {
	c.items.Delete(key)
}

func (c *Cache) Have(key string) bool {
	_, ok := c.items.Load(key)
	return ok
}
