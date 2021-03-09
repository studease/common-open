package utils

import (
	"io"
	"os"
)

// Create creates the named file with mode 0666 (before umask), truncating it if it already exists
func Create(path string) (*os.File, error) {
	err := MkdirAll(path)
	if err != nil {
		return nil, err
	}

	return os.Create(path)
}

// OpenFile opens the named file with mode 0666 (before umask) for reading & writing, without truncating
func OpenFile(path string, flag int) (*os.File, error) {
	err := MkdirAll(path)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(path, flag, 0666)
}

// MkdirAll creates a directory named path, along with any necessary parents, and returns nil, or else returns an error
func MkdirAll(path string) error {
	i := len(path)

	for i > 0 && !os.IsPathSeparator(path[i-1]) {
		i--
	}

	if i > 0 {
		err := os.MkdirAll(path[:i-1], os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// CopyFile copies the src file to the dst path
func CopyFile(dst string, src string) (written int64, err error) {
	s, err := os.Open(src)
	if err != nil {
		return
	}

	defer s.Close()

	d, err := Create(dst)
	if err != nil {
		return
	}

	defer d.Close()

	return io.Copy(d, s)
}
