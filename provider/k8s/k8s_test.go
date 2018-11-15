package k8s_test

import (
	"testing"

	"github.com/convox/rack/provider/k8s"
	fakek8s "k8s.io/client-go/kubernetes/fake"
)

func testProvider(t *testing.T, fn func(*k8s.Provider, *fakek8s.Clientset)) {
	c := fakek8s.NewSimpleClientset()

	p := &k8s.Provider{
		Cluster: c,
		Rack:    "test",
	}

	// p := k8s.NewFromEnv
	// e := cli.New("convox", "test")

	// e.Client = i

	// tmp, err := ioutil.TempDir("", "")
	// require.NoError(t, err)
	// e.Settings = tmp
	// defer os.RemoveAll(tmp)

	fn(p, c)

	// i.AssertExpectations(t)
}

// func testExecute(e *cli.Engine, cmd string, stdin io.Reader) (*result, error) {
//   if stdin == nil {
//     stdin = &bytes.Buffer{}
//   }

//   stdout := bytes.Buffer{}
//   stderr := bytes.Buffer{}

//   e.Reader.Reader = stdin

//   e.Writer.Color = false
//   e.Writer.Stdout = &stdout
//   e.Writer.Stderr = &stderr

//   cp, err := shellquote.Split(cmd)
//   if err != nil {
//     return nil, err
//   }

//   code := e.Execute(cp)

//   res := &result{
//     Code:   code,
//     Stdout: stdout.String(),
//     Stderr: stderr.String(),
//   }

//   return res, nil
// }

// func testLogs(logs []string) io.ReadCloser {
//   return ioutil.NopCloser(strings.NewReader(fmt.Sprintf("%s\n", strings.Join(logs, "\n"))))
// }

// type result struct {
//   Code   int
//   Stdout string
//   Stderr string
// }

// func (r *result) RequireStderr(t *testing.T, lines []string) {
//   stderr := strings.Split(strings.TrimSuffix(r.Stderr, "\n"), "\n")
//   require.Equal(t, lines, stderr)
// }

// func (r *result) RequireStdout(t *testing.T, lines []string) {
//   stdout := strings.Split(strings.TrimSuffix(r.Stdout, "\n"), "\n")
//   require.Equal(t, lines, stdout)
// }
