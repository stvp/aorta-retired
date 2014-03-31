package main

import (
	"fmt"
	"github.com/stvp/resp"
	"net"
	"time"
)

// ServerConn is a connection to a Redis server.
type ServerConn struct {
	host string
	auth string
	RESPConn
}

// NewServerConn returns a new ServerConn configured with the given host, port,
// and password. It uses the given timeout duration for both reading and
// writing. It does not proactively connect to the Redis server.
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

// Do runs the given Redis command on the Redis server and returns the
// response. If there is a connection error, the underlying connection will be
// closed and the error returned. If Redis returns an error, that error will be
// returned as both the response and the error.
func (s *ServerConn) Do(command resp.Command) (response []byte, err error) {
	s.Lock()
	defer s.Unlock()
	return s.do(command)
}

// dial closes the current TCP connection (if any) and opens a new TCP
// connection to the Redis server and authenticates (if needed).
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
