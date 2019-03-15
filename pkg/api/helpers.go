package api

import (
	"fmt"
	"io"
)

func renderStatusCode(w io.Writer, code int) error {
	_, err := fmt.Fprintf(w, "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:%d\n", code)
	return err
}
