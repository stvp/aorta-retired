package cache

import (
	"fmt"
	"testing"
	"time"
)

func TestCacheFetch(t *testing.T) {
	cache := NewCache()

	// Fetch an un-cached value
	result, err := cache.Fetch("mykey", time.Second, func() ([]byte, error) {
		return []byte("cool"), nil
	})
	if err != nil {
		t.Error("An uncached Fetch() call returned an error when it shouldn't have")
	}
	if string(result) != "cool" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}

	// Fetch a cached value
	cache.Fetch("mykey", time.Second, func() ([]byte, error) {
		t.Error("Fetch called the CacheFill function when the key was already cached")
		return []byte("nope"), nil
	})

	// Fetch a cached but stale value
	cache.values["mykey"] = TimeBytes{
		Bytes: []byte("old"),
		Time:  time.Now().Add(-time.Minute),
	}
	result, err = cache.Fetch("mykey", 30*time.Second, func() ([]byte, error) {
		return []byte("even cooler"), nil
	})
	if err != nil {
		t.Error("A stale, cached Fetch() call returned an error when it shouldn't have")
	}
	if string(result) != "even cooler" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}

	// Fetch when the CacheFill returns an error
	result, err = cache.Fetch("uncachedkey", time.Second, func() ([]byte, error) {
		return nil, fmt.Errorf("oh no")
	})
	if err == nil {
		t.Error("Fetch() with a failed CacheFill should return an error, but it didn't")
	}

	// Fetch after a failure runs the CacheFill function
	result, err = cache.Fetch("uncachedkey", time.Second, func() ([]byte, error) {
		return []byte("normal"), nil
	})
	if err != nil {
		t.Error("An uncached Fetch() call returned an error when it shouldn't have")
	}
	if string(result) != "normal" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}
}
