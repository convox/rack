package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/fileutils"
	"gopkg.in/urfave/cli.v1"
)

var (
	IndexOperationConcurrency = 128

	buildCreateFlags = []cli.Flag{
		appFlag,
		cli.BoolFlag{
			Name:  "no-cache",
			Usage: "pull fresh image dependencies",
		},
		cli.BoolFlag{
			Name:  "incremental",
			Usage: "use incremental build",
		},
		cli.StringFlag{
			Name:  "file, f",
			Value: "docker-compose.yml",
			Usage: "path to an alternate docker compose manifest file",
		},
		cli.StringFlag{
			Name:  "description",
			Value: "",
			Usage: "description of the build",
		},
	}
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "build",
		Description: "create a new build",
		Usage:       "",
		Action:      cmdBuildsCreate,
		Flags:       buildCreateFlags,
	})
	stdcli.RegisterCommand(cli.Command{
		Name:        "builds",
		Description: "manage an app's builds",
		Usage:       "",
		Action:      cmdBuilds,
		Flags:       []cli.Flag{appFlag},
		Subcommands: []cli.Command{
			{
				Name:        "create",
				Description: "create a new build",
				Usage:       "",
				Action:      cmdBuildsCreate,
				Flags:       buildCreateFlags,
			},
			{
				Name:        "copy",
				Description: "copy a build to an app",
				Usage:       "<ID> <app>",
				Action:      cmdBuildsCopy,
				Flags: []cli.Flag{
					appFlag,
					cli.BoolFlag{
						Name:  "promote",
						Usage: "promote the release after copy",
					},
				},
			},
			{
				Name:        "info",
				Description: "print output for a build",
				Usage:       "<ID>",
				Action:      cmdBuildsInfo,
				Flags:       []cli.Flag{appFlag},
			},
			{
				Name:        "delete",
				Description: "Archive a build and its artifacts",
				Usage:       "<ID>",
				Action:      cmdBuildsDelete,
				Flags:       []cli.Flag{appFlag},
			},
		},
	})
}

func cmdBuilds(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) > 0 {
		return stdcli.ExitError(fmt.Errorf("`convox builds` does not take arguments. Perhaps you meant `convox builds create`?"))
	}

	if c.Bool("help") {
		stdcli.Usage(c, "")
		return nil
	}

	builds, err := rackClient(c).GetBuilds(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	t := stdcli.NewTable("ID", "STATUS", "RELEASE", "STARTED", "ELAPSED", "DESC")

	for _, build := range builds {
		started := humanizeTime(build.Started)
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
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)
	if err != nil {
		return stdcli.ExitError(err)
	}

	a, err := rackClient(c).GetApp(app)
	if err != nil {
		return stdcli.ExitError(err)
	}

	switch a.Status {
	case "creating":
		return stdcli.ExitError(fmt.Errorf("app is still creating: %s", app))
	case "running", "updating":
	default:
		return stdcli.ExitError(fmt.Errorf("unable to build app: %s", app))
	}

	if len(c.Args()) > 0 {
		dir = c.Args()[0]
	}

	release, err := executeBuild(c, dir, app, c.String("file"), c.String("description"))
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("Release: %s\n", release)
	return nil
}

func cmdBuildsDelete(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "delete")
		return nil
	}

	build := c.Args()[0]

	b, err := rackClient(c).DeleteBuild(app, build)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Printf("Deleted %s\n", b.Id)
	return nil
}

func cmdBuildsInfo(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) != 1 {
		stdcli.Usage(c, "info")
		return nil
	}

	build := c.Args()[0]

	b, err := rackClient(c).GetBuild(app, build)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println(b.Logs)
	return nil
}

func cmdBuildsCopy(c *cli.Context) error {
	_, app, err := stdcli.DirApp(c, ".")
	if err != nil {
		return stdcli.ExitError(err)
	}

	if len(c.Args()) != 2 {
		stdcli.Usage(c, "copy")
		return nil
	}

	build := c.Args()[0]
	destApp := c.Args()[1]

	fmt.Print("Copying build... ")

	b, err := rackClient(c).CopyBuild(app, build, destApp)
	if err != nil {
		return stdcli.ExitError(err)
	}

	fmt.Println("OK")

	releaseId, err := finishBuild(c, destApp, b)
	if err != nil {
		return stdcli.ExitError(err)
	}

	if releaseId != "" {
		if c.Bool("promote") {
			fmt.Printf("Promoting %s %s... ", destApp, releaseId)

			_, err = rackClient(c).PromoteRelease(destApp, releaseId)
			if err != nil {
				return stdcli.ExitError(err)
			}

			fmt.Println("OK")
		} else {
			fmt.Printf("To deploy this copy run `convox releases promote %s --app %s`\n", releaseId, destApp)
		}
	}

	return nil
}

func executeBuild(c *cli.Context, source, app, manifest, description string) (string, error) {
	u, _ := url.Parse(source)

	switch u.Scheme {
	case "http", "https":
		return executeBuildUrl(c, source, app, manifest, description)
	default:
		if c.Bool("incremental") {
			return executeBuildDirIncremental(c, source, app, manifest, description)
		} else {
			return executeBuildDir(c, source, app, manifest, description)
		}
	}

	return "", fmt.Errorf("unreachable")
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

func uploadIndex(c *cli.Context, index client.Index) error {
	missing, err := rackClient(c).IndexMissing(index)

	if err != nil {
		return err
	}

	total := 0

	for _, m := range missing {
		total += index[m].Size
	}

	bar := pb.New(total)

	bar.Prefix("Uploading changes... ")
	bar.SetMaxWidth(40)
	bar.SetUnits(pb.U_BYTES)

	if total == 0 {
		fmt.Println("NONE")
	} else {
		bar.Start()
	}

	inch := make(chan string)
	errch := make(chan error)

	for i := 1; i < IndexOperationConcurrency; i++ {
		go uploadItems(c, index, bar, inch, errch)
	}

	go func() {
		for _, hash := range missing {
			inch <- hash
		}
	}()

	for range missing {
		if err := <-errch; err != nil {
			return err
		}
	}

	close(inch)

	if total > 0 {
		bar.Finish()
	}

	return nil
}

func uploadItem(c *cli.Context, hash string, item client.IndexItem, bar *pb.ProgressBar, ch chan error) {
	data, err := ioutil.ReadFile(item.Name)

	if err != nil {
		ch <- err
		return
	}

	for i := 0; i < 3; i++ {
		err = rackClient(c).IndexUpload(hash, data)

		if err != nil {
			continue
		}

		bar.Add(item.Size)

		ch <- nil
		return
	}

	ch <- fmt.Errorf("max 3 retries on upload")
	return
}

func uploadItems(c *cli.Context, index client.Index, bar *pb.ProgressBar, inch chan string, errch chan error) {
	for hash := range inch {
		uploadItem(c, hash, index[hash], bar, errch)
	}
}

func executeBuildDirIncremental(c *cli.Context, dir, app, manifest, description string) (string, error) {
	system, err := rackClient(c).GetSystem()

	if err != nil {
		return "", err
	}

	// if the rack doesnt support incremental builds then fall back
	if system.Version < "20160226234213" {
		return executeBuildDir(c, dir, app, manifest, description)
	}

	cache := !c.Bool("no-cache")

	dir, err = filepath.Abs(dir)

	if err != nil {
		return "", err
	}

	fmt.Printf("Analyzing source... ")

	index, err := createIndex(dir)

	if err != nil {
		return "", err
	}

	fmt.Println("OK")

	fmt.Printf("Uploading changes... ")

	err = uploadIndex(c, index)

	if err != nil {
		return "", err
	}

	fmt.Printf("Starting build... ")

	build, err := rackClient(c).CreateBuildIndex(app, index, cache, manifest, description)

	if err != nil {
		return "", err
	}

	fmt.Println("OK")

	return finishBuild(c, app, build)
}

func executeBuildDir(c *cli.Context, dir, app, manifest, description string) (string, error) {
	err := warnUnignoredEnv(dir)
	if err != nil {
		return "", err
	}

	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	fmt.Print("Creating tarball... ")

	tar, err := createTarball(dir)

	if err != nil {
		return "", err
	}

	fmt.Println("OK")

	cache := !c.Bool("no-cache")

	fmt.Print("Uploading... ")

	build, err := rackClient(c).CreateBuildSource(app, tar, cache, manifest, description)

	if err != nil {
		return "", err
	}

	fmt.Println("OK")

	return finishBuild(c, app, build)
}

func executeBuildUrl(c *cli.Context, url, app, manifest, description string) (string, error) {
	cache := !c.Bool("no-cache")

	build, err := rackClient(c).CreateBuildUrl(app, url, cache, manifest, description)

	if err != nil {
		return "", err
	}

	return finishBuild(c, app, build)
}

func createTarball(base string) ([]byte, error) {
	cwd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(base)

	if err != nil {
		return nil, err
	}

	args := []string{"cz"}

	// If .dockerignore exists, use it to exclude files from the tarball
	if _, err = os.Stat(".dockerignore"); err == nil {
		args = append(args, "--exclude-from", ".dockerignore")
	}

	args = append(args, ".")

	cmd := exec.Command("tar", args...)

	out, err := cmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	cmd.Start()

	bytes, err := ioutil.ReadAll(out)

	if err != nil {
		return nil, err
	}

	err = cmd.Wait()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(cwd)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func finishBuild(c *cli.Context, app string, build *client.Build) (string, error) {
	if build.Id == "" {
		return "", fmt.Errorf("unable to fetch build id")
	}

	reader, writer := io.Pipe()
	go io.Copy(os.Stdout, reader)
	err := rackClient(c).StreamBuildLogs(app, build.Id, writer)

	if err != nil {
		return "", err
	}

	release, err := waitForBuild(c, app, build.Id)

	if err != nil {
		return "", err
	}

	return release, nil
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
		fmt.Println("WARNING: You have a .env file that is not in your .dockerignore, you may be leaking secrets")
	}
	return nil
}
