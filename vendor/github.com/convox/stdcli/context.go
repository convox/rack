package stdcli

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

type Context struct {
	Args  []string
	Flags []*Flag

	engine *Engine
}

func (c *Context) Execute(cmd string, args ...string) ([]byte, error) {
	if c.engine.Executor == nil {
		return nil, fmt.Errorf("no executor")
	}

	return c.engine.Executor.Execute(cmd, args...)
}

func (c *Context) Run(cmd string, args ...string) error {
	if c.engine.Executor == nil {
		return fmt.Errorf("no executor")
	}

	return c.engine.Executor.Run(c, cmd, args...)
}

func (c *Context) Terminal(cmd string, args ...string) error {
	if c.engine.Executor == nil {
		return fmt.Errorf("no executor")
	}

	return c.engine.Executor.Terminal(cmd, args...)
}

func (c *Context) Version() string {
	return c.engine.Version
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

func (c *Context) TerminalRaw() func() {
	fn := c.Reader().TerminalRaw()
	return func() {
		if fn() {
			c.Writef("\r")
		}
	}
}

func (c *Context) TerminalSize() (int, int, error) {
	return terminal.GetSize(int(os.Stdout.Fd()))
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

func (c *Context) LocalSetting(name string) string {
	return c.engine.LocalSetting(name)
}

func (c *Context) SettingDelete(name string) error {
	return c.engine.SettingDelete(name)
}

func (c *Context) SettingRead(name string) (string, error) {
	return c.engine.SettingRead(name)
}

func (c *Context) SettingReadKey(name, key string) (string, error) {
	return c.engine.SettingReadKey(name, key)
}

func (c *Context) SettingWrite(name, value string) error {
	return c.engine.SettingWrite(name, value)
}

func (c *Context) SettingWriteKey(name, key, value string) error {
	return c.engine.SettingWriteKey(name, key, value)
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

func (c *Context) Error(err error) error {
	return c.Writer().Error(err)
}

func (c *Context) Errorf(format string, args ...interface{}) error {
	return c.Writer().Errorf(format, args...)
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
