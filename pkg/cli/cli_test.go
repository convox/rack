package cli_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	shellquote "github.com/kballard/go-shellquote"
)

type result struct {
	Code   int
	Stdout string
	Stderr string
}

func (r *result) StdoutLines() int {
	return len(strings.Split(strings.TrimSuffix(r.Stdout, "\n"), "\n"))
}

func (r *result) StdoutLine(line int) string {
	return strings.Split(r.Stdout, "\n")[line]
}

func testClient(t *testing.T, fn func(*cli.Engine, *mocksdk.Interface)) {
	i := &mocksdk.Interface{}

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
