// +build  !lambdabinary

package sparta

// Support Windows development, by only requiring `syscall` in the compiled
// linux binary.
func platformKill(parentProcessPID int) {
	// NOP
}
