package main

import (
	"github.com/stvp/resp"
	"github.com/stvp/tempredis"
	"reflect"
	"testing"
	"time"
)

var (
	goodConfig = tempredis.Config{
		"port":        "22000",
		"requirepass": "pw",
	}
	goodAddress = goodConfig.Address()
	goodAuth    = "pw"

	goodConfigNoAuth = tempredis.Config{
		"port": "22000",
	}
)

// -- Helpers

func tempTimeoutServer(fn func(address, auth string, err error)) {
	server, err := tempredis.Start(goodConfig)
	defer server.Kill()
	if err == nil {
		// Block the server
		conn := NewServerConn(goodAddress, goodAuth, time.Minute)
		conn.dial()
		conn.write(resp.NewCommand("DEBUG", "SLEEP", "60"))
	}

	fn(goodAddress, goodAuth, err)
}

// -- Tests

func TestServerDo_NoAuth(t *testing.T) {
	tempredis.Temp(goodConfigNoAuth, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		// Without password
		conn := NewServerConn(goodAddress, "", time.Millisecond)
		response, err := conn.Do(resp.NewCommand("PING"))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(resp.PONG, response) {
			t.Errorf("expected: %#v\ngot: %#v", resp.PONG, response)
		}

		// With password
		conn = NewServerConn(goodAddress, "x", time.Millisecond)
		response, err = conn.Do(resp.NewCommand("PING"))
		if _, ok := err.(resp.Error); !ok {
			t.Errorf("expected resp.Error, got: %#v", err)
		}
	})
}

func TestServerDo_Auth(t *testing.T) {
	tempredis.Temp(goodConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		// Good auth
		conn := NewServerConn(goodAddress, goodAuth, time.Millisecond)
		response, err := conn.Do(resp.NewCommand("PING"))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(resp.PONG, response) {
			t.Errorf("expected: %#v\ngot: %#v", resp.PONG, response)
		}

		// Bad auth
		conn = NewServerConn(goodAddress, "bad", time.Millisecond)
		_, err = conn.Do(resp.NewCommand("PING"))
		if _, ok := err.(resp.Error); !ok {
			t.Errorf("expected resp.Error as error, got: %#v", err)
		}
	})
}

func TestServerDo_AuthTimeout(t *testing.T) {
	tempTimeoutServer(func(address, auth string, err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(address, auth, time.Millisecond)
		_, err = conn.Do(resp.NewCommand("PING"))
		if err != ErrTimeout {
			t.Errorf("expected ErrTimeout but got %#v", err)
		}
	})
}

func TestServerDo_Timeout(t *testing.T) {
	tempredis.Temp(goodConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		// Connect to server
		conn := NewServerConn(goodAddress, goodAuth, time.Millisecond)
		err = conn.dial()
		if err != nil {
			t.Errorf("failed to connect to temp server")
		}

		// Simulate a long-running command
		slowConn := NewServerConn(goodAddress, goodAuth, time.Second)
		slowConn.dial()
		slowConn.write(resp.NewCommand("DEBUG", "SLEEP", "1"))

		_, err = conn.Do(resp.NewCommand("PING"))
		if err != ErrTimeout {
			t.Errorf("expected ErrTimeout but got %#v", err)
		}
	})
}

func TestServerDo_ConnectionDrop(t *testing.T) {
	server, err := tempredis.Start(goodConfig)
	if err != nil {
		server.Term()
		t.Fatal(err)
	}
	conn := NewServerConn(goodAddress, goodAuth, time.Millisecond)
	_, err = conn.Do(resp.NewCommand("PING"))
	if err != nil {
		server.Term()
		t.Fatal(err)
	}

	// Server connection drops
	server.Term()
	_, err = conn.Do(resp.NewCommand("PING"))
	if err != ErrConnClosed {
		t.Errorf("expected ErrConnClosed but got %#v", err)
	}

	// Server comes back
	server, err = tempredis.Start(goodConfig)
	defer server.Term()
	if err != nil {
		t.Fatal(err)
	}

	// Should re-connect automatically
	response, err := conn.Do(resp.NewCommand("PING"))
	if err != nil {
		t.Errorf("%#v\n", err)
	}
	if !reflect.DeepEqual(resp.PONG, response) {
		t.Errorf("expected: %#v\ngot: %#v", resp.PONG, response)
	}
}
