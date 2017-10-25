package types

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"
)

var (
	Alphabet = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Id(prefix string, length int) string {
	id := prefix

	for i := 0; i < length-len(prefix); i++ {
		id += string(Alphabet[rand.Intn(len(Alphabet))])
	}

	return id
}

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
