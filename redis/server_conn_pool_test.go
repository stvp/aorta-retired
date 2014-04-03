package redis

import (
	"sync"
	"testing"
	"time"
)

func TestServerConnPool(t *testing.T) {
	pool := NewServerConnPool()
	serverConn := pool.Get("cool.com:1234", "pw", time.Millisecond)
	if serverConn.address != "cool.com:1234" {
		t.Errorf("incorrect address for ServerConn: %s", serverConn.address)
	}
	if serverConn.auth != "pw" {
		t.Errorf("incorrect auth for ServerConn: %s", serverConn.auth)
	}

	serverConn2 := pool.Get("cool.com:1234", "pw", time.Millisecond)
	if serverConn2 != serverConn {
		t.Errorf("subsequent Get for same server didn't return same ServerConn: %#v", serverConn2)
	}

	serverConn3 := pool.Get("cool.com:1234", "other", time.Millisecond)
	if serverConn3 == serverConn {
		t.Errorf("different auth should return different ServerConn, but didn't")
	}
}

func TestServerConnPoolExpire(t *testing.T) {
	now := time.Now()
	pool := NewServerConnPool()
	pool.Get("foo1:6379", "baz", time.Second)
	pool.Get("foo2:6379", "baz", time.Second)
	pool.Get("foo3:6379", "baz", time.Second)
	pool.pool["foo1:6379:baz"].LastUsed = now
	pool.pool["foo2:6379:baz"].LastUsed = now.Add(-time.Minute)
	pool.pool["foo3:6379:baz"].LastUsed = now.Add(-time.Hour)
	if expired := pool.Expire(now.Add(-time.Minute)); expired != 1 {
		t.Errorf("expected to expire 1 connection, expired %d", expired)
	}
	if _, ok := pool.pool["foo1:6379:baz"]; !ok {
		t.Error("shouldn't have expired foo1")
	}
	if _, ok := pool.pool["foo2:6379:baz"]; !ok {
		t.Error("shouldn't have expired foo2")
	}
	if _, ok := pool.pool["foo3:6379:baz"]; ok {
		t.Error("should have expired foo3")
	}
}

func BenchmarkServerConnPool_1(b *testing.B) {
	pool := NewServerConnPool()
	for i := 0; i < b.N; i++ {
		pool.Get("cool.com:1234", "pw", time.Millisecond)
	}
}

func BenchmarkServerConnPool_10(b *testing.B) {
	pool := NewServerConnPool()
	servers := [][]string{
		[]string{"cool0.com:1234", "pw"},
		[]string{"cool1.com:1234", "pw"},
		[]string{"cool2.com:1234", "pw"},
		[]string{"cool3.com:1234", "pw"},
		[]string{"cool4.com:1234", "pw"},
		[]string{"cool5.com:1234", "pw"},
		[]string{"cool6.com:1234", "pw"},
		[]string{"cool7.com:1234", "pw"},
		[]string{"cool8.com:1234", "pw"},
		[]string{"cool9.com:1234", "pw"},
	}
	var deets []string
	for i := 0; i < b.N; i++ {
		deets = servers[i%len(servers)]
		pool.Get(deets[0], deets[1], time.Millisecond)
	}
}

func BenchmarkServerConnPool_2_GoRoutines(b *testing.B) {
	wg := sync.WaitGroup{}
	pool := NewServerConnPool()

	for i := 0; i < b.N; i++ {
		wg.Add(2)
		go func() {
			pool.Get("cool.com:1234", "pw", time.Millisecond)
			wg.Done()
		}()
		go func() {
			pool.Get("cool.com:1234", "pw", time.Millisecond)
			wg.Done()
		}()
	}

	wg.Wait()
}
