package stdcli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
)

var (
	Binary   string
	Commands []cli.Command
	Exiter   func(code int)
)

func init() {
	Binary = filepath.Base(os.Args[0])
	Exiter = os.Exit

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

func RegisterCommand(cmd cli.Command) {
	Commands = append(Commands, cmd)
}

func Error(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	Exiter(1)
}

func Usage(c *cli.Context, name string) {
	cli.ShowCommandHelp(c, name)
	Exiter(0)
}
