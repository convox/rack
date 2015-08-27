package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/Godeps/_workspace/src/golang.org/x/net/websocket"
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
	// stdcli.RegisterCommand(cli.Command{
	// 	Name:        "build",
	// 	Description: "",
	// 	Usage:       "",
	// 	Action:      cmdBuild,
	// })
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
		stdcli.Error(err)
	}

	fmt.Print("Uploading... ")

	tar, err := createTarball(dir)

	if err != nil {
		return "", err
	}

	fmt.Println("OK")

	build, err := postBuild(tar, app)

	if err != nil {
		return "", err
	}

	err = streamBuild(app, build, 0)

	if err != nil {
		fmt.Printf("%+v\n", err)
		return "", err
	}

	release, err := waitForBuild(app, build)

	if err != nil {
		return "", err
	}

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

	if res.StatusCode/100 > 3 {
		return "", fmt.Errorf(string(data))
	}

	return string(data), nil
}

func streamBuild(app, build string, offset int) error {
	host, password, err := currentLogin()

	if err != nil {
		stdcli.Error(err)
		return err
	}

	origin := fmt.Sprintf("https://%s", host)
	url := fmt.Sprintf("wss://%s/apps/%s/builds/%s/logs", host, app, build)

	config, err := websocket.NewConfig(url, origin)

	if err != nil {
		stdcli.Error(err)
		return err
	}

	userpass := fmt.Sprintf("convox:%s", password)
	userpass_encoded := base64.StdEncoding.EncodeToString([]byte(userpass))

	config.Header.Add("Authorization", fmt.Sprintf("Basic %s", userpass_encoded))

	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	ws, err := websocket.DialConfig(config)

	if err != nil {
		stdcli.Error(err)
		return err
	}

	defer ws.Close()

	var message []byte

	lineno := 0

	for {
		err := websocket.Message.Receive(ws, &message)

		if err == io.EOF {
			status, err := buildStatus(app, build)

			if err != nil {
				stdcli.Error(err)
				return err
			}

			if status == "building" {
				continue
			} else {
				return nil
			}
		}

		if err != nil {
			// fmt.Fprintf(os.Stderr, "ws %s, retrying...\n", err.Error())
			return streamBuild(app, build, lineno)
		}

		if lineno >= offset {
			fmt.Print(string(message))
		}

		lineno += 1
	}

	return nil
}

func buildStatus(app, build string) (string, error) {
	var b Build

	data, err := ConvoxGet(fmt.Sprintf("/apps/%s/builds/%s", app, build))

	if err != nil {
		return "", err
	}

	err = json.Unmarshal(data, &b)

	if err != nil {
		return "", err
	}

	return b.Status, nil
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

func createTarball(base string) ([]byte, error) {
	cwd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	err = os.Chdir(base)

	if err != nil {
		return nil, err
	}

	cmd := exec.Command("tar", "cz", ".")

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
