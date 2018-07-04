package stdcli

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type Context struct {
	Args  []string
	Flags []*Flag

	engine *Engine
	state  *terminal.State
}

func (c *Context) Arg(i int) string {
	if i < len(c.Args) {
		return c.Args[i]
	}

	return ""
}

func (c *Context) Flag(name string) *Flag {
	for _, f := range c.Flags {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func (c *Context) Bool(name string) bool {
	if f := c.Flag(name); f != nil && f.Type() == "bool" {
		switch t := f.Value.(type) {
		case nil:
			v, _ := f.Default.(bool)
			return v
		case bool:
			return t
		}
	}
	return false
}

func (c *Context) Int(name string) int {
	if f := c.Flag(name); f != nil && f.Type() == "int" {
		switch t := f.Value.(type) {
		case nil:
			v, _ := f.Default.(int)
			return v
		case int:
			return t
		}
	}
	return 0
}

func (c *Context) String(name string) string {
	if f := c.Flag(name); f != nil && f.Type() == "string" {
		switch t := f.Value.(type) {
		case nil:
			v, _ := f.Default.(string)
			return v
		case string:
			return t
		}
	}
	return ""
}

func (c *Context) Value(name string) interface{} {
	if f := c.Flag(name); f != nil {
		return f.Value
	}
	return nil
}

func (c *Context) Info() *Info {
	return &Info{Context: c}
}

func (c *Context) ReadSecret() (string, error) {
	data, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (c *Context) TerminalRaw() error {
	state, err := terminal.MakeRaw(int(c.Reader().Fd()))
	if err != nil {
		return err
	}

	c.state = state

	return nil
}

func (c *Context) TerminalRestore() error {
	if c.state == nil {
		return nil
	}

	if err := terminal.Restore(int(c.Reader().Fd()), c.state); err != nil {
		return err
	}

	c.Writef("\r")

	return nil
}

func (c *Context) TerminalSize() (int, int, error) {
	return terminal.GetSize(int(c.Reader().Fd()))
}

func (c *Context) Fail(err error) {
	if err != nil {
		c.Writer().Error(err)
		os.Exit(1)
	}
}

func (c *Context) Read(data []byte) (int, error) {
	return c.Reader().Read(data)
}

func (c *Context) Reader() *Reader {
	return c.engine.Reader
}

func (c *Context) SettingDelete(name string) error {
	file, err := c.engine.settingFile(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(file); err != nil {
		return err
	}

	return nil
}

func (c *Context) LocalSetting(name string) string {
	file := filepath.Join(c.engine.localSettingDir(), name)

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return ""
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func (c *Context) SettingRead(name string) (string, error) {
	file, err := c.engine.settingFile(name)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(file)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func (c *Context) SettingWrite(name, value string) error {
	file, err := c.engine.settingFile(name)
	if err != nil {
		return err
	}

	dir := filepath.Dir(file)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	if err := ioutil.WriteFile(file, []byte(value), 0600); err != nil {
		return err
	}

	return nil
}

func (c *Context) Table(columns ...string) *Table {
	return &Table{Columns: columns, Context: c}
}

func (c *Context) Write(data []byte) (int, error) {
	return c.Writer().Write(data)
}

func (c *Context) Writer() *Writer {
	return c.engine.Writer
}

func (c *Context) OK(id ...string) error {
	c.Writer().Writef("<ok>OK</ok>")

	if len(id) > 0 {
		c.Writer().Writef(", <id>%s</id>", strings.Join(id, " "))
	}

	c.Writer().Writef("\n")

	return nil
}

func (c *Context) Startf(format string, args ...interface{}) {
	c.Writer().Writef(fmt.Sprintf("%s... ", format), args...)
}

func (c *Context) Writef(format string, args ...interface{}) error {
	_, err := c.Writer().Writef(format, args...)
	return err
}

func (c *Context) Options(opts interface{}) error {
	v := reflect.ValueOf(opts).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		u := v.Field(i)

		if n := f.Tag.Get("flag"); n != "" {
			name := strings.Split(n, ",")[0]
			switch typeString(f.Type.Elem()) {
			case "bool":
				if x, ok := c.Value(name).(bool); ok {
					u.Set(reflect.ValueOf(&x))
				}
			case "duration":
				if x, ok := c.Value(name).(time.Duration); ok {
					u.Set(reflect.ValueOf(&x))
				}
			case "int":
				if x, ok := c.Value(name).(int); ok {
					u.Set(reflect.ValueOf(&x))
				}
			case "string":
				if x, ok := c.Value(name).(string); ok {
					u.Set(reflect.ValueOf(&x))
				}
			default:
				return fmt.Errorf("unknown flag type: %s", f.Type.Elem().Kind())
			}
		}
	}

	return nil
}
