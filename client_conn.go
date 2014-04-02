package main

import (
	"github.com/stvp/resp"
	"net"
	"time"
)

// ClientConn is a connection to a Redis client (redis-cli, etc.)
type ClientConn struct {
	RESPConn
}

// NewClientConn takes an open TCP connection and returns a ClientConn. The
// given timeout is used for both reading and writing.
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

// ReadCommand waits for the next command to be received from the client. If
// there is a connection error, the underlying connection will be closed and an
// error will be returned.
func (c *ClientConn) ReadCommand() (resp.Command, error) {
	c.Lock()
	defer c.Unlock()

	response, err := c.readObject()
	if err != nil {
		return nil, err
	}

	switch obj := response.(type) {
	case resp.Array:
		return resp.Command(obj), nil
	default:
		return nil, ErrInvalidCommandFormat
	}
}

// Write sends the given bytes to the Redis client. If a connection error is
// encountered while sending data to the client, the underlying connection will
// be closed and the error returned.
func (c *ClientConn) Write(raw []byte) error {
	c.Lock()
	defer c.Unlock()
	return c.write(raw)
}

// WriteError takes an error message and sends it to the Redis client at a RESP
// error object.
func (c *ClientConn) WriteError(msg string) error {
	return c.Write(resp.NewError(msg))
}
