package helpers

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"
)

var (
	alphabet = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Id(prefix string, length int) string {
	id := prefix

	for i := 0; i < length-len(prefix); i++ {
		id += string(alphabet[rand.Intn(len(alphabet))])
	}

	return id
}

func RandomString(length int) (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	key := fmt.Sprintf("%x", sha256.Sum256(data))

	if len(key) < length {
		return "", fmt.Errorf("key too long")
	}

	return key[0:length], nil
}
