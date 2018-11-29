package helpers

import "io"

type nopReader struct {
	io.Writer
}

func (nr nopReader) Read([]byte) (int, error) {
	return 0, nil
}

func NopReader(w io.Writer) io.ReadWriter {
	return nopReader{w}
}
