package manifest_test

import (
	"math"
	"os/exec"
	"testing"

	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/assert"
)

type ExecResponse struct {
	Output []byte
	Error  error
}

type TestExecer struct {
	CannedResponses []ExecResponse
	Index           int
	Commands        []*exec.Cmd
}

func NewTestExecer() *TestExecer {
	return &TestExecer{
		CannedResponses: []ExecResponse{},
		Commands:        make([]*exec.Cmd, 0),
	}
}

func (p *TestExecer) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	i := int(math.Mod(float64(p.Index), float64(len(p.CannedResponses))))
	resp := p.CannedResponses[i]
	p.Index++
	return resp.Output, resp.Error
}

func (p *TestExecer) Run(s manifest.Stream, cmd *exec.Cmd) error {
	p.Commands = append(p.Commands, cmd)
	return nil
}

func (p *TestExecer) RunAsync(s manifest.Stream, cmd *exec.Cmd, done chan error) {
	p.Run(s, cmd)
	done <- nil
}

func TestBuildWithCache(t *testing.T) {
	output := manifest.NewOutput()
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		ExecResponse{
			Output: []byte("dockerid"),
			Error:  nil,
		},
	}

	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	if err != nil {
		t.Error(err)
	}

	err = m.Build(".", "web", str, true)

	cmd1 := []string{"docker", "build", "-f", "./Dockerfile.dev", "-t", "web/web", "."}
	cmd2 := []string{"docker", "tag", "convox/postgres", "web/database"}

	assert.Equal(t, len(te.Commands), 2)
	assert.Equal(t, te.Commands[0].Args, cmd1)
	assert.Equal(t, te.Commands[1].Args, cmd2)
}

func TestBuildCacheNoImage(t *testing.T) {
	output := manifest.NewOutput()
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		ExecResponse{
			Output: []byte(""),
			Error:  nil,
		},
	}

	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	if err != nil {
		t.Error(err)
	}

	err = m.Build(".", "web", str, true)

	cmd1 := []string{"docker", "build", "-f", "./Dockerfile.dev", "-t", "web/web", "."}
	cmd2 := []string{"docker", "pull", "convox/postgres"}
	cmd3 := []string{"docker", "tag", "convox/postgres", "web/database"}

	assert.Equal(t, len(te.Commands), 3)
	assert.Equal(t, te.Commands[0].Args, cmd1)
	assert.Equal(t, te.Commands[1].Args, cmd2)
	assert.Equal(t, te.Commands[2].Args, cmd3)
}

func TestBuildNoCache(t *testing.T) {
	output := manifest.NewOutput()
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		ExecResponse{
			Output: []byte("dockeid"),
			Error:  nil,
		},
	}

	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	if err != nil {
		t.Error(err)
	}

	err = m.Build(".", "web", str, false)

	cmd1 := []string{"docker", "build", "--no-cache", "-f", "./Dockerfile.dev", "-t", "web/web", "."}
	cmd2 := []string{"docker", "pull", "convox/postgres"}
	cmd3 := []string{"docker", "tag", "convox/postgres", "web/database"}

	assert.Equal(t, len(te.Commands), 3)
	assert.Equal(t, te.Commands[0].Args, cmd1)
	assert.Equal(t, te.Commands[1].Args, cmd2)
	assert.Equal(t, te.Commands[2].Args, cmd3)
}

func TestPush(t *testing.T) {
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

	cmd1 := []string{"docker", "tag", "app/database", "registry/flatten:database.tag"}
	cmd2 := []string{"docker", "push", "registry/flatten:database.tag"}
	cmd3 := []string{"docker", "tag", "app/web", "registry/flatten:web.tag"}
	cmd4 := []string{"docker", "push", "registry/flatten:web.tag"}
	m.Push(str, "app", "registry", "tag", "flatten")

	assert.Equal(t, len(te.Commands), 4)
	assert.Equal(t, te.Commands[0].Args, cmd1)
	assert.Equal(t, te.Commands[1].Args, cmd2)
	assert.Equal(t, te.Commands[2].Args, cmd3)
	assert.Equal(t, te.Commands[3].Args, cmd4)
}
