package stdcli

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/segmentio/analytics-go"
	"github.com/stvp/rollbar"
	"gopkg.in/urfave/cli.v1"
)

var (
	Binary   string
	Commands []cli.Command
	Exiter   func(code int)
	Runner   func(bin string, args ...string) error
	Querier  func(bin string, args ...string) ([]byte, error)
	Spinner  *spinner.Spinner
	Tagger   func() string
	Writer   func(filename string, data []byte, perm os.FileMode) error
)

func init() {
	Binary = filepath.Base(os.Args[0])
	Exiter = os.Exit
	Querier = queryExecCommand
	Runner = runExecCommand
	Spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	Tagger = tagTimeUnix
	Writer = ioutil.WriteFile

	cli.AppHelpTemplate = `{{.Name}}: {{.Usage}}

Usage:
  {{.Name}} <command> [args...]

Subcommands: ({{.Name}} help <subcommand>)
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Description}}
  {{end}}{{if .Flags}}
Options:
  {{range .Flags}}{{.}}
  {{end}}{{end}}
`

	cli.CommandHelpTemplate = fmt.Sprintf(`%s {{.FullName}}: {{.Description}}

Usage:
  %s {{.FullName}} {{.Usage}}
{{if .Subcommands}}
Subcommands: (%s {{.FullName}} help <subcommand>)
  {{range .Subcommands}}{{join .Names ", "}}{{ "\t" }}{{.Description}}
  {{end}}{{end}}{{if .Flags}}
Options:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}
`, Binary, Binary, Binary)

	cli.SubcommandHelpTemplate = `{{.Name}}: {{.Usage}}

Usage:
  {{.Name}} <command> [args...]

Subcommands: ({{.Name}} help <subcommand>)
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Description}}
  {{end}}{{if .Flags}}
Options:
  {{range .Flags}}{{.}}
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
	}

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
		app = path.Base(abs)
	}

	app = strings.ToLower(app)

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

func WriteSetting(setting, value string) error {
	err := ioutil.WriteFile(fmt.Sprintf(".convox/%s", setting), []byte(value), 0777)

	return err
}

func Error(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	Exiter(1)
}

func ExitError(err error) error {
	return cli.NewExitError(fmt.Sprintf("ERROR: %s", err.Error()), 1)
}

type QOSEventProperties struct {
	Error error
	Start time.Time
}

// QOSEventSend sends an internal CLI event to segment for quality-of-service purposes.
// If the event is an error it also sends the error to rollbar, then displays the
// error to the user and exits non-zero.
func QOSEventSend(system, id string, ep QOSEventProperties) error {
	rollbar.Token = "8481f1ec73f549ce8b81711ca4fdf98a"
	rollbar.Environment = id

	segment := analytics.New("JcNCirASuqEvuWhL8K87JTsUkhY68jvX")

	props := map[string]interface{}{}

	if ep.Error != nil {
		props["error"] = ep.Error.Error()
		rollbar.Error(rollbar.ERR, ep.Error, &rollbar.Field{"id", id})
	}

	if !ep.Start.IsZero() {
		props["elapsed"] = float64(time.Now().Sub(ep.Start).Nanoseconds()) / 1000000
	}

	err := segment.Track(&analytics.Track{
		Event:      system,
		UserId:     id,
		Properties: props,
	})
	if err != nil {
		rollbar.Error(rollbar.ERR, err, &rollbar.Field{"id", id})
	}

	err = segment.Close()
	if err != nil {
		rollbar.Error(rollbar.ERR, err, &rollbar.Field{"id", id})
	}

	rollbar.Wait()

	if ep.Error != nil {
		return ExitError(ep.Error)
	}
	return nil
}

func Usage(c *cli.Context, name string) {
	cli.ShowCommandHelp(c, name)
	Exiter(129)
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

func queryExecCommand(bin string, args ...string) ([]byte, error) {
	return exec.Command(bin, args...).CombinedOutput()
}

func tagTimeUnix() string {
	return fmt.Sprintf("%v", time.Now().Unix())
}

func ParseOpts(args []string) map[string]string {
	options := make(map[string]string)
	var key string

	for _, token := range args {
		isFlag := strings.HasPrefix(token, "--")
		if isFlag {
			key = token[2:]
			value := ""
			if strings.Contains(key, "=") {
				pivot := strings.Index(key, "=")
				value = key[pivot+1:]
				key = key[0:pivot]
			}
			options[key] = value
		} else {
			options[key] = strings.TrimSpace(options[key] + " " + token)
		}
	}

	return options
}
