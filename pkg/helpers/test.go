package helpers

import (
	"fmt"
	"os"
)

func Testdata(name string) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("testdata/%s.yml", name))
}
