package stdcli

// ErrorStdCli represents a generic stdcli error
type ErrorStdCli string

// Error satisfies the error interface
func (e ErrorStdCli) Error() string {
	return string(e)
}
