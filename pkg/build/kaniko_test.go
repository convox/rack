package build

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/convox/exec"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLoginForKaniko(t *testing.T) {
	// Set up test directory
	testDir := t.TempDir()
	dockerDir := filepath.Join(testDir, ".docker")
	err := os.MkdirAll(dockerDir, 0755)
	require.NoError(t, err)

	// Save original and patch for test
	originalDockerConfigDir := "/kaniko/.docker"
	configPath := filepath.Join(dockerDir, "config.json")

	// Monkey patch osMkdirAll for this test to redirect to our test directory
	originalMkdirAll := osMkdirAll
	osMkdirAll = func(path string, perm os.FileMode) error {
		if path == originalDockerConfigDir {
			return os.MkdirAll(dockerDir, perm)
		}
		return originalMkdirAll(path, perm)
	}
	defer func() { osMkdirAll = originalMkdirAll }()

	// Monkey patch os.WriteFile for this test
	originalWriteFile := osWriteFile
	osWriteFile = func(name string, data []byte, perm os.FileMode) error {
		if filepath.Dir(name) == originalDockerConfigDir {
			return os.WriteFile(configPath, data, perm)
		}
		return originalWriteFile(name, data, perm)
	}
	defer func() { osWriteFile = originalWriteFile }()

	// Create test auth data
	auth := map[string]struct {
		Username string
		Password string
	}{
		"registry.example.com": {
			Username: "testuser",
			Password: "testpass",
		},
		"index.docker.io": {
			Username: "dockeruser",
			Password: "dockerpass",
		},
	}

	// Run the function under test
	err = loginForKaniko(auth)
	assert.NoError(t, err, "loginForKaniko should not return an error")

	// Check that the docker config file was created correctly
	assert.FileExists(t, configPath, "config.json should be created")

	configData, err := os.ReadFile(configPath)
	assert.NoError(t, err, "Should be able to read config file")
	assert.Contains(t, string(configData), "registry.example.com", "Config should contain registry URL")
	assert.Contains(t, string(configData), "index.docker.io", "Config should contain Docker Hub URL")
}

func TestBuildWithKaniko(t *testing.T) {
	mockExec := new(exec.MockInterface)
	mockProv := new(structs.MockProvider)

	// Setup mock behavior for successful build
	expectedArgs := []interface{}{
		mock.Anything,
		"/kaniko/executor",
		"--dockerfile",
		"/path/to/Dockerfile",
		"--context",
		"/path/to",
		"--destination",
		"example.com/testimage:tag",
		"--cache=true",
	}
	mockExec.On("Run", expectedArgs...).Return(nil)

	b := &Build{
		Exec:     mockExec,
		Provider: mockProv,
		Options: Options{
			App:    "testapp",
			Id:     "TESTBUILD",
			Method: "kaniko",
			Cache:  true,
		},
		writer: &bytes.Buffer{},
	}

	// Call the function under test
	err := b.buildWithKaniko("/path/to", "/path/to/Dockerfile", "example.com/testimage:tag", map[string]string{})
	assert.NoError(t, err, "buildWithKaniko should not return an error")

	// Verify mock was called
	mockExec.AssertExpectations(t)
}

func TestBuildWithKaniko_WithError(t *testing.T) {
	mockExec := new(exec.MockInterface)
	mockProv := new(structs.MockProvider)

	// Setup mock behavior for successful build
	expectedArgs := []interface{}{
		mock.Anything,
		"/kaniko/executor",
		"--dockerfile",
		"/path/to/Dockerfile",
		"--context",
		"/path/to",
		"--destination",
		"example.com/errorimage:tag",
		"--cache=true",
	}
	mockExec.On("Run", expectedArgs...).Return(errors.New("executor error"))

	b := &Build{
		Exec:     mockExec,
		Provider: mockProv,
		Options: Options{
			App:    "testapp",
			Id:     "TESTBUILD",
			Method: "kaniko",
			Cache:  true,
		},
		writer: &bytes.Buffer{},
	}
	// Call the function with error case
	err := b.buildWithKaniko("/path/to", "/path/to/Dockerfile", "example.com/errorimage:tag", map[string]string{})
	assert.Error(t, err, "buildWithKaniko should return an error when executor fails")
}

func TestBuildWithKanikoWithBuildArgs(t *testing.T) {
	mockExec := new(exec.MockInterface)
	mockProv := new(structs.MockProvider)

	// Create a temp Dockerfile for build args test
	tmpDir := t.TempDir()
	dockerfile := filepath.Join(tmpDir, "Dockerfile")
	err := os.WriteFile(dockerfile, []byte(`
FROM alpine:latest
ARG VERSION
ARG BUILD_DATE
LABEL version=$VERSION
LABEL build_date=$BUILD_DATE
`), 0644)
	require.NoError(t, err)

	// Capture buildArgs call
	expectedArgs := []interface{}{
		mock.Anything,
		"/kaniko/executor",
		"--dockerfile",
		fmt.Sprintf("%s/Dockerfile", tmpDir),
		"--context",
		tmpDir,
		"--destination",
		"example.com/testimage:tag",
		"--cache=true",
		"--build-arg",
		"VERSION=1.2.3",
		"--build-arg",
		"BUILD_DATE=2025-04-15",
	}
	mockExec.On("Run", expectedArgs...).Return(nil)

	b := &Build{
		Exec:     mockExec,
		Provider: mockProv,
		Options: Options{
			App:    "testapp",
			Id:     "TESTBUILD",
			Method: "kaniko",
			Cache:  true,
		},
		writer: &bytes.Buffer{},
	}

	// Setup environment with build args
	env := map[string]string{
		"VERSION":    "1.2.3",
		"BUILD_DATE": "2025-04-15",
		"IGNORED":    "this-should-not-be-included",
	}

	// Call the function under test
	err = b.buildWithKaniko(tmpDir, dockerfile, "example.com/testimage:tag", env)
	assert.NoError(t, err, "buildWithKaniko should not return an error")

	// Verify mock was called
	mockExec.AssertExpectations(t)
}

func TestKanikoNoCache(t *testing.T) {
	mockExec := new(exec.MockInterface)
	mockProv := new(structs.MockProvider)

	// Setup mock for no cache build
	expectedArgs := []interface{}{
		mock.Anything,
		"/kaniko/executor",
		"--dockerfile",
		"/path/to/Dockerfile",
		"--context",
		"/path/to",
		"--destination",
		"example.com/testimage:tag",
		"--cache=false",
	}
	mockExec.On("Run", expectedArgs...).Return(nil)

	b := &Build{
		Exec:     mockExec,
		Provider: mockProv,
		Options: Options{
			App:    "testapp",
			Id:     "TESTBUILD",
			Method: "kaniko",
			Cache:  false,
		},
		writer: &bytes.Buffer{},
	}

	// Call the function under test
	err := b.buildWithKaniko("/path/to", "/path/to/Dockerfile", "example.com/testimage:tag", map[string]string{})
	assert.NoError(t, err, "buildWithKaniko should not return an error")

	// Verify mock was called
	mockExec.AssertExpectations(t)
}

func TestKanikoHelperFunctions(t *testing.T) {
	mockExec := new(exec.MockInterface)
	mockProv := new(structs.MockProvider)

	b := &Build{
		Exec:     mockExec,
		Provider: mockProv,
		Options: Options{
			App:    "testapp",
			Id:     "TESTBUILD",
			Method: "kaniko",
			Push:   "example.com/repo",
		},
		writer: &bytes.Buffer{},
	}

	// Test pullWithKaniko
	err := b.pullWithKaniko("example.com/testimage:tag")
	assert.NoError(t, err, "pullWithKaniko should not return an error")

	// Test pushWithKaniko
	err = b.pushWithKaniko("example.com/testimage:tag")
	assert.NoError(t, err, "pushWithKaniko should not return an error")

	// Test tagWithKaniko
	err = b.tagWithKaniko("example.com/testimage:tag", "example.com/repo/testimage:tag")
	assert.NoError(t, err, "tagWithKaniko should not return an error")

	// Test injectConvoxEnvWithKaniko
	err = b.injectConvoxEnvWithKaniko("example.com/testimage:tag")
	assert.NoError(t, err, "injectConvoxEnvWithKaniko should not return an error")
}
