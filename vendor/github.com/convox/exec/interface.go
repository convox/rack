package exec

import "io"

type Interface interface {
	Execute(string, ...string) ([]byte, error)
	Run(io.Writer, string, ...string) error
	Stream(io.Writer, io.Reader, string, ...string) error
	Terminal(string, ...string) error
}
