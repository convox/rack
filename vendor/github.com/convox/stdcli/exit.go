package stdcli

import "fmt"

type ExitCoder interface {
	Code() int
}

type ExitCode struct {
	code int
}

func (e ExitCode) Code() int {
	return e.code
}

func (e ExitCode) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}

func Exit(code int) ExitCode {
	return ExitCode{code: code}
}
