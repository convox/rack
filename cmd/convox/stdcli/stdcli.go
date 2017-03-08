package stdcli

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/urfave/cli.v1"

	"github.com/briandowns/spinner"
	"github.com/segmentio/analytics-go"
	"github.com/stvp/rollbar"
)

// HelpFlags is a slice of all the strings that should be treated as help flags by the CLI
var HelpFlags = []string{"--help", "-h", "h", "help"}

var (
	Binary     string
	Commands   []cli.Command
	FileWriter func(filename string, data []byte, perm os.FileMode) error
	Exiter     func(code int)
	Runner     func(bin string, args ...string) error
	Querier    func(bin string, args ...string) ([]byte, error)
	Spinner    *spinner.Spinner
	Tagger     func() string
)

func init() {
	Binary = filepath.Base(os.Args[0])
	Exiter = os.Exit
	FileWriter = ioutil.WriteFile
	Querier = queryExecCommand
	Runner = runExecCommand
	Spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	Tagger = tagTimeUnix

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

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "app, a",
			Usage: "app name inferred from current directory if not specified",
		},
		cli.StringFlag{
			Name:  "rack",
			Usage: "rack name",
		},
	}

	app.CommandNotFound = func(c *cli.Context, cmd string) {
		fmt.Fprintf(os.Stderr, "No such command \"%s\". Try `%s help`\n", cmd, Binary)
		os.Exit(1)
	}

	app.Writer = DefaultWriter
	app.Before = ValidatePreconditions(CliCheckEnv)

	return app
}

// ValidatePreconditions runs one or more cli.BeforeFuncs where called in Command.Before
func ValidatePreconditions(preconditions ...cli.BeforeFunc) cli.BeforeFunc {
	return func(c *cli.Context) error {
		for _, condition := range preconditions {
			if err := condition(c); err != nil {
				return err
			}
		}
		return nil
	}
}

func Debug() bool {
	if debug := os.Getenv("DEBUG"); debug != "" {
		return true
	}
	return false
}

// RecoverFlag allows us to capture things like --app FOO which would otherwise be discarded by urfave/cli if passed in position 0
func RecoverFlag(c *cli.Context, flagNames ...string) string {
	for _, flagName := range flagNames {
		f := c.String(flagName)
		if f != "" {
			return f
		}

		f = ParseOpts(os.Args)[flagName]
		if f != "" {
			// ParseOpts() includes everything after the flag, so discard everything after the first space
			return strings.Split(f, " ")[0]
		}
	}
	return ""
}

// If user specifies the app's name from command line, then use it;
// if not, try to read the app name from .convox/app
// otherwise use the current working directory's name
func DirApp(c *cli.Context, wd string) (string, string, error) {
	abs, err := filepath.Abs(wd)

	if err != nil {
		return "", "", err
	}
	app := RecoverFlag(c, "a", "app")

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

type QOSEventProperties struct {
	AppType         string
	Error           error
	Start           time.Time
	ValidationError error
}

// QOSEventSend sends an internal CLI event to segment for quality-of-service purposes.
// If the event is an error it also sends the error to rollbar, then displays the
// error to the user and exits non-zero.
func QOSEventSend(system, id string, ep QOSEventProperties) error {
	// don't send events under `make test`
	if os.Getenv("PROVIDER") == "test" {
		return nil
	}

	rollbar.Token = "8481f1ec73f549ce8b81711ca4fdf98a"
	rollbar.Environment = id

	segment := analytics.New("JcNCirASuqEvuWhL8K87JTsUkhY68jvX")

	props := map[string]interface{}{}

	if ep.Error != nil {
		props["error"] = ep.Error.Error()
		rollbar.Error(rollbar.ERR, ep.Error, &rollbar.Field{"id", id})
	}

	if ep.ValidationError != nil {
		props["validation_error"] = ep.ValidationError.Error()
	}

	if ep.AppType != "" {
		props["app_type"] = ep.AppType
	}

	if !ep.Start.IsZero() {
		props["elapsed"] = float64(time.Since(ep.Start).Nanoseconds()) / 1000000
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

	if os.Getenv("ROLLBAR_TOKEN") != "" {
		rollbar.Wait()
	}

	if ep.ValidationError != nil {
		return ep.ValidationError
	}

	if ep.Error != nil {
		return ep.Error
	}

	return nil
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
		isFlag := strings.HasPrefix(token, "-")
		if isFlag {
			key = strings.TrimLeft(token, "-")
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

// CliCheckEnv takes cli.Context as an arg so it can be used as a BeforeFunc
func CliCheckEnv(c *cli.Context) error {
	return CheckEnv()
}

// CheckEnv validates that relevant envvars have acceptable values
func CheckEnv() error {
	vars := map[string][]string{
		"CONVOX_DEBUG": []string{"true", "false", "1", "0", ""},
		"CONVOX_WAIT":  []string{"true", "false", "1", "0", ""},
		"RACK_PRIVATE": []string{"true", "false", "1", "0", ""},
	}

	for varName, okVals := range vars {
		ev := strings.ToLower(os.Getenv(varName))
		ok := false
		for _, val := range okVals {
			if ev == val {
				ok = true
				break
			}
		}
		if !ok {
			msg := fmt.Sprintf("'%s' is not a valid value for environment variable %s ", os.Getenv(varName), varName)
			msg += fmt.Sprintf("(expected: %s)", okVals)
			Warn(msg)
		}
	}
	return nil
}
