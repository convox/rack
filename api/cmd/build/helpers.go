package main

import (
	"crypto/rand"
	"math/big"
)

var (
	idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

func id(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(idAlphabet))))
		if err != nil {
			panic(err)
		}
		b[i] = idAlphabet[idx.Int64()]
	}
	return prefix + string(b)
}
