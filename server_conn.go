package main

import (
	"fmt"
	"github.com/stvp/resp"
	"net"
	"time"
)

type ServerConn struct {
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

func (s *ServerConn) Do(command resp.Command) (response interface{}, err error) {
	s.Lock()
	defer s.Unlock()

	if s.conn == nil {
		err = s.dial()
		if err != nil {
			return nil, err
		}
	}

	return s.do(command)
}

func (s *ServerConn) dial() (err error) {
	s.close()

	conn, err := net.DialTimeout("tcp", s.host, s.timeout)
	if err != nil {
		return wrapErr(err)
	}

	s.conn = conn
	s.reader = resp.NewReaderSize(s.conn, 8192)
	if len(s.auth) > 0 {
		_, err = s.do(resp.NewCommand("AUTH", s.auth))
		if err != nil {
			s.close()
			return err
		}
	}

	return nil
}

func (s *ServerConn) do(command resp.Command) (response interface{}, err error) {
	err = s.write(command)
	if err != nil {
		return nil, err
	}
	response, err = s.readObject()
	if err == nil {
		if e, ok := response.(resp.Error); ok {
			err = e
		}
	}
	return response, err
}
