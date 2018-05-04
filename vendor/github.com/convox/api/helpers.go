package api

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

func functionName(fn interface{}) string {
	sig := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	parts := strings.Split(sig, ".")
	name := parts[len(parts)-1]
	name = strings.TrimSuffix(name, "-fm")
	return name
}

func generateId(length int) (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	key := fmt.Sprintf("%x", sha1.Sum(data))

	if len(key) < length {
		return "", fmt.Errorf("key too long")
	}

	return key[0:length], nil
}
