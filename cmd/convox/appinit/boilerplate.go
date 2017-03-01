package appinit

import "time"

// Boilerplate contains data representing a generic app
type Boilerplate struct{}

// GenerateDockerfile generates a Dockerfile
func (bp *Boilerplate) GenerateDockerfile() ([]byte, error) {
	df := `FROM ubuntu:16.04

COPY . /app`
	return []byte(df), nil
}

// GenerateDockerIgnore generates a .dockerignore file
func (bp *Boilerplate) GenerateDockerIgnore() ([]byte, error) {
	di := `.env
.git`
	return []byte(di), nil
}

// GenerateLocalEnv generates a .env file
func (bp *Boilerplate) GenerateLocalEnv() ([]byte, error) {
	return nil, nil
}

// GenerateGitIgnore generates a .gitignore file
func (bp *Boilerplate) GenerateGitIgnore() ([]byte, error) {
	return writeAsset("appinit/templates/gitignore", nil)
}

// GenerateManifest generates a docker-compose.yml file
func (bp *Boilerplate) GenerateManifest() ([]byte, error) {
	dc := `version: "2"
services:
  main:
    build: .`

	return []byte(dc), nil
}

// Setup runs the buildpacks and collects metadata
// Must be called before other Generate* methods
func (bp *Boilerplate) Setup(dir string) error {
	time.Sleep(100 * time.Millisecond) //harmless...
	return nil
}
