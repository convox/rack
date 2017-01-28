package appify_test

import (
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/convox/rack/cmd/convox/appify"
	"github.com/convox/rack/manifest"
	"github.com/stretchr/testify/assert"
)

type DirKind struct {
	Dir  string
	Kind string
}

func availableHerokuApps() ([]DirKind, error) {
	files, err := ioutil.ReadDir("fixtures/buildpacks")
	if err != nil {
		return []DirKind{}, err
	}

	dirs := []DirKind{}
	for _, file := range files {
		dirs = append(dirs, DirKind{
			Dir:  fmt.Sprintf("fixtures/buildpacks/%s", file.Name()),
			Kind: file.Name(),
		})
	}

	return dirs, nil
}

func TestBuildpackManifest(t *testing.T) {

	dirs, err := availableHerokuApps()
	assert.NoError(t, err)

	for _, d := range dirs {

		bp := appify.Buildpack{}
		err := bp.Setup(d.Dir)
		assert.NoError(t, err)

		m, err := manifest.LoadFile(path.Join(d.Dir, "docker-compose.yml"))
		assert.NoError(t, err)

		assert.Equal(t, *m, bp.Manifest)
	}
}
