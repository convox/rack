// +build !lambdabinary

package sparta

import "github.com/Sirupsen/logrus"

// Support Windows development, by only requiring `syscall` in the compiled
// linux binary.
func platformKill(parentProcessPID int) {
	// NOP
}

func platformLogSysInfo(logger *logrus.Logger) {
	// NOP
}
