package main

import (
	"github.com/stvp/resp"
	"net"
	"time"
)

type ClientConn struct {
	RESPConn
}

func NewClientConn(conn net.Conn, timeout time.Duration) *ClientConn {
	client := &ClientConn{
		RESPConn: RESPConn{
			timeout: timeout,
			conn:    conn,
			reader:  resp.NewReaderSize(conn, 8192),
		},
	}

	return client
}

func (c *ClientConn) ReadCommand() (resp.Command, error) {
	c.Lock()
	defer c.Unlock()

	bytes, err := c.readObjectBytes()
	if err != nil {
		return nil, err
	}

	return resp.Command(bytes), nil
}

func (c *ClientConn) Write(raw []byte) error {
	c.Lock()
	defer c.Unlock()

	return c.write(raw)
}

func (c *ClientConn) WriteError(msg string) error {
	return c.Write(resp.NewError(msg))
}
