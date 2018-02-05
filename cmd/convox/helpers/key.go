package helpers

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

func Key(length int) (string, error) {
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
