package manifest

import (
	"regexp"
)

var (
	regexpInterpolation = regexp.MustCompile(`\$\{([^}]*?)\}`)
)

type Environment map[string]string

func interpolate(data []byte, env Environment) ([]byte, error) {
	p := regexpInterpolation.ReplaceAllFunc(data, func(m []byte) []byte {
		return []byte(env[string(m)[2:len(m)-1]])
	})

	return p, nil
}
