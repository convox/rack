package stdcli

import (
	"syscall"

	"github.com/Azure/go-ansiterm/winterm"
)

const enableVirtualTerminalProcessing = 0x0004

func terminalSetup() error {
	hnd, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return err
	}

	var mode uint32

	if err := syscall.GetConsoleMode(hnd, &mode); err != nil {
		return err
	}

	if err := winterm.SetConsoleMode(uintptr(hnd), mode|enableVirtualTerminalProcessing); err != nil {
		return err
	}

	return nil
}
