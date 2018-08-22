package k8s

import "strings"

type Storage interface {
	Clear(prefix string) error
	Load(key string, v interface{}) error
	List(prefix string) ([]string, error)
	Read(key string) ([]byte, error)
	Write(key string, v interface{}) error
}

func storageNotExists(err error) bool {
	if err == nil {
		return false
	}

	if strings.HasPrefix(err.Error(), "no such key:") {
		return true
	}

	return false
}
