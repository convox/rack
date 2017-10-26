package structs

import (
	"math/rand"
	"time"
)

type Build struct {
	Id          string `json:"id"`
	App         string `json:"app"`
	Description string `json:"description"`
	Logs        string `json:"logs"`
	Manifest    string `json:"manifest"`
	Release     string `json:"release"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`

	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended"`

	Tags map[string]string `json:"tags"`
}

type Builds []Build

type BuildOptions struct {
	Cache       bool
	Config      string
	Description string
}

func NewBuild(app string) *Build {
	return &Build{
		App:    app,
		Id:     generateId("B", 10),
		Status: "created",
		Tags:   map[string]string{},
	}
}

var idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}
	return prefix + string(b)
}
