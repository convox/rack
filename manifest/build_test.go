package manifest_test

import (
	"fmt"
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

type TestCommands [][]string

func NewTestExecer() *TestExecer {
	return &TestExecer{
		CannedResponses: []ExecResponse{},
		Commands:        make([]*exec.Cmd, 0),
	}
}

func (te *TestExecer) AssertCommands(t *testing.T, commands TestCommands) {
	assert.Equal(t, len(te.Commands), len(commands))

	max := int(math.Max(float64(len(te.Commands)), float64(len(commands))))

	for i := 0; i < max; i++ {
		expected := []string{}
		actual := []string{}

		if i < len(te.Commands) {
			expected = te.Commands[i].Args
		}

		if i < len(commands) {
			actual = commands[i]
		}

		assert.Equal(t, expected, actual)
	}
}

func (p *TestExecer) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	if p.Index > len(p.CannedResponses)-1 {
		return nil, fmt.Errorf("CannedResponse index out of range")
	}
	resp := p.CannedResponses[p.Index]
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
	output := manifest.NewOutput(true)
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte("dockerid"),
			Error:  nil,
		},
	}

	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest.BuildOptions{
		Cache: true,
		Environment: map[string]string{
			"FOO": "bar",
		},
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "build", "--build-arg", "FOO=\"bar\"", "-f", "fixtures/Dockerfile.dev", "-t", "web/web", "fixtures"}
	cmd2 := []string{"docker", "tag", "convox/postgres:latest", "web/database"}

	if assert.Equal(t, len(te.Commands), 2) {
		assert.Equal(t, te.Commands[0].Args, cmd1)
		assert.Equal(t, te.Commands[1].Args, cmd2)
	}
}

func TestBuildCacheNoImage(t *testing.T) {
	output := manifest.NewOutput(true)
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte(""),
			Error:  nil,
		},
	}

	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest.BuildOptions{
		Cache: true,
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "build", "-f", "fixtures/Dockerfile.dev", "-t", "web/web", "fixtures"}
	cmd2 := []string{"docker", "pull", "convox/postgres:latest"}
	cmd3 := []string{"docker", "tag", "convox/postgres:latest", "web/database"}

	if assert.Equal(t, len(te.Commands), 3) {
		assert.Equal(t, te.Commands[0].Args, cmd1)
		assert.Equal(t, te.Commands[1].Args, cmd2)
		assert.Equal(t, te.Commands[2].Args, cmd3)
	}
}

func TestBuildWithSpecificService(t *testing.T) {
	output := manifest.NewOutput(true)
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte("dockerid"),
			Error:  nil,
		},
	}

	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("specific-service")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest.BuildOptions{
		Service: "web",
		Cache:   true,
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "build", "-f", "fixtures/Dockerfile.dev", "-t", "web/web", "fixtures"}
	cmd2 := []string{"docker", "tag", "convox/postgres:latest", "web/database"}

	if assert.Equal(t, len(te.Commands), 2) {
		assert.Equal(t, te.Commands[0].Args, cmd1)
		assert.Equal(t, te.Commands[1].Args, cmd2)
	}
}

func TestBuildNoCache(t *testing.T) {
	output := manifest.NewOutput(true)
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte("dockeid"),
			Error:  nil,
		},
	}

	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest.BuildOptions{
		Service: "web",
		Cache:   false,
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile.dev", "-t", "web/web", "fixtures"}
	cmd2 := []string{"docker", "pull", "convox/postgres:latest"}
	cmd3 := []string{"docker", "tag", "convox/postgres:latest", "web/database"}

	if assert.Equal(t, len(te.Commands), 3) {
		assert.Equal(t, te.Commands[0].Args, cmd1)
		assert.Equal(t, te.Commands[1].Args, cmd2)
		assert.Equal(t, te.Commands[2].Args, cmd3)
	}
}

func TestBuildRepeatSimple(t *testing.T) {
	output := manifest.NewOutput(true)
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("repeat-simple")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest.BuildOptions{
		Cache: false,
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile", "-t", "web/monitor", "fixtures"}
	cmd2 := []string{"docker", "build", "--no-cache", "-f", "fixtures/other/Dockerfile", "-t", "web/other", "fixtures/other"}
	cmd3 := []string{"docker", "tag", "web/monitor", "web/web"}

	if assert.Equal(t, len(te.Commands), 3) {
		assert.Equal(t, te.Commands[0].Args, cmd1)
		assert.Equal(t, te.Commands[1].Args, cmd2)
		assert.Equal(t, te.Commands[2].Args, cmd3)
	}
}

func TestBuildRepeatImage(t *testing.T) {
	output := manifest.NewOutput(true)
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte(""),
			Error:  nil,
		},
	}
	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("repeat-image")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest.BuildOptions{
		Cache: false,
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "pull", "convox/rails:latest"}
	cmd2 := []string{"docker", "tag", "convox/rails:latest", "web/web1"}
	cmd3 := []string{"docker", "tag", "convox/rails:latest", "web/web2"}

	if assert.Equal(t, len(te.Commands), 3) {
		assert.Equal(t, te.Commands[0].Args, cmd1)
		assert.Equal(t, te.Commands[1].Args, cmd2)
		assert.Equal(t, te.Commands[2].Args, cmd3)
	}
}

func TestBuildRepeatComplex(t *testing.T) {
	output := manifest.NewOutput(true)
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("repeat-complex")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest.BuildOptions{
		Cache: false,
		Environment: map[string]string{
			"foo": "baz",
		},
	})
	assert.NoError(t, err)

	te.AssertCommands(t, TestCommands{
		[]string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile", "-t", "web/first", "fixtures"},
		[]string{"docker", "build", "--no-cache", "--build-arg", "foo=\"bar\"", "-f", "fixtures/Dockerfile", "-t", "web/monitor", "fixtures"},
		[]string{"docker", "build", "--no-cache", "--build-arg", "foo=\"bar\"", "-f", "fixtures/other/Dockerfile", "-t", "web/othera", "fixtures/other"},
		[]string{"docker", "build", "--no-cache", "--build-arg", "foo=\"baz\"", "-f", "fixtures/Dockerfile.other", "-t", "web/otherb", "fixtures"},
		[]string{"docker", "build", "--no-cache", "--build-arg", "foo=\"other\"", "-f", "fixtures/Dockerfile", "-t", "web/otherc", "fixtures"},
		[]string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile", "-t", "web/otherd", "fixtures"},
		[]string{"docker", "tag", "web/first", "web/othere"},
		[]string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile.otherf", "-t", "web/otherf", "fixtures"},
		[]string{"docker", "tag", "web/otherf", "web/otherg"},
		[]string{"docker", "tag", "web/monitor", "web/web"},
	})
}

func TestDoubleDockerfile(t *testing.T) {
	m, err := manifestFixture("double-dockerfile")

	assert.Nil(t, m, "manifest should be nil")
	assert.Equal(t, fmt.Errorf("dockerfile specified twice for web"), err)
}

func TestPush(t *testing.T) {
	output := manifest.NewOutput(true)
	str := output.Stream("build")
	dr := manifest.DefaultRunner
	te := NewTestExecer()
	manifest.DefaultRunner = te
	defer func() { manifest.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	assert.NoError(t, err)

	cmd1 := []string{"docker", "tag", "app/database", "registry/test:database.tag"}
	cmd2 := []string{"docker", "push", "registry/test:database.tag"}
	cmd3 := []string{"docker", "tag", "app/web", "registry/test:web.tag"}
	cmd4 := []string{"docker", "push", "registry/test:web.tag"}

	m.Push("registry/test:{service}.{build}", "app", "tag", str)

	assert.Equal(t, len(te.Commands), 4)
	assert.Equal(t, te.Commands[0].Args, cmd1)
	assert.Equal(t, te.Commands[1].Args, cmd2)
	assert.Equal(t, te.Commands[2].Args, cmd3)
	assert.Equal(t, te.Commands[3].Args, cmd4)
}
