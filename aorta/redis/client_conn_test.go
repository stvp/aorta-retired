package redis

import (
	"bytes"
	"net"
	"testing"
	"time"
)

type fakeConn struct {
	Closed bool
	bytes.Buffer
}

func (c *fakeConn) Close() error {
	c.Closed = true
	return nil
}
func (c *fakeConn) LocalAddr() net.Addr              { return nil }
func (c *fakeConn) RemoteAddr() net.Addr             { return nil }
func (c *fakeConn) SetDeadline(time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(time.Time) error { return nil }

func TestClientConn_ReadCommand(t *testing.T) {
	bad := [][]byte{
		[]byte{},
		[]byte("*1\r\n"),
		[]byte("*100\r\n"),
	}
	for i, test := range bad {
		conn := fakeConn{}
		client := NewClientConn(&conn, time.Millisecond)
		client.Write(test)
		_, err := client.ReadCommand()
		if err == nil {
			t.Errorf("bad[%d]: didn't return error", i)
		}
		if !conn.Closed {
			t.Errorf("bad[%d]: conn should have been closed, but it wasn't", i)
		}
	}

	good := [][]byte{
		[]byte("*1\r\n$4\r\nPING\r\n"),
		[]byte("*2\r\n$4\r\nINFO\r\n$3\r\nALL\r\n-other stuff ignore me"),
	}
	for i, test := range good {
		conn := fakeConn{}
		client := NewClientConn(&conn, time.Millisecond)
		client.Write(test)
		_, err := client.ReadCommand()
		if err != nil {
			t.Errorf("good[%d]: %s", i, err.Error())
		}
		if conn.Closed {
			t.Errorf("good[%d]: conn shouldn't be closed")
		}
	}
}

func TestClientConn_Write(t *testing.T) {
	conn := fakeConn{}
	client := NewClientConn(&conn, time.Millisecond)
	client.Write([]byte("+OK\r\n"))
	got, err := conn.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if got != "+OK\r\n" {
		t.Errorf("received: %#v", got)
	}
}

func TestClientConn_WriteError(t *testing.T) {
	conn := fakeConn{}
	client := NewClientConn(&conn, time.Millisecond)
	client.WriteError("oops")
	got, err := conn.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if got != "-oops\r\n" {
		t.Errorf("received: %#v", got)
	}
}
