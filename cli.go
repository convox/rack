package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
)

var (
	Binary   string
	Commands []cli.Command
)

func init() {
	Binary = filepath.Base(os.Args[0])

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

func NewCli() *cli.App {
	app := cli.NewApp()

	app.Name = Binary
	app.Commands = Commands

	return app
}

func RegisterCommand(cmd cli.Command) {
	Commands = append(Commands, cmd)
}

func Usage(c *cli.Context, name string) {
	cli.ShowCommandHelp(c, name)
	os.Exit(0)
}
