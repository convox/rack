package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
	"github.com/stretchr/testify/require"
)

var (
	fxObject = structs.Object{
		Url: "object://test",
	}
	fxSystem = structs.System{
		Version: "20180829000000",
	}
	fxSystemClassic = structs.System{
		Version: "20180101000000",
	}
)

var (
	fxStarted = time.Now().UTC().Add(-48 * time.Hour)
)

func testClient(t *testing.T, fn func(*cli.Engine, *mocksdk.Interface)) {
	testClientWait(t, 0*time.Second, fn)
}

func testClientWait(t *testing.T, wait time.Duration, fn func(*cli.Engine, *mocksdk.Interface)) {
	i := &mocksdk.Interface{}

	cli.WaitDuration = wait

	c := cli.New("convox", "test")

	c.Client = i

	fn(c, i)

	// i.AssertExpectations(t)
}

func testExecute(e *cli.Engine, cmd string, stdin io.Reader) (*result, error) {
	if stdin == nil {
		stdin = &bytes.Buffer{}
	}

	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}

	e.Reader.Reader = stdin

	e.Writer.Color = false
	e.Writer.Stdout = &stdout
	e.Writer.Stderr = &stderr

	cp, err := shellquote.Split(cmd)
	if err != nil {
		return nil, err
	}

	code := e.Execute(cp)

	res := &result{
		Code:   code,
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	return res, nil
}

func testLogs(logs []string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(fmt.Sprintf("%s\n", strings.Join(logs, "\n"))))
}

type result struct {
	Code   int
	Stdout string
	Stderr string
}

func (r *result) RequireStderr(t *testing.T, lines []string) {
	stderr := strings.Split(strings.TrimSuffix(r.Stderr, "\n"), "\n")
	require.Equal(t, lines, stderr)
}

func (r *result) RequireStdout(t *testing.T, lines []string) {
	stdout := strings.Split(strings.TrimSuffix(r.Stdout, "\n"), "\n")
	require.Equal(t, lines, stdout)
}
