package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var template = `{
  "description": "Convox App",
  "variables": {
    "CLUSTER": null,
    "APP": null,
    "REPO": null,
    "APPCONF": null,
    "BASE_AMI": "ami-447b042c",
    "AWS_REGION": "us-east-1",
    "AWS_ACCESS": "{{env \"AWS_ACCESS\"}}",
    "AWS_SECRET": "{{env \"AWS_SECRET\"}}"
  },
  "builders": [
    {
      "type": "amazon-ebs",
      "region": "{{user \"AWS_REGION\"}}",
      "access_key": "{{user \"AWS_ACCESS\"}}",
      "secret_key": "{{user \"AWS_SECRET\"}}",
      "source_ami": "{{user \"BASE_AMI\"}}",
      "instance_type": "t2.micro",
      "ssh_username": "ubuntu",
      "ami_name": "{{user \"CLUSTER\"}}-{{user \"APP\"}}-{{timestamp}}",
      "tags": {
        "type": "app",
        "cluster": "{{user \"CLUSTER\"}}",
        "app": "{{user \"APP\"}}"
      }
    }
  ],
  "provisioners": [
    {
      "type": "shell",
      "execute_command": "chmod +x {{ .Path }}; {{ .Vars }} sudo -E -S sh '{{ .Path }}'",
      "inline": [
        "mkdir /build",
        "chown ubuntu:ubuntu /build",
        "mkdir /var/app"
      ]
    },
    {
      "type": "file",
      "source": "{{user \"REPO\"}}/",
      "destination": "/build"
    },
    {
      "type": "shell",
      "inline": [
        "cd /build",
        "/usr/local/bin/fig -p app build"
      ]
    },
    {
      "type": "file",
      "source": "{{user \"APPCONF\"}}",
      "destination": "/tmp/app.conf"
    },
    {
      "type": "shell",
      "execute_command": "chmod +x {{ .Path }}; {{ .Vars }} sudo -E -S sh '{{ .Path }}'",
      "inline": [
        "rm -rf /build",
        "mv /tmp/app.conf /etc/init/app.conf"
      ]
    }
  ]
}`

var appconf = `
start on runlevel [2345]
stop on runlevel [!2345]

respawn

pre-start script
  curl http://169.254.169.254/latest/user-data | jq -r ".env[]" > /var/app/env
  curl http://169.254.169.254/latest/user-data | jq -r ".process" > /var/app/process
  curl http://169.254.169.254/latest/user-data | jq -r '.ports | map("-p \(.):\(.)")[]' | tr '\n' ' ' > /var/app/ports
end script

script
  docker run -a STDOUT -a STDERR --sig-proxy $(cat /var/app/ports) --env-file /var/app/env app_$(cat /var/app/process)
end script
`

type Builder struct {
	AwsRegion string
	AwsAccess string
	AwsSecret string
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(repo, cluster, app string) error {
	dir, err := ioutil.TempDir("", "repo")

	if err != nil {
		return err
	}

	clone := filepath.Join(dir, "clone")

	cmd := exec.Command("git", "clone", repo, clone)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	tf := filepath.Join(dir, "packer.json")
	ioutil.WriteFile(tf, []byte(template), 0644)

	ac := filepath.Join(dir, "app.conf")
	ioutil.WriteFile(ac, []byte(appconf), 0644)

	cmd = exec.Command("packer", "build", "-machine-readable", "-var", "CLUSTER="+cluster, "-var", "APP="+app, "-var", "REPO="+clone, "-var", "APPCONF="+ac, tf)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	cmd.Start()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ",", 5)

		if parts[2] == "ui" && parts[3] == "say" {
			fmt.Println(parts[4])
		}

		if parts[1] == "amazon-ebs" && parts[2] == "artifact" && parts[3] == "0" && parts[4] == "id" {
			createRelease(cluster, app, parts[4])
		}
	}

	cmd.Wait()

	return nil
}

func createRelease(cluster, app, ami string) {
	fmt.Printf("creating release cluster=%s app=%s ami=%s\n", cluster, app, ami)
}
