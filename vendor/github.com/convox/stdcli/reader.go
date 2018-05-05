package stdcli

import "os"

var (
	DefaultReader *Reader
)

type Reader struct {
	*os.File
}

func init() {
	DefaultReader = &Reader{File: os.Stdin}
}

func (r *Reader) Fd() uintptr {
	return r.File.Fd()
}
