package main

import (
	"github.com/stvp/resp"
	"net"
	"sync"
	"time"
)

// A RESPConn is a TCP connection to a Redis server or client.
type RESPConn struct {
	timeout time.Duration
	conn    net.Conn
	reader  *resp.Reader
	sync.Mutex
}

// Close closes the underlying TCP connection. It waits for any currently
// running reads or writes to finish (or fail) before closing the connection.
func (c *RESPConn) Close() error {
	c.Lock()
	defer c.Unlock()
	return c.close()
}

// write writes the given bytes to the TCP connection, returning any errors
// encountered.
func (c *RESPConn) write(raw []byte) error {
	c.conn.SetDeadline(time.Now().Add(c.timeout))

	_, err := c.conn.Write(raw)
	err = wrapErr(err)
	if err == ErrConnClosed {
		c.close()
	}

	return err
}

// readObjectBytes reads one entire RESP object from the TCP connection. If a
// connection error is encountered while reading (closed connection, timeout,
// etc.), the connection is closed and the error returned.
func (c *RESPConn) readObjectBytes() (bytes []byte, err error) {
	c.conn.SetDeadline(time.Now().Add(c.timeout))

	bytes, err = c.reader.ReadObjectBytes()
	err = wrapErr(err)
	if err == ErrConnClosed {
		c.close()
	}
	return bytes, err
}

func (c *RESPConn) close() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
		c.reader = nil
	}

	return wrapErr(err)
}
