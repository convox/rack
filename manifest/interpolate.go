package manifest

import (
	"fmt"
	"os"
	"regexp"
)

var (
	wordChar = regexp.MustCompile("[0-9A-Za-z_]")
)

type Token struct {
	Value []byte
	Kind  string
}

func (t Token) Result() string {
	switch t.Kind {
	case "env":
		return os.Getenv(string(t.Value))
	case "ignore":
		return fmt.Sprintf("$%s", t.Value)
	default:
		return string(t.Value)
	}
}

func parseLine(line string) string {
	tokens := []Token{}
	totalLength := len(line)

	for i := 0; i < totalLength; {
		char := line[i]

		if char == '$' && line[i+1] == '$' {
			tok := Token{
				Kind:  "ignore",
				Value: []byte{},
			}

			//double dollar ignore
			for x := i + 2; x < totalLength; {
				if wordChar.Match([]byte{line[x]}) {
					tok.Value = append(tok.Value, line[x])
					i = x
				} else {
					break
				}
				x++
				i = x
			}
			tokens = append(tokens, tok)
		} else if char == '$' && line[i+1] == '{' {
			//bracket var
			i += 2
			tok := Token{
				Kind:  "defualt",
				Value: []byte{},
			}
			for x := i; x < totalLength; {
				if line[x] != '}' {
					tok.Value = append(tok.Value, line[x])
					if x == (totalLength-1) || !wordChar.Match([]byte{line[x]}) {
						tok.Value = []byte(fmt.Sprintf("${%s", tok.Value))
						break
					}
				} else {
					tok.Kind = "env"
					break
				}
				x++
				i++
			}
			i += 1
			tokens = append(tokens, tok)
		} else if char == '$' && wordChar.Match([]byte{line[i+1]}) {
			//dollar var
			tok := Token{
				Kind:  "env",
				Value: []byte{},
			}
			i += 1
			for x := i; x < totalLength; {
				if wordChar.Match([]byte{line[x]}) {
					tok.Value = append(tok.Value, line[x])
				} else {
					break
				}
				x++
				i++
			}
			tokens = append(tokens, tok)
		} else {
			tokens = append(tokens, Token{
				Value: []byte{char},
				Kind:  "default",
			})
			i++
		}
	}

	str := ""
	for _, t := range tokens {
		str = fmt.Sprintf("%s%s", str, t.Result())
	}

	return str
}
