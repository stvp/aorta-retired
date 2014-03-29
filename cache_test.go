package main

import (
	"fmt"
	"testing"
	"time"
)

func TestCacheFetch(t *testing.T) {
	cache := NewCache()

	// Fetch an un-cached value
	result, err := cache.Fetch("mykey", time.Second, func() (string, error) {
		return "cool", nil
	})
	if err != nil {
		t.Error("An uncached Fetch() call returned an error when it shouldn't have")
	}
	if result != "cool" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}

	// Fetch a cached value
	cache.Fetch("mykey", time.Second, func() (string, error) {
		t.Error("Fetch called the CacheFill function when the key was already cached")
		return "nope", nil
	})

	// Fetch a cached but stale value
	cache.values["mykey"] = TimeString{
		String: "old",
		Time:   time.Now().Add(-time.Minute),
	}
	result, err = cache.Fetch("mykey", 30*time.Second, func() (string, error) {
		return "even cooler", nil
	})
	if err != nil {
		t.Error("A stale, cached Fetch() call returned an error when it shouldn't have")
	}
	if result != "even cooler" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}

	// Fetch when the CacheFill returns an error
	result, err = cache.Fetch("uncachedkey", time.Second, func() (string, error) {
		return "", fmt.Errorf("oh no")
	})
	if err == nil {
		t.Error("Fetch() with a failed CacheFill should return an error, but it didn't")
	}

	// Fetch after a failure runs the CacheFill function
	result, err = cache.Fetch("uncachedkey", time.Second, func() (string, error) {
		return "normal", nil
	})
	if err != nil {
		t.Error("An uncached Fetch() call returned an error when it shouldn't have")
	}
	if result != "normal" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache()

	cache.values["mykey"] = TimeString{
		String: "foo",
		Time:   time.Now(),
	}
	cache.Clear("mykey")

	result, err := cache.Fetch("mykey", time.Second, func() (string, error) {
		return "new value", nil
	})
	if err != nil {
		t.Error("An uncached Fetch() call returned an error when it shouldn't have")
	}
	if result != "new value" {
		t.Errorf("Fetch() returned the wrong value: %#v", result)
	}
}
