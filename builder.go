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

func (b *Builder) Build(repo, name, ref string) error {
	ami, err := buildAmi(repo, name, ref)
	fmt.Printf("ami %+v\n", ami)
	fmt.Printf("err %+v\n", err)

	if err != nil {
		return err
	}

	// cloudformation

	return nil
}

func buildAmi(repo, name, ref string) (string, error) {
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

	data, err := ioutil.ReadFile(filepath.Join(clone, "docker-compose.yml"))

	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		fmt.Printf("manifest|%s\n", scanner.Text())
	}

	if err = writeFile(dir, "app.conf", nil); err != nil {
		return "", err
	}

	if err = writeFile(dir, "packer.json", nil); err != nil {
		return "", err
	}

	if err = writeFile(dir, "cloudwatch-logs.conf", map[string]string{"{{APP}}": name}); err != nil {
		return "", err
	}

	cmd = exec.Command("packer", "build", "-machine-readable", "-var", "NAME="+name, "-var", "SOURCE="+clone, "packer.json")
	cmd.Dir = dir

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
		case parts[3] == "error":
			fmt.Printf("error|%s\n", cleanupPackerString(parts[4]))
		case parts[2] == "ui" && parts[3] == "say":
			fmt.Printf("packer|%s\n", parts[4])
		case strings.HasPrefix(parts[4], "    amazon-ebs: Instance ID:"):
			fmt.Printf("packer|==> amazon-ebs: %s\n", strings.SplitN(parts[4], ": ", 2)[1])
		case parts[2] == "ui" && parts[3] == "message":
			mparts := strings.SplitN(parts[4], ": ", 2)
			if len(mparts) > 1 {
				fmt.Printf("build|%s\n", cleanupPackerString(mparts[1]))
			}
		case parts[1] == "amazon-ebs" && parts[2] == "artifact" && parts[3] == "0" && parts[4] == "id":
			if len(parts) > 5 {
				ami = strings.Split(parts[5], ":")[1]
				fmt.Printf("ami|%s\n", ami)
			}
		}
	}

	err = cmd.Wait()

	if err != nil {
		return "", err
	}

	return ami, nil
}

func cleanupPackerString(s string) string {
	return strings.Replace(s, "%!(PACKER_COMMA)", ",", -1)
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

func writeFile(dir, name string, replacements map[string]string) error {
	data, err := dataRaw(name)

	if err != nil {
		return err
	}

	sdata := string(data)

	if replacements != nil {
		for key, val := range replacements {
			sdata = strings.Replace(sdata, key, val, -1)
		}
	}

	return ioutil.WriteFile(filepath.Join(dir, name), []byte(sdata), 0644)
}
