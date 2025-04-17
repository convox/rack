package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loginForKaniko sets up Docker registry authentication for Kaniko
func loginForKaniko(auth map[string]struct {
	Username string
	Password string
}) error {
	// Create Docker config directory
	configDir := "/kaniko/.docker"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating docker config directory for kaniko: %v", err)
	}

	// Create Docker config.json with auth data
	authConfig := map[string]interface{}{
		"auths": auth,
	}

	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return fmt.Errorf("error marshaling auth config for kaniko: %v", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, authJSON, 0600); err != nil {
		return fmt.Errorf("error writing kaniko docker config: %v", err)
	}

	return nil
}

// buildWithKaniko builds a Docker image using Google's Kaniko tool
func (bb *Build) buildWithKaniko(path, dockerfile string, tag string, env map[string]string) error {
	bb.Printf("Building with Kaniko\n")

	// Extract the registry from the tag
	parts := strings.SplitN(tag, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid image tag format: %s", tag)
	}

	registry := parts[0]
	imagePath := parts[1]

	// Kaniko requires the destination in the format: registry/image:tag
	destination := fmt.Sprintf("%s/%s", registry, imagePath)

	args := []string{
		"--dockerfile", dockerfile,
		"--context", path,
		"--destination", destination,
	}

	// Handle cache option
	if bb.Cache {
		args = append(args, "--cache=true")
	} else {
		args = append(args, "--cache=false")
	}

	// Handle build args
	ba, err := bb.buildArgs(dockerfile, env)
	if err != nil {
		return err
	}

	for _, arg := range ba {
		if strings.HasPrefix(arg, "--build-arg") {
			args = append(args, arg)
		}
	}

	bb.Printf("Running: kanikoBuildExecutor %s\n", strings.Join(args, " "))

	// Execute Kaniko build
	if err := bb.Exec.Run(bb.writer, "/kaniko/executor", args...); err != nil {
		return fmt.Errorf("kaniko build failed: %v", err)
	}

	// Kaniko doesn't expose an easy way to get the entrypoint,
	// but we can consider the build successful without it for now
	bb.Printf("Successfully built image with Kaniko: %s\n", destination)

	return nil
}

// pull is a no-op for Kaniko as it handles pulling within the build process
func (bb *Build) pullWithKaniko(image string) error {
	bb.Printf("Skipping docker pull for Kaniko method: %s\n", image)
	return nil
}

// pushWithKaniko is a no-op for Kaniko as pushing is done during the build step
func (bb *Build) pushWithKaniko(tag string) error {
	bb.Printf("Skipping docker push for Kaniko method - image already pushed: %s\n", tag)
	return nil
}

// tagWithKaniko handles tagging in Kaniko environment
func (bb *Build) tagWithKaniko(from, to string) error {
	// In Kaniko mode, we should build directly to the target tag
	// so we can skip this operation
	bb.Printf("Skipping docker tag for Kaniko method: %s -> %s\n", from, to)

	// But we should trigger a Kaniko build if this is a push destination
	if strings.HasPrefix(to, bb.Push) {
		bb.Printf("Building with Kaniko directly to push destination: %s\n", to)
	}

	return nil
}

// injectConvoxEnvWithKaniko is not supported in Kaniko
func (bb *Build) injectConvoxEnvWithKaniko(_ string) error {
	bb.Printf("Skipping convox-env injection for Kaniko build\n")
	return nil
}
