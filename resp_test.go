package aorta

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"
)

type respTest struct {
	given         []byte
	expected      []byte
	errorExpected bool
}

type respLenTest struct {
	given         []byte
	expected      int
	errorExpected bool
}

func Test_readRESPLine(t *testing.T) {
	tests := []respTest{
		{[]byte{}, []byte{}, true},
		{[]byte("-OK"), []byte{}, true},
		{[]byte("-OK\r"), []byte{}, true},
		{[]byte("-OK\r\n"), []byte("-OK\r\n"), false},
		{[]byte("-OK\r\n..."), []byte("-OK\r\n"), false},
		{[]byte("-OK\r\n-ERR\r\n"), []byte("-OK\r\n"), false},
		{[]byte("*2\r\n-OK\r\n-OK\r\n"), []byte("*2\r\n"), false},
	}

	for i, test := range tests {
		reader := bufio.NewReader(bytes.NewReader(test.given))
		line, err := readRESPLine(reader)
		if test.errorExpected {
			if err == nil {
				t.Errorf("tests[%d]: expected an error but didn't get one", i)
			}
		} else {
			if err != nil {
				t.Errorf("tests[%d]: %s", i, err.Error())
			} else if !reflect.DeepEqual(test.expected, line) {
				t.Errorf("tests[%d]:\nexpected: %v\ngot: %v", i, test.expected, line)
			}
		}
	}
}

func Test_parseRESPLen(t *testing.T) {
	tests := []respLenTest{
		{[]byte{}, 0, true},
		{[]byte(""), 0, true},
		{[]byte("-OK\r\n"), 0, true},
		{[]byte("*0x2\r\n"), 0, true},
		{[]byte("*-19\r\n"), 19, true},
		{[]byte("*1\r\n"), 1, false},
		{[]byte("$987\r\n"), 987, false},
	}

	for i, test := range tests {
		size, err := parseRESPLen(test.given)
		if test.errorExpected {
			if err == nil {
				t.Errorf("tests[%d]: expected an error but didn't get one", i)
			}
		} else {
			if err != nil {
				t.Errorf("tests[%d]: %s", i, err.Error())
			} else if test.expected != size {
				t.Errorf("tests[%d]: expected: %v, got: %v", i, test.expected, size)
			}
		}
	}
}

func Test_readRESP(t *testing.T) {
	tests := []respTest{
		// empty
		{[]byte{}, []byte{}, true},
		// no delimiter
		{[]byte("-OK"), []byte{}, true},
		// invalid delimiter
		{[]byte("-OK\r"), []byte{}, true},
		// simple string
		{[]byte("-OK\r\n"), []byte("-OK\r\n"), false},
		// ignore trailing junk
		{[]byte("-OK\r\n..."), []byte("-OK\r\n"), false},
		// read only one full response
		{[]byte("-OK\r\n-ERR\r\n"), []byte("-OK\r\n"), false},
		// array
		{[]byte("*2\r\n-OK\r\n-OK\r\n"), []byte("*2\r\n-OK\r\n-OK\r\n"), false},
		// empty array
		{[]byte("*0\r\n"), []byte("*0\r\n"), false},
		// array with invalid length
		{[]byte("*5\r\n-OK\r\n"), []byte{}, true},
		// empty bulk string
		{[]byte("$0\r\n\r\n"), []byte("$0\r\n\r\n"), false},
		// bulk string
		{[]byte("$4\r\ncool\r\n"), []byte("$4\r\ncool\r\n"), false},
		// array of arrays
		{[]byte("*2\r\n*1\r\n-OK\r\n*1\r\nOK\r\n"), []byte("*2\r\n*1\r\n-OK\r\n*1\r\nOK\r\n"), false},
	}

	for i, test := range tests {
		reader := bufio.NewReader(bytes.NewReader(test.given))
		line, err := readRESP(reader)
		if test.errorExpected {
			if err == nil {
				t.Errorf("tests[%d]: expected an error but didn't get one", i)
			}
		} else {
			if err != nil {
				t.Errorf("tests[%d]: %s", i, err.Error())
			} else if !reflect.DeepEqual(test.expected, line) {
				t.Errorf("tests[%d]:\nexpected: %v\ngot: %v", i, test.expected, line)
			}
		}
	}
}
