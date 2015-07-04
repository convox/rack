package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

type Build struct {
	Id  string
	App string

	Logs    string
	Release string
	Status  string

	Started time.Time
	Ended   time.Time
}

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "build",
		Description: "build an app for local development",
		Usage:       "<directory>",
		Action:      cmdBuild,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdBuild(c *cli.Context) {
	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, app, err := stdcli.DirApp(c, wd)

	if err != nil {
		stdcli.Error(err)
		return
	}

	_, err = executeBuild(dir, app)

	if err != nil {
		stdcli.Error(err)
		return
	}
}

func executeBuild(dir string, app string) (string, error) {
	dir, err := filepath.Abs(dir)

	if err != nil {
		panic(err)
	}

	tar, err := createTarball(dir)

	if err != nil {
		return "", err
	}

	fmt.Print("Building")

	build, err := postBuild(tar, app)

	if err != nil {
		return "", err
	}

	release, err := waitForBuild(app, build)

	if err != nil {
		return "", err
	}

	fmt.Println(" done")

	return release, nil
}

func postBuild(tar []byte, app string) (string, error) {
	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("source", "source.tgz")

	if err != nil {
		return "", err
	}

	_, err = io.Copy(part, bytes.NewReader(tar))

	if err != nil {
		return "", err
	}

	err = writer.Close()

	if err != nil {
		return "", err
	}

	req, err := convoxRequest("POST", fmt.Sprintf("/apps/%s/build", app), body)

	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := convoxClient().Do(req)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func waitForBuild(app, id string) (string, error) {
	var build Build

	for {
		data, err := ConvoxGet(fmt.Sprintf("/apps/%s/builds/%s", app, id))

		if err != nil {
			return "", err
		}

		err = json.Unmarshal(data, &build)

		if err != nil {
			return "", err
		}

		fmt.Print(".")

		switch build.Status {
		case "complete":
			return build.Release, nil
		case "error":
			return "", fmt.Errorf("build failed")
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("can't get here")
}

func createTarball(base string) ([]byte, error) {
	buf := new(bytes.Buffer)

	gw := gzip.NewWriter(buf)

	tw := tar.NewWriter(gw)

	err := walkToTar(base, ".", tw)

	if err != nil {
		return nil, err
	}

	err = tw.Close()

	if err != nil {
		return nil, err
	}

	err = gw.Close()

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func walkToTar(base, path string, tw *tar.Writer) error {
	abs, err := filepath.Abs(filepath.Join(base, path))

	if err != nil {
		return err
	}

	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(base, path)

		if err != nil {
			return err
		}

		if info != nil && (info.Mode()&os.ModeSymlink == os.ModeSymlink) {
			link, err := filepath.EvalSymlinks(path)

			if err != nil {
				return err
			}

			walkToTar(link, rel, tw)

			return nil
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(path)

		if err != nil {
			return err
		}

		defer file.Close()

		header := &tar.Header{
			Name:    rel,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		err = tw.WriteHeader(header)

		if err != nil {
			return err
		}

		_, err = io.Copy(tw, file)

		if err != nil {
			return err
		}

		return nil
	})

	return err
}
