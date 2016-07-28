package manifest_test

import (
	"os/exec"
	"testing"

	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/assert"
)

type TestExecer struct {
	Commands []*exec.Cmd
}

func NewTestExecer() *TestExecer {
	return &TestExecer{
		Commands: make([]*exec.Cmd, 0),
	}
}

func (p *TestExecer) Run(s manifest.Stream, cmd *exec.Cmd) error {
	p.Commands = append(p.Commands, cmd)
	return nil
}

func (p *TestExecer) RunAsync(s manifest.Stream, cmd *exec.Cmd, done chan error) {
	p.Run(s, cmd)
	done <- nil
}

func TestBuild(t *testing.T) {
	output := manifest.NewOutput()
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	if err != nil {
		t.Error(err)
	}

	err = m.Build(".", "web", str, true)

	cmd1 := []string{"docker", "build", "--no-cache", "-f", "./Dockerfile.dev", "-t", "web/web", "."}
	cmd2 := []string{"docker", "pull", "convox/postgres"}
	cmd3 := []string{"docker", "tag", "convox/postgres", "web/database"}

	assert.Equal(t, len(te.Commands), 3)
	assert.Equal(t, te.Commands[0].Args, cmd1)
	assert.Equal(t, te.Commands[1].Args, cmd2)
	assert.Equal(t, te.Commands[2].Args, cmd3)
}
