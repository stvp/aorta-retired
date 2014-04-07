package cache

import (
	"container/list"
	"github.com/stvp/resp"
	"sync"
	"time"
)

// A Cache is a simple cache for RESP objects with string keys. The primary
// goal is to ensure that the cache fill function for a given key is never
// called more often than needed. If you continually call Fetch() with a max
// age of 1 second ago, the cache fill function will never be called more than
// once a second regardless of whether the cache fill function is fast or slow
// for a given key. If a cache fill for a key is slow, simultaneous Fetch()
// calls for that key will block until the cache is filled.
//
// Cache is designed to hold up to multiple millions of keys. Memory efficiency
// is not a goal, but the overhead for a million keys shouldn't be more than
// 16-32 megabytes.
type Cache struct {
	Hits   int
	Misses int

	l            list.List
	m            map[string]*list.Element
	mutexes      map[string]*sync.Mutex
	mutexesMutex sync.Mutex
}

type cachedObject struct {
	key       string
	object    resp.Object
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
func (c *Cache) Fetch(key string, maxAge time.Time, fn func() (resp.Object, error)) (resp.Object, error) {
	c.lockKey(key)
	defer c.unlockKey(key)

	// Try to use cached value
	element, ok := c.m[key]
	if ok {
		obj := element.Value.(*cachedObject)
		if obj.timestamp.After(maxAge) {
			c.Hits++
			return obj.object, nil
		}
	}

	c.Misses++

	// Cache is empty or stale, fill it up
	object, err := fn()
	if err != nil {
		return object, err
	}

	value := &cachedObject{
		key:       key,
		object:    object,
		timestamp: time.Now(),
	}
	c.m[key] = c.l.PushFront(value)

	return object, nil
}

// Expire is an exact expiration loop that expires all keys (up to a given
// maximum count) that are older than the given time.Time. It expires the
// oldest values first, and returns the number of values that were expired.
// It's moderately fast: on a MacBook Pro, it expires ~2,000 items per
// millisecond.
func (c *Cache) Expire(maxCount int, maxAge time.Time) (expired int) {
	var v *cachedObject
	var cursor, prev *list.Element
	cursor = c.l.Back()

	for cursor != nil {
		v = cursor.Value.(*cachedObject)
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

// Len returns the number of keys in the cache.
func (c *Cache) Len() (count int) {
	return len(c.m)
}

func (c *Cache) remove(e *list.Element) {
	value := e.Value.(*cachedObject)
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
