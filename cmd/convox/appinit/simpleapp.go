package appinit

type SimpleApp struct {
	Kind string
}

func (sa *SimpleApp) GenerateEntrypoint() ([]byte, error) {
	return writeAsset("appinit/templates/entrypoint.sh", nil)
}
func (sa *SimpleApp) GenerateDockerfile() ([]byte, error) {
	input := map[string]interface{}{
		"kind": sa.Kind,
	}
	return writeAsset("appinit/templates/Dockerfile", input)
}
func (sa *SimpleApp) GenerateDockerIgnore() ([]byte, error) {
	return writeAsset("appinit/templates/dockerignore", nil)
}
func (sa *SimpleApp) GenerateManifest(dir string) ([]byte, error) {
	return generateManifestData(dir)
}
