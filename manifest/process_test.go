package manifest_test

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/assert"
)

func TestProcessNew(t *testing.T) {
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

func TestProcessGenerateArgs(t *testing.T) {
	s := manifest.Service{
		Name: "foo",
	}

	m := manifest.Manifest{
		Services: map[string]manifest.Service{
			"foo": s,
		},
	}

	p := manifest.NewProcess("api", s, m)

	expectedArgs := []string{
		"-i",
		"--rm",
		"--name",
		"api-foo",
		"api/foo",
	}

	assert.Equal(t, p.Name, "api-foo")
	assert.Equal(t, p.Args, expectedArgs)

	p.Args = p.GenerateArgs(&manifest.ArgOptions{
		Command:     "foobar",
		IgnorePorts: true,
		Name:        "fake-name",
	})

	assert.Equal(t, []string{"-i", "--rm", "--name", "fake-name", "api/foo", "sh", "-c", "foobar"}, p.Args)

	p.Args = p.GenerateArgs(&manifest.ArgOptions{
		Command: "newcommand",
	})

	assert.Equal(t, []string{"-i", "--rm", "--name", "api-foo", "api/foo", "sh", "-c", "newcommand"}, p.Args)

}

func TestProcessCommandString(t *testing.T) {
	s := manifest.Service{
		Name:    "foo",
		Command: manifest.Command{String: "ls -la"},
	}

	m := manifest.Manifest{
		Services: map[string]manifest.Service{
			"foo": s,
		},
	}

	p := manifest.NewProcess("api", s, m)

	if assert.NotNil(t, p) {
		assert.Equal(t, []string{"-i", "--rm", "--name", "api-foo", "api/foo", "sh", "-c", "ls -la"}, p.Args)
	}
}

func TestProcessStringArray(t *testing.T) {
	s := manifest.Service{
		Name:    "foo",
		Command: manifest.Command{Array: []string{"ls", "-la"}},
	}

	m := manifest.Manifest{
		Services: map[string]manifest.Service{
			"foo": s,
		},
	}

	p := manifest.NewProcess("api", s, m)

	if assert.NotNil(t, p) {
		assert.Equal(t, []string{"-i", "--rm", "--name", "api-foo", "api/foo", "ls", "-la"}, p.Args)
	}
}
