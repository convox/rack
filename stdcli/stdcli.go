package stdcli

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/briandowns/spinner"
	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
)

var (
	Binary   string
	Commands []cli.Command
	Exiter   func(code int)
	Runner   func(bin string, args ...string) error
	Querier  func(bin string, args ...string) ([]byte, error)
	Spinner  *spinner.Spinner
	Tagger   func() string
)

func init() {
	Binary = filepath.Base(os.Args[0])
	Exiter = os.Exit
	Querier = queryExecCommand
	Runner = runExecCommand
	Spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	Tagger = tagTimeUnix

	cli.AppHelpTemplate = `{{.Name}}: {{.Usage}}

Usage:
  {{.Name}} <command> [args...]

Commands:
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Description}}
  {{end}}{{if .Flags}}
Options:
  {{range .Flags}}{{.}}
  {{end}}{{end}}
`

	cli.CommandHelpTemplate = fmt.Sprintf(`%s {{.Name}}: {{.Description}}

Usage:
  %s {{.Name}} {{.Usage}}
{{if .Flags}}
Options:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}
`, Binary, Binary)
}

func New() *cli.App {
	app := cli.NewApp()

	app.Name = Binary
	app.Commands = Commands

	return app
}

func DirApp(c *cli.Context, wd string) (string, string, error) {
	abs, err := filepath.Abs(wd)

	if err != nil {
		return "", "", err
	}

	app := c.String("app")

	if app == "" {
		app = path.Base(abs)
	}

	return abs, app, nil
}

func RegisterCommand(cmd cli.Command) {
	Commands = append(Commands, cmd)
}

func Run(bin string, args ...string) error {
	return Runner(bin, args...)
}

func Query(bin string, args ...string) ([]byte, error) {
	return Querier(bin, args...)
}

func Tag() string {
	return Tagger()
}

func VersionPrinter(printer func(*cli.Context)) {
	cli.VersionPrinter = printer
}

func Error(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	Exiter(1)
}

func Usage(c *cli.Context, name string) {
	cli.ShowCommandHelp(c, name)
	Exiter(0)
}

func runExecCommand(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func queryExecCommand(bin string, args ...string) ([]byte, error) {
	return exec.Command(bin, args...).CombinedOutput()
}

func tagTimeUnix() string {
	return fmt.Sprintf("%v", time.Now().Unix())
}
