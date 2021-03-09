package utils

import (
	"bytes"
	"runtime"
	"strconv"
)

// GoID returns the current goroutine id
func GoID() int64 {
	buf := make([]byte, 64)
	i := runtime.Stack(buf, false)

	buf = bytes.TrimPrefix(buf[:i], []byte("goroutine "))
	i = bytes.IndexByte(buf, ' ')

	n, err := strconv.ParseInt(string(buf[:i]), 10, 64)
	if err != nil {
		panic(err)
	}

	return n
}
