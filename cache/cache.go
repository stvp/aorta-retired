package cache

import (
	"sync"
	"time"
)

type CacheFill func() ([]byte, error)

type TimeBytes struct {
	Bytes []byte
	Time  time.Time
}

type Cache struct {
	values  map[string]TimeBytes
	mutexes map[string]*sync.Mutex
	sync.Mutex
}

func NewCache() *Cache {
	return &Cache{
		values:  map[string]TimeBytes{},
		mutexes: map[string]*sync.Mutex{},
	}
}

func (c *Cache) lock(key string) {
	c.Lock()
	defer c.Unlock()

	mutex, ok := c.mutexes[key]
	if !ok {
		mutex = &sync.Mutex{}
		c.mutexes[key] = mutex
	}
	mutex.Lock()
}

func (c *Cache) unlock(key string) {
	mutex := c.mutexes[key]
	mutex.Unlock()
}

func (c *Cache) Fetch(key string, maxAge time.Duration, fill CacheFill) ([]byte, error) {
	c.lock(key)
	defer c.unlock(key)

	now := time.Now()

	value, ok := c.values[key]
	if ok && now.Sub(value.Time) < maxAge {
		return value.Bytes, nil
	}

	b, err := fill()
	if err != nil {
		return b, err
	}

	c.values[key] = TimeBytes{
		Bytes: b,
		Time:  now,
	}

	return b, nil
}

func (c *Cache) clear(key string) {
	c.lock(key)
	delete(c.values, key)
	delete(c.mutexes, key)
}
