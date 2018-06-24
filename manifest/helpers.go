package manifest

import (
	"math/rand"
	"regexp"
)

func coalesce(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

var regexpInterpolation = regexp.MustCompile(`\$\{([^}]*?)\}`)

func interpolate(data []byte, env map[string]string) ([]byte, error) {
	p := regexpInterpolation.ReplaceAllFunc(data, func(m []byte) []byte {
		return []byte(env[string(m)[2:len(m)-1]])
	})

	return p, nil
}

var randomAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

func randomString(size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return string(b)
}
