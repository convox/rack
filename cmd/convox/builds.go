package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/convox/rack/helpers"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("build", "create a build", Build, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.BuildCreateOptions{}), flagRack, flagApp, flagId),
		Usage:    "[dir]",
		Validate: stdcli.ArgsMax(1),
	})

	CLI.Command("builds", "list builds", Builds, stdcli.CommandOptions{
		Flags:    append(stdcli.OptionFlags(structs.BuildListOptions{}), flagRack, flagApp),
		Validate: stdcli.Args(0),
	})

	CLI.Command("builds export", "export a build", BuildsExport, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagApp,
			stdcli.StringFlag("file", "f", "import from file"),
		},
		Usage:    "<build>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("builds import", "import a build", BuildsImport, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			flagApp,
			flagId,
			stdcli.StringFlag("file", "f", "export to file"),
		},
		Validate: stdcli.Args(0),
	})

	CLI.Command("builds info", "get information about a build", BuildsInfo, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<build>",
		Validate: stdcli.Args(1),
	})

	CLI.Command("builds logs", "get logs for a build", BuildsLogs, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack, flagApp},
		Usage:    "<build>",
		Validate: stdcli.Args(1),
	})
}

func Build(c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	b, err := build(c)
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

func build(c *stdcli.Context) (*structs.Build, error) {
	var opts structs.BuildCreateOptions

	if err := c.Options(&opts); err != nil {
		return nil, err
	}

	c.Startf("Packaging source")

	data, err := helpers.Tarball(coalesce(c.Arg(0), "."))
	if err != nil {
		return nil, err
	}

	c.OK()

	s, err := provider(c).SystemGet()
	if err != nil {
		return nil, err
	}

	var b *structs.Build

	if s.Version < "20180222001443" {
		c.Startf("Starting build")

		b, err = provider(c).BuildCreateUpload(app(c), bytes.NewReader(data), opts)
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

		o, err := provider(c).ObjectStore(app(c), tmp, bytes.NewReader(data), structs.ObjectStoreOptions{})
		if err != nil {
			return nil, err
		}

		c.OK()

		c.Startf("Starting build")

		b, err = provider(c).BuildCreate(app(c), o.Url, opts)
		if err != nil {
			return nil, err
		}
	}

	c.OK()

	r, err := provider(c).BuildLogs(app(c), b.Id, structs.LogsOptions{})
	if err != nil {
		return nil, err
	}

	io.Copy(c, r)

	for {
		b, err = provider(c).BuildGet(app(c), b.Id)
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

func Builds(c *stdcli.Context) error {
	var opts structs.BuildListOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	bs, err := provider(c).BuildList(app(c), opts)
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

func BuildsExport(c *stdcli.Context) error {
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

	if err := provider(c).BuildExport(app(c), c.Arg(0), w); err != nil {
		return err
	}

	return c.OK()
}

func BuildsImport(c *stdcli.Context) error {
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
		r = ioutil.NopCloser(c.Reader().File)
	}

	defer r.Close()

	s, err := provider(c).SystemGet()
	if err != nil {
		return err
	}

	c.Startf("Importing build")

	var b *structs.Build

	if s.Version <= "20180416200237" {
		b, err = provider(c).BuildImportMultipart(app(c), r)
	} else if s.Version <= "20180708231844" {
		b, err = provider(c).BuildImportUrl(app(c), r)
	} else {
		b, err = provider(c).BuildImport(app(c), r)
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

func BuildsInfo(c *stdcli.Context) error {
	b, err := provider(c).BuildGet(app(c), c.Arg(0))
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

func BuildsLogs(c *stdcli.Context) error {
	var opts structs.LogsOptions

	if err := c.Options(&opts); err != nil {
		return err
	}

	r, err := provider(c).BuildLogs(app(c), c.Arg(0), opts)
	if err != nil {
		return err
	}

	io.Copy(c, r)

	return nil
}
