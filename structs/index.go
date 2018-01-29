package structs

import (
	"os"
	"time"
)

type Index map[string]IndexItem

type IndexItem struct {
	Name    string      `json:"name"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"mtime"`
}
