package cli

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("build", "create a build", Build, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.BuildCreateOptions{}), flagRack, flagApp, flagId),
		Usage:    "[dir]",
		Validate: stdcli.ArgsMax(1),
	})

	register("builds", "list builds", Builds, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.BuildListOptions{}), flagRack, flagApp),
		Validate: stdcli.Args(0),
	})

	register("builds export", "export a build", BuildsExport, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagApp,
			stdcli.StringFlag("file", "f", "import from file"),
		},
		Usage:    "<build>",
		Validate: stdcli.Args(1),
	})

	register("builds import", "import a build", BuildsImport, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagApp,
			flagId,
			stdcli.StringFlag("file", "f", "export to file"),
		},
		Validate: stdcli.Args(0),
	})

	register("builds info", "get information about a build", BuildsInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<build>",
		Validate: stdcli.Args(1),
	})

	register("builds logs", "get logs for a build", BuildsLogs, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<build>",
		Validate: stdcli.Args(1),
	})
}

func Build(rack sdk.Interface, c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	b, err := build(rack, c, c.Bool("development"))
	if err != nil {
		return err
	}

	c.Writef("Build:   <build>%s</build>\n", b.Id)
	c.Writef("Release: <release>%s</release>\n", b.Release)

	if c.Bool("id") {
		fmt.Fprintf(stdout, b.Release)
	}

	return nil
}

func build(rack sdk.Interface, c *stdcli.Context, development bool) (*structs.Build, error) {
	var opts structs.BuildCreateOptions

	if development {
		opts.Development = options.Bool(true)
	}

	if err := c.Options(&opts); err != nil {
		return nil, err
	}

	if opts.Description == nil {
		if err := exec.Command("git", "diff", "--quiet").Run(); err == nil {
			if data, err := exec.Command("git", "log", "-n", "1", "--pretty=%h %s", "--abbrev=10").CombinedOutput(); err == nil {
				opts.Description = options.String(fmt.Sprintf("build %s", strings.TrimSpace(string(data))))
			}
		}
	}

	c.Startf("Packaging source")

	data, err := helpers.Tarball(coalesce(c.Arg(0), "."))
	if err != nil {
		return nil, err
	}

	c.OK()

	s, err := rack.SystemGet()
	if err != nil {
		return nil, err
	}

	var b *structs.Build

	if s.Version < "20180708231844" {
		c.Startf("Starting build")

		b, err = rack.BuildCreateUpload(app(c), bytes.NewReader(data), opts)
		if err != nil {
			return nil, err
		}
	} else {
		tmp, err := generateTempKey()
		if err != nil {
			return nil, err
		}

		tmp += ".tgz"

		c.Startf("Uploading source")

		o, err := rack.ObjectStore(app(c), tmp, bytes.NewReader(data), structs.ObjectStoreOptions{})
		if err != nil {
			return nil, err
		}

		c.OK()

		c.Startf("Starting build")

		b, err = rack.BuildCreate(app(c), o.Url, opts)
		if err != nil {
			return nil, err
		}
	}

	c.OK()

	r, err := rack.BuildLogs(app(c), b.Id, structs.LogsOptions{})
	if err != nil {
		return nil, err
	}

	count, _ := io.Copy(c, r)
	defer finalizeBuildLogs(rack, c, b, count)

	for {
		b, err = rack.BuildGet(app(c), b.Id)
		if err != nil {
			return nil, err
		}

		if b.Status == "failed" {
			return nil, fmt.Errorf("build failed")
		}

		if b.Status != "running" {
			break
		}

		time.Sleep(1 * time.Second)
	}

	return b, nil
}

func finalizeBuildLogs(rack structs.Provider, c *stdcli.Context, b *structs.Build, count int64) error {
	r, err := rack.BuildLogs(b.App, b.Id, structs.LogsOptions{})
	if err != nil {
		return err
	}
	defer r.Close()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if int64(len(data)) > count {
		c.Write(data[count:])
	}

	return nil
}

func Builds(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.BuildListOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	bs, err := rack.BuildList(app(c), opts)
	if err != nil {
		return err
	}

	t := c.Table("ID", "STATUS", "RELEASE", "STARTED", "ELAPSED", "DESCRIPTION")

	for _, b := range bs {
		started := helpers.Ago(b.Started)
		elapsed := helpers.Duration(b.Started, b.Ended)

		t.AddRow(b.Id, b.Status, b.Release, started, elapsed, b.Description)
	}

	return t.Print()
}

func BuildsExport(rack sdk.Interface, c *stdcli.Context) error {
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

	c.Startf("Exporting build")

	if err := rack.BuildExport(app(c), c.Arg(0), w); err != nil {
		return err
	}

	return c.OK()
}

func BuildsImport(rack sdk.Interface, c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

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

	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	c.Startf("Importing build")

	var b *structs.Build

	if s.Version <= "20180416200237" {
		b, err = rack.BuildImportMultipart(app(c), r)
	} else if s.Version <= "20180708231844" {
		b, err = rack.BuildImportUrl(app(c), r)
	} else {
		b, err = rack.BuildImport(app(c), r)
	}
	if err != nil {
		return err
	}

	c.OK(b.Release)

	if c.Bool("id") {
		fmt.Fprintf(stdout, b.Release)
	}

	return nil
}

func BuildsInfo(rack sdk.Interface, c *stdcli.Context) error {
	b, err := rack.BuildGet(app(c), c.Arg(0))
	if err != nil {
		return err
	}

	i := c.Info()

	i.Add("Id", b.Id)
	i.Add("Status", b.Status)
	i.Add("Release", b.Release)
	i.Add("Description", b.Description)
	i.Add("Started", helpers.Ago(b.Started))
	i.Add("Elapsed", helpers.Duration(b.Started, b.Ended))

	return i.Print()
}

func BuildsLogs(rack sdk.Interface, c *stdcli.Context) error {
	var opts structs.LogsOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	r, err := rack.BuildLogs(app(c), c.Arg(0), opts)
	if err != nil {
		return err
	}

	io.Copy(c, r)

	return nil
}
