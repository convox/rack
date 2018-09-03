package cli

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("apps", "list apps", Apps, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("apps cancel", "cancel an app update", AppsCancel, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps create", "create an app", AppsCreate, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.AppCreateOptions{}), flagRack, flagWait),
		Usage:    "[name]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps delete", "delete an app", AppsDelete, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagWait},
		Usage:    "<app>",
		Validate: stdcli.Args(1),
	})

	register("apps export", "export an app", AppsExport, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagApp,
			flagRack,
			stdcli.StringFlag("file", "f", "import from file"),
		},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps import", "import an app", AppsImport, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagApp,
			flagRack,
			stdcli.StringFlag("file", "f", "import from file"),
		},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps info", "get information about an app", AppsInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps params", "display app parameters", AppsParams, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps params set", "set app parameters", AppsParamsSet, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack, flagWait},
		Usage:    "<Key=Value> [Key=Value]...",
		Validate: stdcli.ArgsMin(1),
	})

	register("apps sleep", "sleep an app", AppsSleep, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps wake", "wake an app", AppsWake, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})

	register("apps wait", "wait for an app to finish updating", AppsWait, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagApp, flagRack},
		Usage:    "[app]",
		Validate: stdcli.ArgsMax(1),
	})
}

func Apps(rack sdk.Interface, c *stdcli.Context) error {
	as, err := rack.AppList()
	if err != nil {
		return err
	}

	t := c.Table("APP", "STATUS", "GEN", "RELEASE")

	for _, a := range as {
		t.AddRow(a.Name, a.Status, a.Generation, a.Release)
	}

	return t.Print()
}

func AppsCancel(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Cancelling <app>%s</app>", app)

	if err := rack.AppCancel(app); err != nil {
		return err
	}

	return c.OK()
}

func AppsCreate(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	var opts structs.AppCreateOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	c.Startf("Creating <app>%s</app>", app)

	if _, err := rack.AppCreate(app, opts); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppRunning(rack, app); err != nil {
			return err
		}
	}

	return c.OK()
}

func AppsDelete(rack sdk.Interface, c *stdcli.Context) error {
	app := c.Args[0]

	c.Startf("Deleting <app>%s</app>", app)

	if err := rack.AppDelete(app); err != nil {
		return err
	}

	if c.Bool("wait") {
		if err := waitForAppDeleted(rack, c, app); err != nil {
			return err
		}
	}

	return c.OK()
}

func AppsExport(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	var w io.Writer

	if file := c.String("file"); file != "" {
		f, err := os.Create(file)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	} else {
		if c.Writer().IsTerminal() {
			return fmt.Errorf("pipe this command into a file or specify --file")
		}
		w = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	if err := appExport(rack, c, app, w); err != nil {
		return err
	}

	return nil
}

func AppsImport(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	var r io.ReadCloser

	if file := c.String("file"); file != "" {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		r = f
	} else {
		if c.Reader().IsTerminal() {
			return fmt.Errorf("pipe a file into this command or specify --file")
		}
		r = ioutil.NopCloser(c.Reader())
	}

	defer r.Close()

	if err := appImport(rack, c, app, r); err != nil {
		return err
	}

	return nil
}

func AppsInfo(rack sdk.Interface, c *stdcli.Context) error {
	a, err := rack.AppGet(coalesce(c.Arg(0), app(c)))
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Name", a.Name)
	i.Add("Status", a.Status)
	i.Add("Gen", a.Generation)
	i.Add("Release", a.Release)

	return i.Print()
}

func AppsParams(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	var params map[string]string

	app := coalesce(c.Arg(0), app(c))

	if s.Version <= "20180708231844" {
		params, err = rack.AppParametersGet(app)
		if err != nil {
			return err
		}
	} else {
		a, err := rack.AppGet(app)
		if err != nil {
			return err
		}
		params = a.Parameters
	}

	keys := []string{}

	for k := range params {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	i := c.Info()

	for _, k := range keys {
		i.Add(k, params[k])
	}

	return i.Print()
}

func AppsParamsSet(rack sdk.Interface, c *stdcli.Context) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	opts := structs.AppUpdateOptions{
		Parameters: map[string]string{},
	}

	for _, arg := range c.Args {
		parts := strings.SplitN(arg, "=", 2)

		if len(parts) != 2 {
			return fmt.Errorf("Key=Value expected: %s", arg)
		}

		opts.Parameters[parts[0]] = parts[1]
	}

	c.Startf("Updating parameters")

	if s.Version <= "20180708231844" {
		if err := rack.AppParametersSet(app(c), opts.Parameters); err != nil {
			return err
		}
	} else {
		if err := rack.AppUpdate(app(c), opts); err != nil {
			return err
		}
	}

	if c.Bool("wait") {
		if err := waitForAppWithLogs(rack, c, app(c)); err != nil {
			return err
		}
	}

	return c.OK()
}

func AppsSleep(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Sleeping <app>%s</app>", app)

	if err := rack.AppUpdate(app, structs.AppUpdateOptions{Sleep: options.Bool(true)}); err != nil {
		return err
	}

	return c.OK()
}

func AppsWake(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Waking <app>%s</app>", app)

	if err := rack.AppUpdate(app, structs.AppUpdateOptions{Sleep: options.Bool(false)}); err != nil {
		return err
	}

	return c.OK()
}

func AppsWait(rack sdk.Interface, c *stdcli.Context) error {
	app := coalesce(c.Arg(0), app(c))

	c.Startf("Waiting for app")

	if err := waitForAppWithLogs(rack, c, app); err != nil {
		return err
	}

	return c.OK()
}

func appExport(rack sdk.Interface, c *stdcli.Context, app string, w io.Writer) error {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	c.Startf("Exporting app <app>%s</app>", app)

	a, err := rack.AppGet(app)
	if err != nil {
		return err
	}

	for k, v := range a.Parameters {
		if v == "****" {
			delete(a.Parameters, k)
		}
	}

	data, err := json.Marshal(a)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(tmp, "app.json"), data, 0600); err != nil {
		return err
	}

	c.OK()

	if a.Release != "" {
		c.Startf("Exporting env")

		_, r, err := helpers.AppManifest(rack, app)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(filepath.Join(tmp, "env"), []byte(r.Env), 0600); err != nil {
			return err
		}

		c.OK()

		if r.Build != "" {
			c.Startf("Exporting build <build>%s</build>", r.Build)

			fd, err := os.OpenFile(filepath.Join(tmp, "build.tgz"), os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			defer fd.Close()

			if err := rack.BuildExport(app, r.Build, fd); err != nil {
				return err
			}

			c.OK()
		}
	}

	c.Startf("Packaging export")

	tgz, err := helpers.Tarball(tmp)
	if err != nil {
		return err
	}

	if _, err := w.Write(tgz); err != nil {
		return err
	}

	c.OK()

	return nil
}

func appImport(rack sdk.Interface, c *stdcli.Context, app string, r io.Reader) error {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	if err := helpers.Unarchive(gz, tmp); err != nil {
		return err
	}

	var a structs.App

	data, err := ioutil.ReadFile(filepath.Join(tmp, "app.json"))
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	c.Startf("Creating app <app>%s</app>", app)

	if _, err := rack.AppCreate(app, structs.AppCreateOptions{Generation: options.String(a.Generation)}); err != nil {
		return err
	}

	if err := waitForAppRunning(rack, app); err != nil {
		return err
	}

	c.OK()

	build := filepath.Join(tmp, "build.tgz")
	env := filepath.Join(tmp, "env")
	release := ""

	if _, err := os.Stat(build); !os.IsNotExist(err) {
		fd, err := os.Open(build)
		if err != nil {
			return err
		}

		c.Startf("Importing build")

		b, err := rack.BuildImport(app, fd)
		if err != nil {
			return err
		}

		c.OK(b.Release)

		release = b.Release
	}

	if _, err := os.Stat(env); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(env)
		if err != nil {
			return err
		}

		c.Startf("Importing env")

		r, err := rack.ReleaseCreate(app, structs.ReleaseCreateOptions{Env: options.String(string(data))})
		if err != nil {
			return err
		}

		c.OK(r.Id)

		release = r.Id
	}

	if release != "" {
		c.Startf("Promoting <release>%s</release>", release)

		if err := rack.ReleasePromote(app, release); err != nil {
			return err
		}

		if err := waitForAppRunning(rack, app); err != nil {
			return err
		}

		c.OK()
	}

	if len(a.Parameters) > 0 {
		ae, err := rack.AppGet(app)
		if err != nil {
			return err
		}

		if !reflect.DeepEqual(a.Parameters, ae.Parameters) {
			c.Startf("Updating parameters")

			if err := rack.AppUpdate(app, structs.AppUpdateOptions{Parameters: a.Parameters}); err != nil {
				return err
			}

			if err := waitForAppRunning(rack, app); err != nil {
				return err
			}

			c.OK()
		}
	}

	return nil
}
