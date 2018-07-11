package stdcli

import (
	"fmt"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/pflag"
)

type Engine struct {
	Commands []Command
	Name     string
	Reader   *Reader
	Settings string
	Version  string
	Writer   *Writer
}

func (e *Engine) Command(command, description string, fn HandlerFunc, opts CommandOptions) {
	e.Commands = append(e.Commands, Command{
		Command:     strings.Split(command, " "),
		Description: description,
		Handler:     fn,
		Flags:       opts.Flags,
		Invisible:   opts.Invisible,
		Usage:       opts.Usage,
		Validate:    opts.Validate,
		engine:      e,
	})
}

func (e *Engine) Execute(args []string) int {
	var version bool

	fs := pflag.NewFlagSet(e.Name, pflag.ContinueOnError)
	fs.Usage = func() {}
	fs.BoolVarP(&version, "version", "v", false, "display version")
	fs.Parse(args)

	if version {
		fmt.Println(e.Version)
		return 0
	}

	var m *Command
	var cargs []string

	for _, c := range e.Commands {
		d := c
		if a, ok := d.Match(args); ok {
			if m == nil || len(m.Command) < len(c.Command) {
				m = &d
				cargs = a
			}
		}
	}

	if m == nil {
		m = &(e.Commands[0])
	}

	err := m.Execute(cargs)
	switch t := err.(type) {
	case nil:
		return 0
	case ExitCoder:
		return t.Code()
	default:
		e.Writer.Errorf("%s", t)
		return 1
	}

	return 0
}

func (e *Engine) settingFile(name string) (string, error) {
	if dir := e.Settings; dir != "" {
		return filepath.Join(dir, name), nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, fmt.Sprintf(".%s", e.Name), name), nil
}

func (e *Engine) localSettingDir() string {
	return fmt.Sprintf(".%s", e.Name)
}
