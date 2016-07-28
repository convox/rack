package manifest_test

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/assert"
)

func TestNewProcess(t *testing.T) {
	s := manifest.Service{
		Name: "foo",
		Volumes: []string{
			"/data/data",
			"/foo:/data/data",
		},
	}

	m := manifest.Manifest{
		Services: map[string]manifest.Service{
			"foo": s,
		},
	}

	p := manifest.NewProcess("api", s, m)

	usr, _ := user.Current()
	dir := usr.HomeDir

	expectedArgs := []string{
		"-i",
		"--rm",
		"--name",
		"api-foo",
		"-v",
		fmt.Sprintf("%s/.convox/volumes/api/foo/data/data:/data/data", dir),
		"-v",
		"/foo:/data/data",
		"api/foo",
	}

	assert.Equal(t, p.Name, "api-foo")
	assert.Equal(t, p.Args, expectedArgs)
}
