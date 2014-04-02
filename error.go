package main

import (
	"errors"
	"io"
	"net"
)

var (
	ErrConnClosed           = errors.New("aorta: connection closed")
	ErrTimeout              = errors.New("aorta: timeout")
	ErrInvalidCommandFormat = errors.New("aorta: invalid command format")
)

func wrapErr(err error) error {
	if err == nil {
		return nil
	} else if err == io.EOF || err == io.ErrUnexpectedEOF {
		return ErrConnClosed
	} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return ErrTimeout
	} else if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
		return ErrTimeout
	} else {
		return err
	}
}
