package aorta

import (
	"bufio"
	"github.com/stvp/resp"
	"io"
	"net"
	"sync"
	"time"
)

type RESPConn struct {
	// Read and write timeout
	timeout time.Duration

	// TCP connection
	conn net.Conn

	// RESP I/O buffers
	bw *bufio.Writer
	br *resp.Reader

	sync.Mutex
}

func (c *RESPConn) Close() error {
	c.Lock()
	defer c.Unlock()
	return c.close()
}

func (c *RESPConn) updateConnDeadline() {
	c.conn.SetDeadline(time.Now().Add(c.timeout))
}

func (c *RESPConn) handleError(err error) error {
	if err == io.EOF {
		c.close()
	}
	return err
}

func (c *RESPConn) close() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
		c.bw = nil
		c.br = nil
	}

	return err
}
