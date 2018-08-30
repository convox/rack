package cli_test

import (
	"time"

	"github.com/convox/rack/pkg/structs"
)

var fxRelease = structs.Release{
	Id:       "release1",
	App:      "app1",
	Build:    "build1",
	Env:      "env",
	Manifest: "manifest",
	Created:  time.Now().UTC(),
}
