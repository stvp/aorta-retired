package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCacheFetch(t *testing.T) {
	now := time.Now()
	secondAgo := now.Add(-time.Second)

	cache := NewCache()

	// Fetch an un-cached value
	result, err := cache.Fetch("mykey", secondAgo, func() ([]byte, error) {
		return []byte("cool"), nil
	})
	if err != nil {
		t.Error(err)
	}
	if string(result) != "cool" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}

	// Fetch a cached value
	result, err = cache.Fetch("mykey", secondAgo, func() ([]byte, error) {
		t.Error("Fetch called the CacheFill function when the key was already cached")
		return []byte("nope"), nil
	})
	if err != nil {
		t.Error(err)
	}
	if string(result) != "cool" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}

	// Fetch a cached but stale value
	result, err = cache.Fetch("mykey", time.Now(), func() ([]byte, error) {
		return []byte("even cooler"), nil
	})
	if err != nil {
		t.Error(err)
	}
	if string(result) != "even cooler" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}

	// Fetch when the CacheFill returns an error
	result, err = cache.Fetch("uncachedkey", now, func() ([]byte, error) {
		return nil, fmt.Errorf("oh no")
	})
	if err == nil {
		t.Error("Fetch() with a failed CacheFill should return an error, but it didn't")
	}

	// Fetch after a failure runs the given function to fill cache
	result, err = cache.Fetch("uncachedkey", secondAgo, func() ([]byte, error) {
		return []byte("normal"), nil
	})
	if err != nil {
		t.Error(err)
	}
	if string(result) != "normal" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}
}

func BenchmarkFetchFillRate(b *testing.B) {
	cache := NewCache()
	var fills int
	wg := sync.WaitGroup{}
	wg.Add(2)
	start := time.Now()
	go func() {
		for i := 0; i < b.N; i++ {
			cache.Fetch("mykey", time.Now().Add(-time.Millisecond), func() ([]byte, error) {
				fills++
				return []byte{}, nil
			})
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < b.N; i++ {
			cache.Fetch("mykey", time.Now().Add(-time.Millisecond), func() ([]byte, error) {
				fills++
				return []byte{}, nil
			})
		}
		wg.Done()
	}()
	wg.Wait()
	stop := time.Now()
	duration := stop.Sub(start)

	fmt.Printf("\n%d fills in %d ms (expect 1 per ms)\t", fills, duration/time.Millisecond)
}

func TestExpire(t *testing.T) {
	cache := NewCache()

	// With no elements
	expired := cache.Expire(-1, time.Now())
	if expired != 0 {
		t.Errorf("expected to expire 0 elements, got: %d", expired)
	}

	// a is oldest element
	for _, letter := range []string{"a", "b", "c", "d", "e", "f"} {
		cache.Fetch(letter, time.Now(), func() ([]byte, error) { return []byte{}, nil })
	}
	// a, b, and c should get expired
	for _, letter := range []string{"a", "b", "c"} {
		v := cache.m[letter].Value.(*cachedValue)
		v.timestamp = time.Now().Add(-time.Hour)
	}

	// Expire 2 keys
	expired = cache.Expire(2, time.Now().Add(-time.Minute))
	if expired != 2 {
		t.Errorf("expected to expire 2 elements, got: %d", expired)
	}
	if _, ok := cache.m["b"]; ok {
		t.Error("didn't remove b, which was old")
	}
	if _, ok := cache.m["c"]; !ok {
		t.Error("shouldn't remove more than 2 keys")
	}

	// Expire everything
	expired = cache.Expire(-1, time.Now())
	if expired != 4 {
		t.Errorf("expected to expire 4 elements, got: %d", expired)
	}
	if _, ok := cache.m["c"]; ok {
		t.Error("didn't remove c, which was old")
	}
}

func BenchmarkExpire(b *testing.B) {
	cache := NewCache()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("some_key_%d", i)
		cache.Fetch(key, time.Now(), func() ([]byte, error) {
			return []byte("some value here"), nil
		})
	}

	b.ResetTimer()
	cache.Expire(-1, time.Now())
}
