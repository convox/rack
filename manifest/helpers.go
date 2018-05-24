package manifest

import "regexp"

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
