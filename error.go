package main

import (
	"errors"
	"io"
	"net"
)

var (
	ErrConnClosed = errors.New("aorta: connection closed")
	ErrTimeout    = errors.New("aorta: timeout")
)

func wrapErr(err error) error {
	if err == nil {
		return nil
	} else if err == io.EOF || err == io.ErrUnexpectedEOF {
		return ErrConnClosed
	} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return ErrTimeout
	} else {
		return err
	}
}
