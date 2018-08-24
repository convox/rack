package structs

import "math/rand"

var alphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func id(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return prefix + string(b)
}
