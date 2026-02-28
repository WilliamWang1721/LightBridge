package main

import (
	"bufio"
	"bytes"
	"io"
)

var dataPrefix = []byte("data:")

func newSSEScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	// SSE lines can be large for tool call arguments / long outputs.
	sc.Buffer(make([]byte, 0, 64*1024), 52_428_800) // ~50MB
	return sc
}

func hasDataPrefix(line []byte) bool {
	line = bytes.TrimSpace(line)
	return bytes.HasPrefix(line, dataPrefix)
}

func trimDataPrefix(line []byte) []byte {
	line = bytes.TrimSpace(line)
	if !bytes.HasPrefix(line, dataPrefix) {
		return nil
	}
	return bytes.TrimSpace(line[len(dataPrefix):])
}
