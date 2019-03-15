package stdapi

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

func coalesce(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

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
		return "", errors.WithStack(err)
	}

	key := fmt.Sprintf("%x", sha1.Sum(data))

	if len(key) < length {
		return "", errors.WithStack(fmt.Errorf("key too long"))
	}

	return key[0:length], nil
}
