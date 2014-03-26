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
	pongReply = []byte("+PONG\r\n")

	goodConfig = tempredis.Config{
		"port": "11040",
	}
	goodUrl = url.URL{
		Host: goodConfig.Address(),
	}

	goodAuthConfig = tempredis.Config{
		"port":        "11041",
		"requirepass": "pw",
	}
	goodAuthUrl = url.URL{
		Host: goodAuthConfig.Address(),
		User: url.UserPassword("", goodAuthConfig["requirepass"]),
	}
)

func tempTimeoutServer(fn func(err error)) {
	tempredis.Temp(goodAuthConfig, func(err error) {
		if err == nil {
			server := NewServerConn(goodAuthUrl, time.Second)
			server.sendCommand(resp.NewCommand("DEBUG", "SLEEP", "1"))
		}
		fn(err)
	})
}

func TestServerRun(t *testing.T) {
	// Handles no auth
	tempredis.Temp(goodConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}
		server := NewServerConn(goodUrl, time.Millisecond)
		response, err := server.Run(resp.NewCommand("PING"))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(pongReply, response) {
			t.Errorf("expected: %#v\ngot: %#v", pongReply, response)
		}
	})

	// Handles auth
	tempredis.Temp(goodAuthConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}
		server := NewServerConn(goodAuthUrl, time.Millisecond)
		response, err := server.Run(resp.NewCommand("PING"))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(pongReply, response) {
			t.Errorf("expected: %#v\ngot: %#v", pongReply, response)
		}
	})

	// Handles auth timeouts
	tempTimeoutServer(func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		server := NewServerConn(goodAuthUrl, time.Millisecond)
		_, err = server.Run(resp.NewCommand("PING"))
		if err == nil {
			t.Error("expected error but got none")
		}
	})

	// Handles read timeouts
	tempredis.Temp(goodAuthConfig, func(err error) {
		if err != nil {
			t.Fatal(err)
		}

		server := NewServerConn(goodAuthUrl, time.Millisecond)
		err = server.dial()
		if err != nil {
			t.Errorf("failed to connect to temp server")
		}

		// Pause the server
		slow := NewServerConn(goodAuthUrl, time.Second)
		slow.sendCommand(resp.NewCommand("DEBUG", "SLEEP", "1"))

		_, err = server.Run(resp.NewCommand("PING"))
		if err == nil {
			t.Error("expected error but got none")
		}
	})
}
