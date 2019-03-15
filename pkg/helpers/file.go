package helpers

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func FileExists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func WriteFile(filename string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, data, mode); err != nil {
		return err
	}

	return nil
}
