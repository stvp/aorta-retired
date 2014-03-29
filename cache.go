package main

import (
	"sync"
	"time"
)

type CacheFill func() (string, error)

type TimeString struct {
	String string
	Time   time.Time
}

type Cache struct {
	values  map[string]TimeString
	mutexes map[string]*sync.Mutex
	sync.Mutex
}

func NewCache() *Cache {
	return &Cache{
		values:  map[string]TimeString{},
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

func (c *Cache) Fetch(key string, maxAge time.Duration, fill CacheFill) (string, error) {
	c.lock(key)
	defer c.unlock(key)

	now := time.Now()

	value, ok := c.values[key]
	if ok && now.Sub(value.Time) < maxAge {
		return value.String, nil
	}

	str, err := fill()
	if err != nil {
		return "", err
	}

	c.values[key] = TimeString{
		String: str,
		Time:   now,
	}

	return str, nil
}

func (c *Cache) Clear(key string) {
	c.lock(key)
	delete(c.values, key)
	delete(c.mutexes, key)
}
