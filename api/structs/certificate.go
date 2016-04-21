package structs

import "time"

type Certificate struct {
	Id         string    `json:"id"`
	Domain     string    `json:"domain"`
	Expiration time.Time `json:"expiration"`
}

type Certificates []Certificate
