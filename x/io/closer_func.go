package xio

import "io"

type CloserFunc func() error

func (c CloserFunc) Close() error {
	return c()
}

type ReaderCloser struct {
	io.Reader
	io.Closer
}
