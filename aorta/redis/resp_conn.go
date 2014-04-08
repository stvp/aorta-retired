package redis

import (
	"github.com/stvp/resp"
	"net"
	"sync"
	"time"
)

// A RESPConn is a TCP connection to a Redis server or client with methods for
// reading and writing RESP.
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
	if c.conn == nil {
		return ErrConnClosed
	}

	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	_, err := c.conn.Write(raw)
	err = wrapErr(err)
	if err == ErrConnClosed {
		c.close()
	}

	return err
}

func (c *RESPConn) readObject() (obj resp.Object, err error) {
	if c.conn == nil {
		return nil, ErrConnClosed
	}

	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	obj, err = c.reader.ReadObject()
	err = wrapErr(err)
	if err == ErrConnClosed {
		c.close()
	}
	return obj, err
}

func (c *RESPConn) close() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
		c.reader = nil
	}

	return wrapErr(err)
}
