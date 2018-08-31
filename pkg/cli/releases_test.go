package cli_test

import (
	"time"

	"github.com/convox/rack/pkg/structs"
)

var fxRelease = structs.Release{
	Id:       "release1",
	App:      "app1",
	Build:    "build1",
	Env:      "FOO=bar\nBAZ=quux",
	Manifest: "manifest",
	Created:  time.Now().UTC().Add(-49 * time.Hour),
}
