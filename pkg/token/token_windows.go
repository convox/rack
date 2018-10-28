package token

import "fmt"

func Authenticate(req []byte) ([]byte, error) {
	return nil, fmt.Errorf("no u2f support on windows")
}
