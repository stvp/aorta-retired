package main

import (
	"github.com/stvp/resp"
	"github.com/stvp/tempredis"
	"reflect"
	"testing"
	"time"
)

var (
	pongReply = []byte("+PONG\r\n")

	goodConfig = tempredis.Config{
		"port":        "22000",
		"requirepass": "pw",
	}
	goodHost = goodConfig.Bind()
	goodPort = goodConfig.Port()
	goodAuth = "pw"

	goodConfigNoAuth = tempredis.Config{
		"port": "22000",
	}
)

// -- Helpers

func tempTimeoutServer(fn func(host, port, auth string, err error)) {
	server, err := tempredis.Start(goodConfig)
	defer server.Kill()
	if err == nil {
		// Block the server
		conn := NewServerConn(goodHost, goodPort, goodAuth, time.Minute)
		conn.dial()
		conn.write(resp.NewCommand("DEBUG", "SLEEP", "60"))
	}

	fn(goodHost, goodPort, goodAuth, err)
}

// -- Tests

func TestServerDo_NoAuth(t *testing.T) {
	tempredis.Temp(goodConfigNoAuth, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(goodHost, goodPort, "", time.Millisecond)
		response, err := conn.Do(resp.NewCommand("PING"))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(pongReply, response) {
			t.Errorf("expected: %#v\ngot: %#v", pongReply, response)
		}
	})
}

func TestServerDo_Auth(t *testing.T) {
	tempredis.Temp(goodConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(goodHost, goodPort, goodAuth, time.Millisecond)
		response, err := conn.Do(resp.NewCommand("PING"))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(pongReply, response) {
			t.Errorf("expected: %#v\ngot: %#v", pongReply, response)
		}
	})
}

func TestServerDo_BadAuth(t *testing.T) {
	tempredis.Temp(goodConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(goodHost, goodPort, "bad", time.Millisecond)
		response, err := conn.Do(resp.NewCommand("PING"))
		if err != nil {
			t.Fatal(err)
		}
		e, err := resp.Parse(response)
		if err != nil {
			t.Fatal(err)
		}
		switch e.(type) {
		case resp.Error:
		default:
			t.Errorf("expected an error response but got: %s", response)
		}
	})
}

func TestServerDo_AuthTimeout(t *testing.T) {
	tempTimeoutServer(func(host, port, auth string, err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(host, port, auth, time.Millisecond)
		_, err = conn.Do(resp.NewCommand("PING"))
		if err != ErrTimeout {
			t.Errorf("expected ErrTimeout but got %#v", err)
		}
	})
}

func TestServerDo_ReadTimeout(t *testing.T) {
	tempredis.Temp(goodConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(goodHost, goodPort, goodAuth, time.Millisecond)
		_, err = conn.dial()
		if err != nil {
			t.Errorf("failed to connect to temp server")
		}

		slowConn := NewServerConn(goodHost, goodPort, goodAuth, time.Second)
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
	conn := NewServerConn(goodHost, goodPort, goodAuth, time.Millisecond)
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
	if !reflect.DeepEqual(pongReply, response) {
		t.Errorf("expected: %#v\ngot: %#v", pongReply, response)
	}
}
