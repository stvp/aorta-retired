package aorta

import (
	"bufio"
	"errors"
	"github.com/stvp/resp"
	"net"
	"net/url"
	"time"
)

var (
	ErrReadClosedConn = errors.New("aorta: can't read from closed connection")
)

type ServerConn struct {
	// Redis server connection settings
	host string
	auth string

	RESPConn
}

func NewServerConn(redisUrl url.URL, timeout time.Duration) *ServerConn {
	server := &ServerConn{
		host: redisUrl.Host,
		RESPConn: RESPConn{
			timeout: timeout,
		},
	}
	if redisUrl.User != nil {
		server.auth, _ = redisUrl.User.Password()
	}
	return server
}

func (s *ServerConn) Run(command resp.Command) ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return s.run(command)
}

func (s *ServerConn) dial() error {
	if s.conn != nil {
		s.close()
	}

	conn, err := net.DialTimeout("tcp", s.host, s.timeout)
	if err == nil {
		s.conn = conn
		// TODO make configurable buffer sizes
		s.bw = bufio.NewWriterSize(s.conn, 8192)
		s.br = resp.NewReaderSize(s.conn, 8192)
	}

	if len(s.auth) > 0 {
		_, err = resp.Parse(s.run(resp.NewCommand("AUTH", s.auth)))
		if err != nil {
			return s.handleError(err)
		}
	}

	return err
}

func (s *ServerConn) run(command resp.Command) (response []byte, err error) {
	err = s.sendCommand(command)
	if err != nil {
		return []byte{}, err
	}

	s.updateConnDeadline()
	response, err = s.br.ReadObjectBytes()
	return response, s.handleError(err)
}

func (s *ServerConn) sendCommand(command resp.Command) (err error) {
	if s.conn == nil {
		err = s.dial()
		if err != nil {
			return err
		}
	}
	return s.sendObject(command)
}

func (s *ServerConn) receiveResponse() (response []byte, err error) {
	if s.conn == nil {
		return []byte{}, ErrReadClosedConn
	}
	return s.receiveObject()
}
