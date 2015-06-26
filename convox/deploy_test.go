package main

import (
	"testing"

	"github.com/convox/cli/convox/build"
)

func TestBuildTagPush(t *testing.T) {
	m, _ := build.ManifestFromBytes([]byte(`web:
  build: .
  command: ruby web.rb
  ports:
  - 5000:3000
worker:
  build: .
  command: ruby worker.rb
redis:
  image: convox/redis
`))

	expect(t,
		m.ImageNames("myproj"),
		[]string{"convox/redis", "myproj_web", "myproj_worker"},
	)

	expect(t,
		m.TagNames("private.registry.com:5000", "myproj", "123"),
		[]string{"private.registry.com:5000/convox/redis:123", "private.registry.com:5000/myproj_web:123", "private.registry.com:5000/myproj_worker:123"},
	)
}
