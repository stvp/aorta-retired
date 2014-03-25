package aorta

import (
	"bufio"
	"github.com/stvp/resp"
	"net"
	"net/url"
	"sync"
	"time"
)

type ServerConn struct {
	// Read and write timeout
	timeout time.Duration

	// Redis connection
	host string
	auth string
	conn net.Conn

	// Redis I/O buffers
	bw *bufio.Writer
	br *resp.Reader

	sync.Mutex
}

func NewServerConn(redisUrl url.URL, timeout time.Duration) *ServerConn {
	server := &ServerConn{
		host:    redisUrl.Host,
		timeout: timeout,
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

func (s *ServerConn) Close() error {
	s.Lock()
	defer s.Unlock()
	return s.close()
}

func (s *ServerConn) updateConnDeadline() {
	s.conn.SetDeadline(time.Now().Add(s.timeout))
}

func (s *ServerConn) dial() error {
	var err error

	if s.conn != nil {
		s.close()
	}

	conn, err := net.DialTimeout("tcp", s.host, s.timeout)
	if err == nil {
		s.conn = conn
		s.bw = bufio.NewWriterSize(s.conn, 8192)
		s.br = resp.NewReaderSize(s.conn, 8192)
	}

	if len(s.auth) > 0 {
		_, err = resp.Parse(s.Run(resp.NewCommandStrings("AUTH", s.auth)))
		if err != nil {
			s.close()
			return err
		}
	}

	return err
}

func (s *ServerConn) run(command resp.Command) (response []byte, err error) {
	if s.conn == nil {
		err = s.dial()
		if err != nil {
			return nil, err
		}
	}

	s.updateConnDeadline()
	_, err = s.bw.Write(command)
	if err == nil {
		err = s.bw.Flush()
	}
	if err != nil {
		return nil, err
	}

	s.updateConnDeadline()
	return s.br.ReadObjectBytes()
}

func (s *ServerConn) close() (err error) {
	if s.conn != nil {
		err = s.conn.Close()
		s.conn = nil
		s.bw = nil
		s.br = nil
	}

	return err
}
