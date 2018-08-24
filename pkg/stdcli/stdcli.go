package stdcli

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/urfave/cli.v1"
)

var (
	Binary   string
	Commands []cli.Command

// FileWriter func(filename string, data []byte, perm os.FileMode) error
// Exiter     func(code int)
// Runner     func(bin string, args ...string) error
// Spinner    *spinner.Spinner
)

func init() {
	Binary = filepath.Base(os.Args[0])
	// Exiter = os.Exit
	// FileWriter = ioutil.WriteFile
	// Querier = queryExecCommand
	// Runner = runExecCommand
	// Spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)

	cli.AppHelpTemplate = `{{.Name}}: {{.Usage}}

Usage:
  {{.Name}} <command> [args...]

Subcommands: ({{.Name}} help <subcommand>)
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Description}}
  {{end}}{{if .VisibleFlags}}
Options:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
`

	cli.CommandHelpTemplate = fmt.Sprintf(`%s {{.FullName}}: {{.Description}}

Usage:
  %s {{.FullName}} {{.Usage}}
{{if .Subcommands}}
Subcommands: (%s {{.FullName}} help <subcommand>)
  {{range .Subcommands}}{{join .Names ", "}}{{ "\t" }}{{.Description}}
  {{end}}{{end}}{{if .VisibleFlags}}
Options:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{ end }}
`, Binary, Binary, Binary)

	cli.SubcommandHelpTemplate = `{{.Name}}: {{.Usage}}

Usage:
  {{.Name}} <command> [args...]

Subcommands: ({{.Name}} help <subcommand>)
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Description}}
  {{end}}{{if .VisibleFlags}}
Options:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
`
}

func New() *cli.App {
	app := cli.NewApp()

	app.EnableBashCompletion = true

	app.Name = Binary
	app.Commands = Commands

	app.CommandNotFound = func(c *cli.Context, cmd string) {
		fmt.Fprintf(os.Stderr, "No such command \"%s\". Try `%s help`\n", cmd, Binary)
		os.Exit(1)
	}

	app.Writer = DefaultWriter

	return app
}

func Debug() bool {
	if debug := os.Getenv("DEBUG"); debug != "" {
		return true
	}
	return false
}

// If user specifies the app's name from command line, then use it;
// if not, try to read the app name from .convox/app
// otherwise use the current working directory's name
func DirApp(c *cli.Context, wd string) (string, string, error) {
	abs, err := filepath.Abs(wd)

	if err != nil {
		return "", "", err
	}
	app := c.String("app")

	if app == "" {
		app = ReadSetting("app")
	}

	if app == "" {
		app = filepath.Base(abs)
	}

	app = strings.ToLower(app)

	// If there are dots in the directory name, replace them with hyphens instead
	app = strings.Replace(app, ".", "-", -1)

	return abs, app, nil
}

func ReadSetting(setting string) string {
	value, err := ioutil.ReadFile(fmt.Sprintf(".convox/%s", setting))
	if err != nil {
		return ""
	}

	output := strings.TrimSpace(string(value))

	return output
}

func RegisterCommand(cmd cli.Command) {
	Commands = append(Commands, cmd)
}

func VersionPrinter(printer func(*cli.Context)) {
	cli.VersionPrinter = printer
}

func WriteSetting(setting, value string) error {
	err := ioutil.WriteFile(fmt.Sprintf(".convox/%s", setting), []byte(value), 0777)

	return err
}

// IsTerminal tells you if a given file descriptor has a tty on the other side
func IsTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}

	// stat.Mode() & os.ModeCharDevice) == 0 means data is being piped to stdin
	// otherwise stdin is from a terminal
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func Usage(c *cli.Context) error {
	cli.ShowCommandHelp(c, c.Command.Name)
	return nil
}

func runExecCommand(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if Debug() {
		fmt.Fprintf(os.Stderr, "DEBUG: exec: '%v', '%v', '%v'\n", bin, args, err)
	}

	return err
}
