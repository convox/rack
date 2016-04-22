package structs

import (
	"strings"
	"time"
)

type Certificate struct {
	Id         string    `json:"id"`
	Domain     string    `json:"domain"`
	Expiration time.Time `json:"expiration"`
}

type Certificates []Certificate

func (c Certificates) Len() int           { return len(c) }
func (c Certificates) Less(i, j int) bool { return strings.ToUpper(c[i].Id) < strings.ToUpper(c[j].Id) }
func (c Certificates) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
