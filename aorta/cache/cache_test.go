package cache

import (
	"fmt"
	"github.com/stvp/resp"
	"sync"
	"testing"
	"time"
)

func TestCacheFetch(t *testing.T) {
	now := time.Now()
	secondAgo := now.Add(-time.Second)

	cache := NewCache()

	// Fetch an un-cached value
	obj, err := cache.Fetch("mykey", secondAgo, func() (resp.Object, error) {
		return resp.NewBulkString("cool"), nil
	})
	if err != nil {
		t.Error(err)
	}
	if obj.(resp.String).String() != "cool" {
		t.Errorf("Fetch() returned the wrong object: %#v", obj)
	}

	// Fetch a cached value
	obj, err = cache.Fetch("mykey", secondAgo, func() (resp.Object, error) {
		t.Error("Fetch called the fill function when the key was already cached")
		return resp.NewBulkString("nope"), nil
	})
	if err != nil {
		t.Error(err)
	}
	if obj.(resp.String).String() != "cool" {
		t.Errorf("Fetch() returned the wrong object: %#v", obj)
	}

	// Fetch a cached but stale value
	obj, err = cache.Fetch("mykey", time.Now(), func() (resp.Object, error) {
		return resp.NewBulkString("even cooler"), nil
	})
	if err != nil {
		t.Error(err)
	}
	if obj.(resp.String).String() != "even cooler" {
		t.Errorf("Fetch() returned the wrong object: %#v", obj)
	}

	// Fetch when the CacheFill returns an error
	obj, err = cache.Fetch("uncachedkey", now, func() (resp.Object, error) {
		return nil, fmt.Errorf("oh no")
	})
	if err == nil {
		t.Error("Fetch() with a failed fill should return an error, but it didn't")
	}

	// Fetch after a failure runs the given function to fill cache
	obj, err = cache.Fetch("uncachedkey", secondAgo, func() (resp.Object, error) {
		return resp.NewBulkString("normal"), nil
	})
	if err != nil {
		t.Error(err)
	}
	if obj.(resp.String).String() != "normal" {
		t.Errorf("Fetch() returned the wrong object: %#v", obj)
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
			cache.Fetch("mykey", time.Now().Add(-time.Millisecond), func() (resp.Object, error) {
				fills++
				return resp.String{}, nil
			})
		}
		wg.Done()
	}()
	go func() {
		for i := 0; i < b.N; i++ {
			cache.Fetch("mykey", time.Now().Add(-time.Millisecond), func() (resp.Object, error) {
				fills++
				return resp.String{}, nil
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
		cache.Fetch(letter, time.Now(), func() (resp.Object, error) { return resp.String{}, nil })
	}
	// a, b, and c should get expired
	for _, letter := range []string{"a", "b", "c"} {
		v := cache.m[letter].Value.(*cachedObject)
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
		cache.Fetch(key, time.Now(), func() (resp.Object, error) {
			return resp.NewBulkString("some value here"), nil
		})
	}

	b.ResetTimer()
	cache.Expire(-1, time.Now())
}
