package stdcli

import "fmt"

//Error returns an error with custom formatting for output via the cli package writer
func Error(err error) error {
	return fmt.Errorf(renderTags("<error>%s</error>"), err)
}
