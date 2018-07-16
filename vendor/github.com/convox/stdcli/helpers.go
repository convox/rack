package stdcli

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func interfaceSlice(ss []string) []interface{} {
	is := make([]interface{}, len(ss))

	for i, s := range ss {
		is[i] = s
	}

	return is
}

func IsTerminal(f *os.File) bool {
	return terminal.IsTerminal(int(f.Fd()))
}
