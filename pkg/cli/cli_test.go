package cli_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
)

var fxSystem = structs.System{
	Version: "20180829000000",
}

var fxSystemClassic = structs.System{
	Version: "20180101000000",
}

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

type result struct {
	Code   int
	Stdout string
	Stderr string
}

func (r *result) StdoutLines() int {
	lines := strings.Split(r.Stdout, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[0 : len(lines)-1]
	}
	return len(lines)
}

func (r *result) StdoutLine(line int) string {
	return strings.Split(r.Stdout, "\n")[line]
}
