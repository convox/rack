package build

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	convexec "github.com/convox/exec"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuildWithKaniko(t *testing.T) {
	mockExec := new(convexec.MockInterface)
	mockProv := new(structs.MockProvider)

	// Setup mock behavior for successful build
	expectedArgs := []interface{}{
		mock.Anything,
		"docker",
		"run",
		"--rm",
		"-v", "/path/to:/workspace",
		"-v", "/kaniko/.docker:/kaniko/.docker",
		kanikoImage,
		"--dockerfile",
		"Dockerfile",
		"--context",
		"dir:///workspace",
		"--cache=false",
		"--no-push",
	}
	mockExec.On("Run", expectedArgs...).Return(nil)

	b := &Build{
		Exec:     mockExec,
		Provider: mockProv,
		Options: Options{
			App:    "testapp",
			Id:     "TESTBUILD",
			Method: "kaniko",
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
	mockExec := new(convexec.MockInterface)
	mockProv := new(structs.MockProvider)

	// Setup mock behavior for failure case
	expectedArgs := []interface{}{
		mock.Anything,
		"docker",
		"run",
		"--rm",
		"-v", "/path/to:/workspace",
		"-v", "/kaniko/.docker:/kaniko/.docker",
		kanikoImage,
		"--dockerfile",
		"Dockerfile",
		"--context",
		"dir:///workspace",
		"--cache=false",
		"--no-push",
	}
	mockExec.On("Run", expectedArgs...).Return(errors.New("executor error"))

	b := &Build{
		Exec:     mockExec,
		Provider: mockProv,
		Options: Options{
			App:    "testapp",
			Id:     "TESTBUILD",
			Method: "kaniko",
		},
		writer: &bytes.Buffer{},
	}
	// Call the function with error case
	err := b.buildWithKaniko("/path/to", "/path/to/Dockerfile", "example.com/errorimage:tag", map[string]string{})
	assert.Error(t, err, "buildWithKaniko should return an error when executor fails")
}

func TestBuildWithKanikoWithBuildArgs(t *testing.T) {
	mockExec := new(convexec.MockInterface)
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

	// Capture buildArgs call - include all arguments in proper format for the Run method
	expectedArgs := []interface{}{
		mock.Anything,
		"docker",
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/workspace", tmpDir),
		"-v", "/kaniko/.docker:/kaniko/.docker",
		kanikoImage,
		"--dockerfile",
		"Dockerfile",
		"--context",
		"dir:///workspace",
		"--cache=false",
		"--no-push",
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

func TestKanikoIntegration(t *testing.T) {
	// Check for Docker availability - this is an environment limitation so we still skip
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Cannot run Docker command: %v", err)
	}

	// Set up test directory with a simple Dockerfile
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	err := os.WriteFile(dockerfilePath, []byte(`
FROM alpine:latest
ARG TEST_ARG=default
RUN echo "Test arg is: $TEST_ARG" > /test.txt
CMD ["cat", "/test.txt"]
`), 0644)
	require.NoError(t, err)

	// Create a test file to include in the build context
	testFilePath := filepath.Join(tmpDir, "test-file.txt")
	err = os.WriteFile(testFilePath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create directories for mounting volumes
	kanikoDir := filepath.Join(tmpDir, "kaniko")
	err = os.MkdirAll(filepath.Join(kanikoDir, ".docker"), 0755)
	require.NoError(t, err)

	// Configure a temporary tag for the built image
	testRegistry := "localhost"
	testImage := "test-kaniko-integration"
	testTag := fmt.Sprintf("v%d", time.Now().Unix()) // Use unique tag
	fullImageTag := fmt.Sprintf("%s/%s:%s", testRegistry, testImage, testTag)

	// Create a buffer to capture execution output
	var outputBuffer bytes.Buffer

	// Create build instance for testing with the real exec implementation
	realExec := &convexec.Exec{}
	b := &Build{
		Exec: realExec,
		Options: Options{
			App:    "test-app",
			Id:     "TEST123",
			Method: "kaniko",
		},
		writer: &outputBuffer,
	}

	defer func() {
		// output the buffer to the test log
		fmt.Printf("Kaniko output: %s\n", outputBuffer.String())
	}()

	// Set up environment variables with build arguments
	env := map[string]string{
		"TEST_ARG": "integration-test-value",
	}

	// Test buildWithKaniko directly using our real kaniko setup
	err = b.buildWithKaniko(tmpDir, dockerfilePath, fullImageTag, env)
	require.NoError(t, err, "buildWithKaniko direct call failed")
}
