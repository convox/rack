package manifest1_test

import (
	"fmt"
	"math"
	"os/exec"
	"testing"

	"github.com/convox/rack/pkg/manifest1"
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
			actual = te.Commands[i].Args
		}

		if i < len(commands) {
			expected = commands[i]
		}

		assert.Equal(t, expected, actual)
	}
}

// CombinedOutput test method
func (te *TestExecer) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	if te.Index > len(te.CannedResponses)-1 {
		return nil, fmt.Errorf("CannedResponse index out of range")
	}
	resp := te.CannedResponses[te.Index]
	te.Index++
	return resp.Output, resp.Error
}

// Run test method
func (te *TestExecer) Run(s manifest1.Stream, cmd *exec.Cmd, opts manifest1.RunnerOptions) error {
	te.Commands = append(te.Commands, cmd)
	return nil
}

// RunAsync test method
func (te *TestExecer) RunAsync(s manifest1.Stream, cmd *exec.Cmd, done chan error, opts manifest1.RunnerOptions) {
	te.Run(s, cmd, opts)
	done <- nil
}

func TestBuildWithCache(t *testing.T) {
	output := manifest1.NewOutput(true)
	str := output.Stream("build")
	dr := manifest1.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte("dockerid"),
			Error:  nil,
		},
	}

	manifest1.DefaultRunner = te
	defer func() { manifest1.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest1.BuildOptions{
		Cache: true,
		Environment: map[string]string{
			"FOO": "bar",
		},
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "build", "--build-arg", "FOO=bar", "-f", "fixtures/Dockerfile.dev", "-t", "web/web", "fixtures"}
	cmd2 := []string{"docker", "tag", "convox/postgres:latest", "web/database"}

	if assert.Equal(t, len(te.Commands), 2) {
		assert.Equal(t, te.Commands[0].Args, cmd1)
		assert.Equal(t, te.Commands[1].Args, cmd2)
	}
}

func TestBuildCacheNoImage(t *testing.T) {
	output := manifest1.NewOutput(true)
	str := output.Stream("build")
	dr := manifest1.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte(""),
			Error:  nil,
		},
	}

	manifest1.DefaultRunner = te
	defer func() { manifest1.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest1.BuildOptions{
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
	output := manifest1.NewOutput(true)
	str := output.Stream("build")
	dr := manifest1.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte("dockerid"),
			Error:  nil,
		},
	}

	manifest1.DefaultRunner = te
	defer func() { manifest1.DefaultRunner = dr }()

	m, err := manifestFixture("specific-service")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest1.BuildOptions{
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
	output := manifest1.NewOutput(true)
	str := output.Stream("build")
	dr := manifest1.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte("dockeid"),
			Error:  nil,
		},
	}

	manifest1.DefaultRunner = te
	defer func() { manifest1.DefaultRunner = dr }()

	m, err := manifestFixture("full-v1")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest1.BuildOptions{
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
	output := manifest1.NewOutput(true)
	str := output.Stream("build")
	dr := manifest1.DefaultRunner
	te := NewTestExecer()
	manifest1.DefaultRunner = te
	defer func() { manifest1.DefaultRunner = dr }()

	m, err := manifestFixture("repeat-simple")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest1.BuildOptions{
		Cache: false,
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile", "-t", "web/web", "fixtures"}
	cmd2 := []string{"docker", "build", "--no-cache", "-f", "fixtures/other/Dockerfile", "-t", "web/other", "fixtures/other"}
	cmd3 := []string{"docker", "tag", "web/web", "web/monitor"}

	if assert.Equal(t, len(te.Commands), 3) {
		assert.Equal(t, cmd1, te.Commands[0].Args)
		assert.Equal(t, cmd2, te.Commands[1].Args)
		assert.Equal(t, cmd3, te.Commands[2].Args)
	}
}

func TestBuildRepeatImage(t *testing.T) {
	output := manifest1.NewOutput(true)
	str := output.Stream("build")
	dr := manifest1.DefaultRunner
	te := NewTestExecer()
	te.CannedResponses = []ExecResponse{
		{
			Output: []byte(""),
			Error:  nil,
		},
	}
	manifest1.DefaultRunner = te
	defer func() { manifest1.DefaultRunner = dr }()

	m, err := manifestFixture("repeat-image")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest1.BuildOptions{
		Cache: false,
	})
	assert.NoError(t, err)

	cmd1 := []string{"docker", "pull", "convox/rails:latest"}
	cmd2 := []string{"docker", "tag", "convox/rails:latest", "web/web2"}
	cmd3 := []string{"docker", "tag", "convox/rails:latest", "web/web1"}

	if assert.Equal(t, len(te.Commands), 3) {
		assert.Equal(t, cmd1, te.Commands[0].Args)
		assert.Equal(t, cmd2, te.Commands[1].Args)
		assert.Equal(t, cmd3, te.Commands[2].Args)
	}
}

func TestBuildRepeatComplex(t *testing.T) {
	output := manifest1.NewOutput(true)
	str := output.Stream("build")
	dr := manifest1.DefaultRunner
	te := NewTestExecer()
	manifest1.DefaultRunner = te
	defer func() { manifest1.DefaultRunner = dr }()

	m, err := manifestFixture("repeat-complex")
	assert.NoError(t, err)

	err = m.Build("fixtures", "web", str, manifest1.BuildOptions{
		Cache: false,
		Environment: map[string]string{
			"foo": "baz",
		},
	})
	assert.NoError(t, err)

	te.AssertCommands(t, TestCommands{
		[]string{"docker", "build", "--no-cache", "--build-arg", "foo=bar", "-f", "fixtures/Dockerfile", "-t", "web/web", "fixtures"},
		[]string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile.otherf", "-t", "web/otherg", "fixtures"},
		[]string{"docker", "tag", "web/otherg", "web/otherf"},
		[]string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile", "-t", "web/othere", "fixtures"},
		[]string{"docker", "build", "--no-cache", "-f", "fixtures/Dockerfile", "-t", "web/otherd", "fixtures"},
		[]string{"docker", "build", "--no-cache", "--build-arg", "foo=other", "-f", "fixtures/Dockerfile", "-t", "web/otherc", "fixtures"},
		[]string{"docker", "build", "--no-cache", "--build-arg", "foo=baz", "-f", "fixtures/Dockerfile.other", "-t", "web/otherb", "fixtures"},
		[]string{"docker", "build", "--no-cache", "--build-arg", "foo=bar", "-f", "fixtures/other/Dockerfile", "-t", "web/othera", "fixtures/other"},
		[]string{"docker", "tag", "web/web", "web/monitor"},
		[]string{"docker", "tag", "web/othere", "web/first"},
	})
}

func TestDoubleDockerfile(t *testing.T) {
	m, err := manifestFixture("double-dockerfile")

	assert.Nil(t, m, "manifest should be nil")
	assert.Equal(t, fmt.Errorf("dockerfile specified twice for web"), err)
}

func TestPush(t *testing.T) {
	output := manifest1.NewOutput(true)
	str := output.Stream("build")
	dr := manifest1.DefaultRunner
	te := NewTestExecer()
	manifest1.DefaultRunner = te
	defer func() { manifest1.DefaultRunner = dr }()

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
