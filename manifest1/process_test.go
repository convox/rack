package manifest1_test

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/convox/rack/manifest1"
	"github.com/stretchr/testify/assert"
)

func TestProcessNew(t *testing.T) {
	s := manifest1.Service{
		Name: "foo",
		Volumes: []string{
			"/data/data",
			"/foo:/data/data",
		},
	}

	m := manifest1.Manifest{
		Services: map[string]manifest1.Service{
			"foo": s,
		},
	}

	p := manifest1.NewProcess("api", s, m)

	usr, _ := user.Current()
	dir := usr.HomeDir

	expectedArgs := []string{
		"-i",
		"--rm",
		"--name",
		"api-foo",
		"-v",
		fmt.Sprintf("%s/.convox/volumes/api/foo//data/data:/data/data", dir),
		"-v",
		"/foo:/data/data",
		"api/foo",
	}

	assert.Equal(t, p.Name, "api-foo")
	assert.Equal(t, p.Args, expectedArgs)
}

func TestProcessGenerateArgs(t *testing.T) {
	s := manifest1.Service{
		Name: "foo",
	}

	m := manifest1.Manifest{
		Services: map[string]manifest1.Service{
			"foo": s,
		},
	}

	p := manifest1.NewProcess("api", s, m)

	expectedArgs := []string{
		"-i",
		"--rm",
		"--name",
		"api-foo",
		"api/foo",
	}

	assert.Equal(t, p.Name, "api-foo")
	assert.Equal(t, p.Args, expectedArgs)

	args, err := p.GenerateArgs(&manifest1.ArgOptions{
		Command:     "foobar",
		IgnorePorts: true,
		Name:        "fake-name",
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"-i", "--rm", "--name", "fake-name", "api/foo", "sh", "-c", "foobar"}, args)

	args, err = p.GenerateArgs(&manifest1.ArgOptions{
		Command: "newcommand",
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"-i", "--rm", "--name", "api-foo", "api/foo", "sh", "-c", "newcommand"}, args)
}

func TestProcessCommandString(t *testing.T) {
	s := manifest1.Service{
		Name:    "foo",
		Command: manifest1.Command{String: "ls -la"},
	}

	m := manifest1.Manifest{
		Services: map[string]manifest1.Service{
			"foo": s,
		},
	}

	p := manifest1.NewProcess("api", s, m)

	if assert.NotNil(t, p) {
		assert.Equal(t, []string{"-i", "--rm", "--name", "api-foo", "api/foo", "sh", "-c", "ls -la"}, p.Args)
	}
}

func TestProcessStringArray(t *testing.T) {
	s := manifest1.Service{
		Name:    "foo",
		Command: manifest1.Command{Array: []string{"ls", "-la"}},
	}

	m := manifest1.Manifest{
		Services: map[string]manifest1.Service{
			"foo": s,
		},
	}

	p := manifest1.NewProcess("api", s, m)

	if assert.NotNil(t, p) {
		assert.Equal(t, []string{"-i", "--rm", "--name", "api-foo", "api/foo", "ls", "-la"}, p.Args)
	}
}
