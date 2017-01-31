package stdcli

import "fmt"

func Error(err error) error {
	return fmt.Errorf(renderTags("<error>%s</error>"), err)
}
