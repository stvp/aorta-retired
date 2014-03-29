package main

import (
	"fmt"
	"github.com/stvp/resp"
	"net"
	"time"
)

// NOTE any number of goroutines can use a ServerConn, so lock everything.
type ServerConn struct {
	// Redis server connection settings
	host string
	auth string

	RESPConn
}

func NewServerConn(host, port, auth string, timeout time.Duration) *ServerConn {
	server := &ServerConn{
		host: fmt.Sprintf("%s:%s", host, port),
		auth: auth,
		RESPConn: RESPConn{
			timeout: timeout,
		},
	}
	return server
}

func (s *ServerConn) Do(command resp.Command) (response []byte, err error) {
	s.Lock()
	defer s.Unlock()
	return s.do(command)
}

func (s *ServerConn) dial() (response []byte, err error) {
	s.close()

	conn, err := net.DialTimeout("tcp", s.host, s.timeout)
	if err == nil {
		s.conn = conn
		s.reader = resp.NewReaderSize(s.conn, 8192)
		if len(s.auth) > 0 {
			response, err = s.do(resp.NewCommand("AUTH", s.auth))
		}
	}

	return response, wrapErr(err)
}

func (s *ServerConn) do(command resp.Command) (response []byte, err error) {
	if s.conn == nil {
		response, err = s.dial()
		if err == ErrConnClosed {
			panic(err)
		}
		if err != nil {
			return response, err
		}
	}

	err = s.write(command)
	if err != nil {
		return []byte{}, err
	}

	return s.readObjectBytes()
}
