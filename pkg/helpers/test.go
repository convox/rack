package helpers

import (
	"fmt"
	"io/ioutil"
)

func Testdata(name string) ([]byte, error) {
	return ioutil.ReadFile(fmt.Sprintf("testdata/%s.yml", name))
}
