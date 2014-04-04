package cache

import (
	"container/list"
	"sync"
	"time"
)

// A Cache is a simple cache for byte slices with string keys. The primary goal
// is to ensure that the cache fill function for a given key is never called
// more often than needed. If you continually call Fetch() with a max age of 1
// second ago, the cache fill function will never be called more than once a
// second regardless of whether the cache fill function is fast or slow for a
// given key. If a cache fill for a key is slow, Fetch() calls for that key
// will block until the cache is filled.
//
// Cache is designed to hold at most millions of keys. Memory efficiency is not
// a design goal, but the overhead for a million keys shouldn't be more than
// 16-32 megabytes.
type Cache struct {
	l            list.List
	m            map[string]*list.Element
	mutexes      map[string]*sync.Mutex
	mutexesMutex sync.Mutex
}

type cachedValue struct {
	key       string
	bytes     []byte
	timestamp time.Time
}

// NewCache returns an initialized Cache, ready for use.
func NewCache() *Cache {
	return &Cache{
		m:       make(map[string]*list.Element),
		mutexes: make(map[string]*sync.Mutex),
	}
}

// Fetch takes a key and returns the cached value, if the key is cached and is
// not older than the given time.Time. If the key is not cached, the given
// cache fill function will be called to fill the cache. If the cache fill
// function returns an error, the cache will not be filled and the error will
// be returned by Fetch.
func (c *Cache) Fetch(key string, maxAge time.Time, fn func() ([]byte, error)) ([]byte, error) {
	c.lockKey(key)
	defer c.unlockKey(key)

	// Try to use cached value
	element, ok := c.m[key]
	if ok {
		value := element.Value.(*cachedValue)
		if value.timestamp.After(maxAge) {
			return value.bytes, nil
		}
	}

	// Cache is empty or stale, fill it up
	bytes, err := fn()
	if err != nil {
		return bytes, err
	}

	value := &cachedValue{
		key:       key,
		bytes:     bytes,
		timestamp: time.Now(),
	}
	c.m[key] = c.l.PushFront(value)

	return bytes, nil
}

// Expire is an exact expiration loop that expires all keys (up to a given
// maximum count) that are older than the given time.Time. It expires the
// oldest values first, and returns the number of values that were expired.
// It's moderately fast: on a MacBook Pro, it expires ~2,000 items per
// millisecond.
func (c *Cache) Expire(maxCount int, maxAge time.Time) (expired int) {
	var v *cachedValue
	var cursor, prev *list.Element
	cursor = c.l.Back()

	for cursor != nil {
		v = cursor.Value.(*cachedValue)
		if v.timestamp.After(maxAge) {
			break
		}
		prev = cursor.Prev()
		c.remove(cursor)
		cursor = prev
		expired++
		if maxCount > 0 && expired == maxCount {
			break
		}
	}

	return expired
}

func (c *Cache) remove(e *list.Element) {
	value := e.Value.(*cachedValue)
	c.lockKey(value.key)
	c.l.Remove(e)
	delete(c.m, value.key)
	c.mutexesMutex.Lock()
	delete(c.mutexes, value.key)
	c.mutexesMutex.Unlock()
}

func (c *Cache) lockKey(key string) {
	c.mutexesMutex.Lock()
	mutex, ok := c.mutexes[key]
	if !ok {
		mutex = &sync.Mutex{}
		c.mutexes[key] = mutex
	}
	c.mutexesMutex.Unlock()
	mutex.Lock()
}

func (c *Cache) unlockKey(key string) {
	c.mutexes[key].Unlock()
}
