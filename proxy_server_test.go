package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/stvp/tempredis"
	// "github.com/stvp/resp"
	"testing"
	"time"
)

func startProxy() (*ProxyServer, *tempredis.Server) {
	proxy := NewProxyServer("0.0.0.0:12001", "pw", time.Second, time.Millisecond)
	err := proxy.Listen()
	if err != nil {
		panic(err)
	}
	server, err := tempredis.Start(goodConfig)
	if err != nil {
		panic(err)
	}
	return proxy, server
}

func withProxy(fn func(*ProxyServer)) {
	proxy, server := startProxy()
	defer proxy.Close()
	defer server.Term()
	fn(proxy)
}

func dialProxy(proxy *ProxyServer) redis.Conn {
	redis, err := redis.Dial("tcp", proxy.bind)
	if err != nil {
		panic(err)
	}
	return redis
}

func TestProxyServer_NoAuth(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		conn := dialProxy(proxy)
		_, err := conn.Do("PROXY", "localhost", "9999", "x")
		if err == nil || err.Error() != "NOAUTH Authentication required." {
			t.Fatalf("expected auth error, got: %#v", err)
		}
	})
}

func TestProxyServer_BadAuth(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		conn := dialProxy(proxy)
		_, err := conn.Do("AUTH", "wrong")
		if err == nil || err.Error() != "ERR invalid password" {
			t.Fatalf("expected auth error, got: %#v", err)
		}
	})
}

func TestProxyServer_GoodAuth(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		conn := dialProxy(proxy)
		_, err := conn.Do("AUTH", "pw")
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestProxyServer_NoProxy(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		_, err := conn.Do("PING")
		if err == nil || err.Error() != "aorta: proxy destination not set" {
			t.Fatalf("expected proxy error, got: %#v", err)
		}
	})
}

func TestProxyServer_InvalidProxy(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		_, err := conn.Do("PROXY", "0.0.0.0", "6379")
		if err == nil || err.Error() != "ERR wrong number of arguments for 'proxy' command" {
			t.Fatalf("expected proxy error, got: %#v", err)
		}
	})
}

func TestProxyServer_Proxy(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		response, err := redis.String(conn.Do("PROXY", "0.0.0.0", goodConfig["port"], goodConfig["requirepass"]))
		if err != nil {
			t.Fatal(err)
		}
		if response != "OK" {
			t.Fatalf("Expected OK, got: %#v", response)
		}
		response, err = redis.String(conn.Do("PROXY", "0.0.0.0", "12002", "pw"))
	})
}

func TestProxyServer_ProxyToDeadServer(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		response, err := redis.String(conn.Do("PROXY", "0.0.0.0", "12002", "pw"))
		if err != nil {
			t.Fatal(err)
		}
		if response != "OK" {
			t.Fatalf("Expected OK, got: %#v", response)
		}
		response, err = redis.String(conn.Do("PING"))
		if err == nil || err.Error() != "dial tcp 0.0.0.0:12002: connection refused" {
			t.Fatalf("expected dial error, got: %#v", err)
		}
	})
}

// TODO
// * bad proxy call
//   * bad hostname
//   * timeout
//   * bad auth
// * tests for servers with no auth
//   * auth provided (error)
//   * good (no auth)
// * client calls QUIT
// * client connection closes
// * pipelining
// * scripting?
// * proxy to one server and then switch to another
