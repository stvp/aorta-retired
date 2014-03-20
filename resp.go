package aorta

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// readRESPLine returns one full RESP line. The smallest RESP line is an empty
// bulk string ("\r\n").
func readRESPLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		return nil, fmt.Errorf("invalid response: line is too long")
	}
	if err != nil {
		return nil, err
	}

	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, fmt.Errorf("invalid response: bad line terminator")
	}

	return line, nil
}

// parseRESPLen takes a RESP array or bulk string length line and returns the
// expected length of the array or bulk string.
func parseRESPLen(line []byte) (int, error) {
	if len(line) < 4 {
		return 0, fmt.Errorf("invalid response: bad length line")
	}

	if line[0] != '$' && line[0] != '*' {
		return 0, fmt.Errorf("invalid response: bad length prefix")
	}

	// Shortcut for null bulk strings
	if len(line) == 4 && line[1] == '-' && line[2] == '1' {
		return -1, nil
	}

	var n int
	for _, b := range line[1 : len(line)-2] {
		n *= 10
		if b < '0' || b > '9' {
			return -1, fmt.Errorf("invalid response: bad length characters")
		}
		n += int(b - '0')
	}

	return n, nil
}

// readRESP returns one full valid RESP object from the given bufio.Reader.
func readRESP(r *bufio.Reader) ([]byte, error) {
	line, err := readRESPLine(r)
	if err != nil {
		return nil, err
	}

	buf := bytes.Buffer{}
	buf.Write(line)

	switch line[0] {
	case '$':
		length, err := parseRESPLen(line)
		if err != nil {
			return nil, err
		}
		if length < 0 {
			break // null bulk string
		}

		_, err = io.CopyN(&buf, r, int64(length)+2)
		if err != nil {
			return nil, err
		}
	// Array
	case '*':
		length, err := parseRESPLen(line)
		if err != nil {
			return nil, err
		}
		for i := 0; i < length; i++ {
			subResp, err := readRESP(r)
			if err != nil {
				return nil, err
			}
			buf.Write(subResp)
		}
	}

	return buf.Bytes(), nil
}
