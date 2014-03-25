package aorta

import (
	"github.com/stvp/resp"
	"github.com/stvp/tempredis"
	"net/url"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	s, err := tempredis.Start(tempredis.Config{"port": "11042"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Term()

	server := NewServerConn(url.URL{Host: "0.0.0.0:11042"}, time.Millisecond)
	_, err = server.Run(resp.NewCommandStrings("PING"))
	if err != nil {
		t.Fatal(err)
	}
}
