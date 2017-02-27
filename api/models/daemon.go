package models

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"regexp"
	"strings"

	"github.com/convox/rack/manifest"
)

type Daemon struct {
	Service *manifest.Service
	App     *App
}

func (d *Daemon) ShortName() string {
	shortName := strings.Title(d.Service.Name)

	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		panic(err)
	}

	return reg.ReplaceAllString(shortName, "")
}

func (d *Daemon) LongName() string {
	prefix := fmt.Sprintf("%s-%s", d.App.StackName(), d.Service.Name)
	hash := sha256.Sum256([]byte(prefix))
	suffix := "-" + base32.StdEncoding.EncodeToString(hash[:])[:7]

	// $prefix-$suffix-schedule" needs to be <= 64 characters
	if len(prefix) > 55-len(suffix) {
		prefix = prefix[:55-len(suffix)]
	}
	return prefix + suffix
}

func (a App) Daemons(m manifest.Manifest) []Daemon {
	daemons := []Daemon{}

	for _, entry := range m.Services {
		labels := entry.LabelsByPrefix("convox.daemon")

		if len(labels) == 1 {
			d := Daemon{
				Service: &entry,
				App:     &a,
			}
			daemons = append(daemons, d)
		}
	}
	return daemons
}
