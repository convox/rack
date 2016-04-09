package composure

import (
	"fmt"
	"testing"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/docker/libcompose/docker"
	"github.com/docker/libcompose/project"
)

func TestNewProject(t *testing.T) {
	project, err := docker.NewProject(&docker.Context{
		Context: project.Context{
			ComposeFiles: []string{"fixtures/httpd.yml"},
			ProjectName:  "httpd",
		},
	})

	assert.Nil(t, err)

	fmt.Printf("PROJECT: %+v\n", project)
}
