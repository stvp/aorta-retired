package aorta

import (
	"github.com/stvp/resp"
	"net"
	"sync"
	"time"
)

type RESPConn struct {
	// Read and write timeout
	timeout time.Duration

	// TCP connection
	conn net.Conn

	// RESP reader
	reader *resp.Reader

	sync.Mutex
}

func (c *RESPConn) Close() error {
	c.Lock()
	defer c.Unlock()
	return c.close()
}

func (c *RESPConn) write(raw []byte) error {
	c.conn.SetDeadline(time.Now().Add(c.timeout))

	_, err := c.conn.Write(raw)
	err = wrapErr(err)
	if err == ErrConnClosed {
		c.close()
	}

	return err
}

func (c *RESPConn) readObject() (object interface{}, err error) {
	c.conn.SetDeadline(time.Now().Add(c.timeout))

	object, err = resp.Parse(c.reader.ReadObjectBytes())
	err = wrapErr(err)
	if err == ErrConnClosed {
		c.close()
	}

	return object, err
}

func (c *RESPConn) close() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
		c.reader = nil
	}

	return wrapErr(err)
}
