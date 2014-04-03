package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/stvp/resp"
	"github.com/stvp/tempredis"
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

// -- Helpers

func startProxyAndServers(serverCount int) (*ProxyServer, []*tempredis.Server) {
	proxy := NewProxyServer("0.0.0.0:12001", "pw", 10*time.Millisecond, 10*time.Millisecond)
	err := proxy.Listen()
	if err != nil {
		panic(err)
	}
	servers := make([]*tempredis.Server, serverCount)
	for i := 0; i < serverCount; i++ {
		server, err := tempredis.Start(tempredis.Config{
			"port":        strconv.Itoa(22000 + i),
			"requirepass": strconv.Itoa(i),
		})
		if err != nil {
			panic(err)
		}
		servers[i] = server
	}
	return proxy, servers
}

func withProxy(fn func(*ProxyServer)) {
	proxy := NewProxyServer("0.0.0.0:12001", "pw", 10*time.Millisecond, 10*time.Millisecond)
	defer proxy.Close()
	err := proxy.Listen()
	if err != nil {
		panic(err)
	}
	fn(proxy)
}

func withProxyAndServers(serverCount int, fn func(*ProxyServer, []*tempredis.Server)) {
	proxy, servers := startProxyAndServers(serverCount)
	defer proxy.Close()
	defer func() {
		for _, server := range servers {
			server.Term()
		}
	}()
	fn(proxy, servers)
}

func dialProxy(proxy *ProxyServer) redis.Conn {
	timeout := 50 * time.Millisecond
	redis, err := redis.DialTimeout("tcp", proxy.bind, timeout, timeout, timeout)
	if err != nil {
		panic(err)
	}
	return redis
}

func blockServer(server *tempredis.Server) {
	conn := NewServerConn(server.Config.Address(), server.Config.Password(), time.Minute)
	conn.dial()
	conn.write(resp.NewCommand("DEBUG", "SLEEP", "60"))
}

func connClosed(conn net.Conn) bool {
	conn.SetReadDeadline(time.Now().Add(time.Millisecond))
	b := make([]byte, 1)
	_, err := conn.Read(b)
	return err == io.EOF
}

func redisConnClosed(conn redis.Conn) bool {
	_, err := conn.Receive()
	return err != nil && (err == io.EOF || err.Error() == "use of closed network connection")
}

// -- Tests

func TestProxyServer_Auth(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		// No auth
		conn := dialProxy(proxy)
		_, err := conn.Do("PROXY", "localhost", "9999", "x")
		if err == nil || err.Error() != "NOAUTH Authentication required." {
			t.Fatalf("expected auth error, got: %#v", err)
		}
		if !redisConnClosed(conn) {
			t.Error("client connection is still open")
		}

		// Bad auth
		conn = dialProxy(proxy)
		_, err = conn.Do("AUTH", "wrong")
		if err == nil || err.Error() != "ERR invalid password" {
			t.Fatalf("expected auth error, got: %#v", err)
		}

		// Good auth
		conn = dialProxy(proxy)
		_, err = conn.Do("AUTH", "pw")
		if err != nil {
			t.Fatal(err)
		}
		// ... then bad auth again
		_, err = conn.Do("AUTH", "x")
		if err == nil || err.Error() != "ERR invalid password" {
			t.Fatalf("expected auth error, got: %#v", err)
		}
		_, err = conn.Do("PROXY", "0.0.0.0", "9999", "pw")
		if err == nil || err.Error() != "NOAUTH Authentication required." {
			t.Fatalf("expected auth error, got: %#v", err)
		}
	})
}

func TestProxyServer_BadProxy(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		// No proxy command
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		_, err := conn.Do("PING")
		if err == nil || err.Error() != "aorta: proxy destination not set" {
			t.Fatalf("expected proxy error, got: %#v", err)
		}

		// Invalid proxy command
		conn = dialProxy(proxy)
		conn.Do("AUTH", "pw")
		_, err = conn.Do("PROXY", "0.0.0.0", "6379")
		if err == nil || err.Error() != "ERR wrong number of arguments for 'proxy' command" {
			t.Fatalf("expected proxy error, got: %#v", err)
		}

		// Proxy to dead server
		conn = dialProxy(proxy)
		conn.Do("AUTH", "pw")
		response, err := redis.String(conn.Do("PROXY", "0.0.0.0", "9999", "pw"))
		if err != nil {
			t.Fatal(err)
		}
		if response != "OK" {
			t.Fatalf("Expected OK, got: %#v", response)
		}
		response, err = redis.String(conn.Do("PING"))
		if err == nil || err.Error() != "dial tcp 0.0.0.0:9999: connection refused" {
			t.Fatalf("expected dial error, got: %#v", err)
		}

		// Proxy to bad host
		conn = dialProxy(proxy)
		conn.Do("AUTH", "pw")
		response, err = redis.String(conn.Do("PROXY", "invalid", "9999", "pw"))
		if err != nil {
			t.Fatal(err)
		}
		if response != "OK" {
			t.Fatalf("Expected OK, got: %#v", response)
		}
		response, err = redis.String(conn.Do("PING"))
		// TODO this is super brittle -- need a better way to test this
		if err == nil || err.Error() != "dial tcp: lookup invalid: no such host" {
			t.Fatalf("expected dial error, got: %#v", err)
		}
	})
}

func TestProxyServer_GoodProxy(t *testing.T) {
	withProxyAndServers(1, func(proxy *ProxyServer, servers []*tempredis.Server) {
		serverConfig := servers[0].Config

		// Good proxy command
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		response, err := redis.String(conn.Do("PROXY", serverConfig.Bind(), serverConfig.Port(), serverConfig.Password()))
		if err != nil {
			t.Fatal(err)
		}
		if response != "OK" {
			t.Fatalf("Expected OK, got: %#v", response)
		}

		// Proxy with incorrect auth
		conn = dialProxy(proxy)
		conn.Do("AUTH", "pw")
		response, err = redis.String(conn.Do("PROXY", serverConfig.Bind(), serverConfig.Port(), "x"))
		if err != nil {
			t.Fatal(err)
		}
		if response != "OK" {
			t.Fatalf("Expected OK, got: %#v", response)
		}
		response, err = redis.String(conn.Do("PING"))
		if err == nil || err.Error() != "ERR invalid password" {
			t.Fatalf("expected dial error, got: %#v", err)
		}
	})
}

func TestProxyServer_ProxyToBlockedServer(t *testing.T) {
	withProxyAndServers(1, func(proxy *ProxyServer, servers []*tempredis.Server) {
		server := servers[0]
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		conn.Do("PROXY", server.Config.Bind(), server.Config.Port(), server.Config.Password())
		blockServer(servers[0])
		_, err := redis.String(conn.Do("PING"))
		if err == nil || err.Error() != "aorta: timeout" {
			t.Fatal(err)
		}
	})
}

func TestProxyServer_ClientQuit(t *testing.T) {
	withProxy(func(proxy *ProxyServer) {
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		conn.Do("QUIT")
		if !redisConnClosed(conn) {
			t.Error("client connection is still open")
		}
	})
}

func TestProxyServer_SwitchServer(t *testing.T) {
	withProxyAndServers(2, func(proxy *ProxyServer, servers []*tempredis.Server) {
		conn := dialProxy(proxy)
		conn.Do("AUTH", "pw")
		conn.Do("PROXY", servers[0].Config.Bind(), servers[0].Config.Port(), servers[0].Config.Password())
		conn.Do("SET", "foo", "bar")
		foo, err := redis.String(conn.Do("GET", "foo"))
		if err != nil {
			t.Fatal(err)
		}
		if foo != "bar" {
			t.Fatalf("expected \"bar\", got: %#v", foo)
		}

		// Switch to good address
		conn.Do("PROXY", servers[1].Config.Bind(), servers[1].Config.Port(), servers[1].Config.Password())
		_, err = redis.String(conn.Do("GET", "foo"))
		if err == nil || err.Error() != "redigo: nil returned" {
			t.Fatalf("should be connected to servers[1], got: %#v", err)
		}

		// Switch to bad address
		conn.Do("PROXY", servers[1].Config.Bind(), "9999", servers[1].Config.Password())
		_, err = redis.String(conn.Do("GET", "foo"))
		if err == nil || err.Error() != "dial tcp 0.0.0.0:9999: connection refused" {
			t.Fatalf("expected connection error but got: %#v", err)
		}
		if !redisConnClosed(conn) {
			t.Error("client connection is still open")
		}
	})
}

func BenchmarkDirectServer(b *testing.B) {
	server, err := tempredis.Start(tempredis.Config{"port": "16000"})
	if err != nil {
		b.Fatal(err)
	}
	defer server.Term()
	conn, err := redis.Dial("tcp", "0.0.0.0:16000")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn.Do("PING")
	}
}

func BenchmarkProxyServer(b *testing.B) {
	withProxyAndServers(1, func(proxy *ProxyServer, servers []*tempredis.Server) {
		serverConfig := servers[0].Config

		conn := dialProxy(proxy)
		_, err := conn.Do("AUTH", "pw")
		if err != nil {
			b.Fatal(err)
		}
		_, err = conn.Do("PROXY", serverConfig.Bind(), serverConfig.Port(), serverConfig.Password())
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			conn.Do("PING")
		}

		b.StopTimer()
	})
}
