package build

import "testing"

// SetConvoxEnvPath points injectConvoxEnv at path for the rest of the test.
func SetConvoxEnvPath(t *testing.T, path string) {
	old := convoxEnvPath
	convoxEnvPath = path
	t.Cleanup(func() { convoxEnvPath = old })
}
