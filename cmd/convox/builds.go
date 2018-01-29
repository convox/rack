package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"gopkg.in/urfave/cli.v1"

	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
)

var (
	buildCreateFlags = []cli.Flag{
		appFlag,
		rackFlag,
		cli.BoolFlag{
			Name:  "no-cache",
			Usage: "pull fresh image dependencies",
		},
		cli.BoolFlag{
			Name:  "id",
			Usage: "send build logs to stderr, send release id to stdout (useful for scripting)",
		},
		cli.BoolFlag{
			Name:  "incremental",
			Usage: "use incremental build",
		},
		cli.StringFlag{
			Name:   "file, f",
			EnvVar: "COMPOSE_FILE",
			Usage:  "path to an alternate docker compose manifest file",
		},
		cli.StringFlag{
			Name:  "description, d",
			Value: "",
			Usage: "description of the build",
		},
	}
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "build",
		Description: "create a new build",
		Usage:       "[directory] [options]",
		Action:      cmdBuildsCreate,
		Flags:       buildCreateFlags,
	})
	stdcli.RegisterCommand(cli.Command{
		Name:        "builds",
		Description: "manage an app's builds",
		Usage:       "[subcommand] [options] [args...]",
		Action:      cmdBuilds,
		Flags:       []cli.Flag{appFlag, rackFlag},
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new build",
				Usage:       "[directory] [options]",
				ArgsUsage:   "[directory]",
				Action:      cmdBuildsCreate,
				Flags:       buildCreateFlags,
			},
			{
				Name:        "export",
				Description: "export a build artifact to stdout",
				Usage:       "<id>",
				Action:      cmdBuildsExport,
				Flags: []cli.Flag{
					appFlag,
					rackFlag,
					cli.StringFlag{
						Name:  "file, f",
						Usage: "export to file",
					},
				},
			},
			{
				Name:        "logs",
				Description: "get logs for a build",
				Usage:       "<build id>",
				ArgsUsage:   "<build id>",
				Action:      cmdBuildsLogs,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
			{
				Name:        "import",
				Description: "import a build artifact from stdin",
				Usage:       "[options]",
				Action:      cmdBuildsImport,
				Flags: []cli.Flag{
					appFlag,
					rackFlag,
					cli.StringFlag{
						Name:  "file, f",
						Usage: "import from file",
					},
					cli.BoolFlag{
						Name:  "id",
						Usage: "send import logs to stderr, send release id to stdout (useful for scripting)",
					},
				},
			},
			{
				Name:        "info",
				Description: "print output for a build",
				Usage:       "<build id>",
				ArgsUsage:   "<build id>",
				Action:      cmdBuildsInfo,
				Flags:       []cli.Flag{appFlag, rackFlag},
			},
		},
	})
}

func cmdBuilds(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 0)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	builds, err := rackClient(c).GetBuilds(app)
	if err != nil {
		return stdcli.Error(err)
	}

	t := stdcli.NewTable("ID", "STATUS", "RELEASE", "STARTED", "ELAPSED", "DESC")

	for _, build := range builds {
		started := helpers.HumanizeTime(build.Started)
		elapsed := stdcli.Duration(build.Started, build.Ended)

		if build.Ended.IsZero() {
			elapsed = ""
		}

		t.AddRow(build.Id, build.Status, build.Release, started, elapsed, build.Description)
	}

	t.Print()
	return nil
}

func cmdBuildsCreate(c *cli.Context) error {
	stdcli.NeedHelp(c)
	wd := "."

	if len(c.Args()) > 0 {
		stdcli.NeedArg(c, 1)
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)
	if err != nil {
		return stdcli.Error(err)
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.Error(err)
	}

	if a.Status == "creating" {
		return stdcli.Error(fmt.Errorf("app %s is still being created, for more information try `convox apps info`", app))
	}

	if len(c.Args()) > 0 {
		dir = c.Args()[0]
	}

	output := os.Stdout

	if c.Bool("id") {
		output = os.Stderr
	}

	_, release, err := executeBuild(c, dir, app, c.String("file"), c.String("description"), output)
	if err != nil {
		return stdcli.Error(err)
	}

	output.Write([]byte(fmt.Sprintf("Release: %s\n", release)))

	if c.Bool("id") {
		os.Stdout.Write([]byte(release))
		output.Write([]byte("\n"))
	}

	return nil
}

func cmdBuildsInfo(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	build := c.Args()[0]

	b, err := rackClient(c).GetBuild(app, build)
	if err != nil {
		return stdcli.Error(err)
	}

	fmt.Printf("Build        %s\n", b.Id)
	fmt.Printf("Status       %s\n", b.Status)
	fmt.Printf("Release      %s\n", b.Release)
	fmt.Printf("Description  %s\n", b.Description)
	fmt.Printf("Started      %s\n", helpers.HumanizeTime(b.Started))
	fmt.Printf("Elapsed      %s\n", stdcli.Duration(b.Started, b.Ended))

	return nil
}

func cmdBuildsExport(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if stdcli.IsTerminal(os.Stdout) && c.String("file") == "" {
		return stdcli.Error(fmt.Errorf("please pipe the output of this command to a file or specify -f"))
	}

	build := c.Args()[0]

	fmt.Fprintf(os.Stderr, "Exporting %s... ", build)

	out := os.Stdout

	if file := c.String("file"); file != "" {
		fd, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return stdcli.Error(err)
		}
		defer fd.Close()
		out = fd
	}

	if err := rackClient(c).ExportBuild(app, build, out); err != nil {
		return stdcli.Error(err)
	}

	fmt.Fprintf(os.Stderr, "OK\n")

	return nil
}

func cmdBuildsImport(c *cli.Context) error {
	stdcli.NeedHelp(c)
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	if stdcli.IsTerminal(os.Stdin) && c.String("file") == "" {
		return stdcli.Error(fmt.Errorf("please pipe a file into this command or specify -f"))
	}

	in := os.Stdin
	if file := c.String("file"); file != "" {
		fd, err := os.Open(file)
		if err != nil {
			return stdcli.Error(err)
		}
		defer fd.Close()
		in = fd
	}

	output := os.Stdout

	if c.Bool("id") {
		output = os.Stderr
	}

	build, err := rackClient(c).ImportBuild(app, in, client.ImportBuildOptions{Progress: progress("Uploading: ", "Importing build... ", output)})
	if err != nil {
		return stdcli.Error(err)
	}

	output.Write([]byte(fmt.Sprintf("Release: %s\n", build.Release)))

	if c.Bool("id") {
		os.Stdout.Write([]byte(build.Release))
		output.Write([]byte("\n"))
	}

	return nil
}

func cmdBuildsLogs(c *cli.Context) error {
	stdcli.NeedHelp(c)
	stdcli.NeedArg(c, 1)

	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.Error(err)
	}

	build := c.Args()[0]

	if err := rackClient(c).StreamBuildLogs(app, build, os.Stdout); err != nil {
		return stdcli.Error(err)
	}

	return nil
}

func executeBuild(c *cli.Context, source, app, manifest, description string, output io.WriteCloser) (string, string, error) {
	u, err := url.Parse(source)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "http", "https", "ssh":
		return executeBuildURL(c, source, app, manifest, description, output)
	default:
		if c.Bool("incremental") {
			return executeBuildDirIncremental(c, source, app, manifest, description, output)
		} else {
			return executeBuildDir(c, source, app, manifest, description, output)
		}
	}

	return "", "", fmt.Errorf("unreachable")
}

func createIndex(dir string) (client.Index, error) {
	index := client.Index{}

	err := warnUnignoredEnv(dir)
	if err != nil {
		return nil, err
	}

	ignore, err := readDockerIgnore(dir)
	if err != nil {
		return nil, err
	}

	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(resolved, indexWalker(resolved, index, ignore))
	if err != nil {
		return nil, err
	}

	return index, nil
}

func indexWalker(root string, index client.Index, ignore []string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		rel, err := filepath.Rel(root, path)

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		match, err := fileutils.Matches(rel, ignore)
		if err != nil {
			return err
		}

		if match {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		sum := sha256.Sum256(data)
		hash := hex.EncodeToString([]byte(sum[:]))

		index[hash] = client.IndexItem{
			Name:    rel,
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			Size:    len(data),
		}

		return nil
	}
}

func readDockerIgnore(dir string) ([]string, error) {
	fd, err := os.Open(filepath.Join(dir, ".dockerignore"))

	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	ignore, err := dockerignore.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	return ignore, nil
}

func uploadIndex(c *cli.Context, index client.Index, output io.WriteCloser) error {
	missing, err := rackClient(c).IndexMissing(index)
	if err != nil {
		return err
	}

	output.Write([]byte("Identifying changes... "))

	if len(missing) == 0 {
		output.Write([]byte("NONE\n"))
		return nil
	}

	output.Write([]byte(fmt.Sprintf("%d files\n", len(missing))))

	buf := &bytes.Buffer{}

	gz := gzip.NewWriter(buf)

	tw := tar.NewWriter(gz)

	for _, m := range missing {
		data, err := ioutil.ReadFile(index[m].Name)
		if err != nil {
			return err
		}

		header := &tar.Header{
			Typeflag: tar.TypeReg,
			Name:     m,
			Mode:     0600,
			Size:     int64(len(data)),
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if _, err := tw.Write(data); err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}

	if err := gz.Close(); err != nil {
		return err
	}

	if err := rackClient(c).IndexUpdate(buf, client.IndexUpdateOptions{Progress: progress("Uploading: ", "Storing changes... ", output)}); err != nil {
		return err
	}

	output.Write([]byte("OK\n"))

	return nil
}

func executeBuildDirIncremental(c *cli.Context, dir, app, manifest, description string, output io.WriteCloser) (string, string, error) {
	system, err := rackClient(c).GetSystem()
	if err != nil {
		return "", "", err
	}

	// if the rack doesnt support incremental builds then fall back
	if system.Version < "20160226234213" {
		return executeBuildDir(c, dir, app, manifest, description, output)
	}

	cache := !c.Bool("no-cache")

	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	output.Write([]byte("Analyzing source... "))

	index, err := createIndex(dir)
	if err != nil {
		return "", "", err
	}

	output.Write([]byte("OK\n"))

	err = uploadIndex(c, index, output)
	if err != nil {
		return "", "", err
	}

	output.Write([]byte("Starting build... "))

	build, err := rackClient(c).CreateBuildIndex(app, index, cache, manifest, description)
	if err != nil {
		return "", "", err
	}

	return finishBuild(c, app, build, output)
}

func executeBuildDir(c *cli.Context, dir, app, manifest, description string, output io.WriteCloser) (string, string, error) {
	err := warnUnignoredEnv(dir)
	if err != nil {
		return "", "", err
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	output.Write([]byte("Creating tarball... "))

	tar, err := createTarball(dir)
	if err != nil {
		return "", "", err
	}

	output.Write([]byte("OK\n"))

	opts := client.CreateBuildSourceOptions{
		Cache:       !c.Bool("no-cache"),
		Config:      manifest,
		Description: description,
		Progress:    progress("Uploading: ", "Starting build... ", output),
	}

	build, err := rackClient(c).CreateBuildSource(app, bytes.NewReader(tar), opts)
	if err != nil {
		return "", "", err
	}

	return finishBuild(c, app, build, output)
}

func executeBuildURL(c *cli.Context, url, app, manifest, description string, output io.WriteCloser) (string, string, error) {
	cache := !c.Bool("no-cache")

	output.Write([]byte("Starting build... "))

	build, err := rackClient(c).CreateBuildUrl(app, url, cache, manifest, description)
	if err != nil {
		return "", "", err
	}

	return finishBuild(c, app, build, output)
}

func createTarball(base string) ([]byte, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	sym, err := filepath.EvalSymlinks(base)
	if err != nil {
		return nil, err
	}

	err = os.Chdir(sym)
	if err != nil {
		return nil, err
	}

	var includes = []string{"."}
	var excludes []string

	dockerIgnorePath := path.Join(sym, ".dockerignore")
	dockerIgnore, err := os.Open(dockerIgnorePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		//There is no docker ignore
		excludes = make([]string, 0)
	} else {
		excludes, err = dockerignore.ReadAll(dockerIgnore)
		if err != nil {
			return nil, err
		}
	}

	// If .dockerignore mentions .dockerignore or the Dockerfile
	// then make sure we send both files over to the daemon
	// because Dockerfile is, obviously, needed no matter what, and
	// .dockerignore is needed to know if either one needs to be
	// removed.  The deamon will remove them for us, if needed, after it
	// parses the Dockerfile.
	keepThem1, _ := fileutils.Matches(".dockerignore", excludes)
	keepThem2, _ := fileutils.Matches("Dockerfile", excludes)
	if keepThem1 || keepThem2 {
		includes = append(includes, ".dockerignore", "Dockerfile")
	}

	// if err := builder.ValidateContextDirectory(contextDirectory, excludes); err != nil {
	// 	return nil, fmt.Errorf("Error checking context is accessible: '%s'. Please check permissions and try again.", err)
	// }

	options := &archive.TarOptions{
		Compression:     archive.Gzip,
		ExcludePatterns: excludes,
		IncludeFiles:    includes,
	}

	out, err := archive.TarWithOptions(sym, options)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadAll(out)
	if err != nil {
		return nil, err
	}

	err = os.Chdir(cwd)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func finishBuild(c *cli.Context, app string, build *client.Build, output io.WriteCloser) (string, string, error) {
	if build.Id == "" {
		return "", "", fmt.Errorf("unable to fetch build id")
	}

	output.Write([]byte("OK\n"))

	err := rackClient(c).StreamBuildLogs(app, build.Id, output)
	if err != nil {
		return "", "", err
	}

	release, err := waitForBuild(c, app, build.Id)
	if err != nil {
		return "", "", err
	}

	return build.Id, release, nil
}

func waitForBuild(c *cli.Context, app, id string) (string, error) {
	for {
		build, err := rackClient(c).GetBuild(app, id)
		if err != nil {
			return "", err
		}

		switch build.Status {
		case "complete":
			return build.Release, nil
		case "error":
			return "", fmt.Errorf("%s build failed", app)
		case "failed":
			return "", fmt.Errorf("%s build failed", app)
		case "timeout":
			return "", fmt.Errorf("%s build timed out", app)
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("can't get here")
}

func warnUnignoredEnv(dir string) error {
	hasDockerIgnore := false
	hasDotEnv := false
	warn := false

	if _, err := os.Stat(".env"); err == nil {
		hasDotEnv = true
	}

	if _, err := os.Stat(".dockerignore"); err == nil {
		hasDockerIgnore = true
	}

	if !hasDockerIgnore && hasDotEnv {
		warn = true
	} else if hasDockerIgnore && hasDotEnv {
		lines, err := readDockerIgnore(dir)
		if err != nil {
			return err
		}

		if len(lines) == 0 {
			warn = true
		} else {
			warn = true
			for _, line := range lines {
				if line == ".env" {
					warn = false
					break
				}
			}
		}
	}
	if warn {
		stdcli.Warn("You have a .env file that is not in your .dockerignore, you may be leaking secrets")
	}
	return nil
}
