package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type Builder struct {
	AwsRegion string
	AwsAccess string
	AwsSecret string
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(repo, app, ref string) error {
	ami, err := buildAmi(repo, app, ref)
	fmt.Printf("ami %+v\n", ami)
	fmt.Printf("err %+v\n", err)

	if err != nil {
		return err
	}

	// cloudformation

	return nil
}

func buildAmi(repo, app, ref string) (string, error) {
	dir, err := ioutil.TempDir("", "repo")

	if err != nil {
		return "", err
	}

	clone := filepath.Join(dir, "clone")

	cmd := exec.Command("git", "clone", repo, clone)
	cmd.Dir = dir
	err = cmd.Run()

	if err != nil {
		return "", err
	}

	if ref != "" {
		cmd = exec.Command("git", "checkout", ref)
		cmd.Dir = clone
		err = cmd.Run()

		if err != nil {
			return "", err
		}
	}

	data, err := ioutil.ReadFile(filepath.Join(clone, "fig.yml"))

	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		fmt.Printf("manifest|%s\n", scanner.Text())
	}

	output, err := dataRaw("packer.json")

	if err != nil {
		return "", err
	}

	packerjson := filepath.Join(dir, "packer.json")

	err = ioutil.WriteFile(packerjson, output, 0644)

	if err != nil {
		return "", err
	}

	output, err = dataRaw("upstart.conf")

	if err != nil {
		return "", err
	}

	appconf := filepath.Join(dir, "app.conf")

	err = ioutil.WriteFile(appconf, output, 0644)

	if err != nil {
		return "", err
	}

	cmd = exec.Command("packer", "build", "-machine-readable", "-var", "APP="+app, "-var", "SOURCE="+clone, "-var", "APPCONF="+appconf, packerjson)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return "", err
	}

	cmd.Start()

	scanner = bufio.NewScanner(stdout)

	ami := ""

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ",", 6)

		switch {
		case len(parts) < 5:
			fmt.Printf("unknown|%s\n", scanner.Text())
		case parts[2] == "ui" && parts[3] == "say":
			fmt.Printf("packer|%s\n", parts[4])
		case strings.HasPrefix(parts[4], "    amazon-ebs: Instance ID:"):
			fmt.Printf("packer|==> amazon-ebs: %s\n", strings.SplitN(parts[4], ": ", 2)[1])
		case parts[2] == "ui" && parts[3] == "message":
			fmt.Printf("build|%s\n", strings.Replace(strings.SplitN(parts[4], ": ", 2)[1], "%!(PACKER_COMMA)", ",", -1))
		case parts[1] == "amazon-ebs" && parts[2] == "artifact" && parts[3] == "0" && parts[4] == "id":
			ami = strings.Split(parts[5], ":")[1]
			fmt.Printf("ami|%s\n", ami)
		}
	}

	err = cmd.Wait()

	if err != nil {
		return "", err
	}

	return ami, nil
}

func dataRaw(path string) ([]byte, error) {
	return Asset(fmt.Sprintf("data/%s", path))
}

func dataTemplate(path, section string, object interface{}) ([]byte, error) {
	data, err := dataRaw(path)

	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(section).Parse(string(data))

	if err != nil {
		return nil, err
	}

	var output bytes.Buffer

	err = tmpl.Execute(&output, object)

	if err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}
