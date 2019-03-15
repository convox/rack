package stdcli

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

var (
	DefaultReader *Reader
)

type Reader struct {
	io.Reader
}

func init() {
	DefaultReader = &Reader{os.Stdin}
}

func (r *Reader) IsTerminal() bool {
	if f, ok := r.Reader.(*os.File); ok {
		return IsTerminal(f)
	}
	return false
}

func (r *Reader) TerminalRaw() func() bool {
	var fd int
	var state *terminal.State

	if f, ok := r.Reader.(*os.File); ok {
		fd = int(f.Fd())
		if s, err := terminal.MakeRaw(fd); err == nil {
			state = s
		}
	}

	return func() bool {
		if state != nil {
			terminal.Restore(fd, state)
			return true
		}
		return false
	}
}
