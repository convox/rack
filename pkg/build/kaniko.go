package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Kaniko image to use
	kanikoImage = "gcr.io/kaniko-project/executor:v1.23.2"
)

// buildWithKaniko builds a Docker image using Google's Kaniko tool
func (bb *Build) buildWithKaniko(path, dockerfile string, tag string, env map[string]string) error {
	bb.Printf("Building with Kaniko\n")

	// Pass relative path to the Dockerfile
	dfRel, err := filepath.Rel(path, dockerfile)
	if err != nil {
		return fmt.Errorf("error making Dockerfile path relative: %w", err)
	}

	args := []string{
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/workspace", path),
		"-v", "/kaniko/.docker:/kaniko/.docker",
		kanikoImage,
		"--dockerfile",
		dfRel,
		"--context",
		"dir:///workspace",
		"--cache=false",
		"--no-push",
	}
	if bb.Cache {
		bb.Printf("Cache not enabled for Kaniko build\n")
	}

	// Handle build args
	ba, err := bb.buildArgs(dockerfile, env)
	if err != nil {
		if os.IsNotExist(err) {
			ba = []string{}
		} else {
			return err
		}
	}
	args = append(args, ba...)

	bb.Printf("Running: kanikoBuildExecutor %s\n", strings.Join(args, " "))

	// Execute Kaniko build
	err = bb.Exec.Run(bb.writer, "docker", args...)
	if err != nil {
		return fmt.Errorf("kaniko build failed: %v", err)
	}

	return nil
}
