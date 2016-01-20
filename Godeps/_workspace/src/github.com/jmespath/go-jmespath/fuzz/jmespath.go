package jmespath

import "github.com/convox/rack/Godeps/_workspace/src/github.com/jmespath/go-jmespath"

// Fuzz will fuzz test the JMESPath parser.
func Fuzz(data []byte) int {
	p := jmespath.NewParser()
	_, err := p.Parse(string(data))
	if err != nil {
		return 1
	}
	return 0
}
