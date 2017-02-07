package models

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"regexp"
	"strings"

	"github.com/convox/rack/manifest"
)

// Agent represents a Service which runs exactly once on every ECS agent
type Agent struct {
	Service *manifest.Service
	App     *App
}

var shortNameRegex = regexp.MustCompile("[^A-Za-z0-9]+")

// ShortName returns the name of the Agent Service, sans any invalid characters
func (d *Agent) ShortName() string {
	shortName := strings.Title(d.Service.Name)
	return shortNameRegex.ReplaceAllString(shortName, "")
}

// LongName returns the name of the Agent Service in [stack name]-[service name]-[hash] format
func (d *Agent) LongName() string {
	prefix := fmt.Sprintf("%s-%s", d.App.StackName(), d.Service.Name)
	hash := sha256.Sum256([]byte(prefix))
	suffix := "-" + base32.StdEncoding.EncodeToString(hash[:])[:7]

	// $prefix-$suffix-change" needs to be <= 64 characters
	if len(prefix) > 57-len(suffix) {
		prefix = prefix[:57-len(suffix)]
	}
	return prefix + suffix
}

// Agents returns any Agent Services defined in the given Manifest
func (a App) Agents(m manifest.Manifest) []Agent {
	agents := []Agent{}

	for _, entry := range m.Services {
		labels := entry.LabelsByPrefix("convox.agent")

		if len(labels) == 1 {
			d := Agent{
				Service: &entry,
				App:     &a,
			}
			agents = append(agents, d)
		}
	}
	return agents
}
