package aorta

import (
	"github.com/stvp/resp"
	"github.com/stvp/tempredis"
	"net/url"
	"reflect"
	"testing"
	"time"
)

var (
	pongReply = resp.String("+PONG\r\n")

	goodConfig = tempredis.Config{
		"port": "22000",
	}
	goodUrl = url.URL{
		Host: goodConfig.Address(),
	}

	goodAuthConfig = tempredis.Config{
		"port":        "22001",
		"requirepass": "pw",
	}
	goodAuthUrl = url.URL{
		Host: goodAuthConfig.Address(),
		User: url.UserPassword("", goodAuthConfig["requirepass"]),
	}
	badAuthUrl = url.URL{
		Host: goodAuthConfig.Address(),
		User: url.UserPassword("", "oops"),
	}
)

// -- Helpers

func tempTimeoutServer(fn func(redisUrl url.URL, err error)) {
	server, err := tempredis.Start(goodAuthConfig)
	defer server.Kill()
	if err == nil {
		// Block the server
		conn := NewServerConn(goodAuthUrl, time.Minute)
		conn.dial()
		conn.write(resp.NewCommand("DEBUG", "SLEEP", "60"))
	}

	fn(goodAuthUrl, err)
}

// -- Tests

func TestServerDo_NoAuth(t *testing.T) {
	tempredis.Temp(goodConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(goodUrl, time.Millisecond)
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
	tempredis.Temp(goodAuthConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(goodAuthUrl, time.Millisecond)
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
	tempredis.Temp(goodAuthConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(badAuthUrl, time.Millisecond)
		_, err = conn.Do(resp.NewCommand("PING"))
		if _, ok := err.(resp.Error); !ok {
			t.Errorf("expected a resp.Error but got %#v", err)
		}
	})
}

func TestServerDo_AuthTimeout(t *testing.T) {
	tempTimeoutServer(func(redisUrl url.URL, err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(redisUrl, time.Millisecond)
		_, err = conn.Do(resp.NewCommand("PING"))
		if err != ErrTimeout {
			t.Errorf("expected ErrTimeout but got %#v", err)
		}
	})
}

func TestServerDo_ReadTimeout(t *testing.T) {
	tempredis.Temp(goodAuthConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		conn := NewServerConn(goodAuthUrl, time.Millisecond)
		_, err = conn.dial()
		if err != nil {
			t.Errorf("failed to connect to temp server")
		}

		slowConn := NewServerConn(goodAuthUrl, time.Second)
		slowConn.dial()
		slowConn.write(resp.NewCommand("DEBUG", "SLEEP", "1"))

		_, err = conn.Do(resp.NewCommand("PING"))
		if err != ErrTimeout {
			t.Errorf("expected ErrTimeout but got %#v", err)
		}
	})
}

func TestServerDo_ConnectionDrop(t *testing.T) {
	server, err := tempredis.Start(goodAuthConfig)
	if err != nil {
		server.Term()
		t.Fatal(err)
	}
	conn := NewServerConn(goodAuthUrl, time.Millisecond)
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
	server, err = tempredis.Start(goodAuthConfig)
	defer server.Term()
	if err != nil {
		t.Fatal(err)
	}

	// Should re-connect automaticaly
	response, err := conn.Do(resp.NewCommand("PING"))
	if err != nil {
		t.Errorf("%#v\n", err)
	}
	if !reflect.DeepEqual(pongReply, response) {
		t.Errorf("expected: %#v\ngot: %#v", pongReply, response)
	}
}
