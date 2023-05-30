.PHONY: all build package release

pkg_darwin  = pkg/convox-darwin-amd64
pkg_darwin_arm64  = pkg/convox-darwin-arm64
pkg_linux   = pkg/convox-linux-amd64
pkg_linux_arm64   = pkg/convox-linux-arm64
pkg_windows = pkg/convox-windows-amd64.exe

all: build

build:
	go install ./...

package:
	mkdir -p pkg/
	env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -tags=hidraw -ldflags "-X main.version=$(VERSION)" -o $(pkg_linux)
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -mod=vendor -tags=hidraw -ldflags "-X main.version=$(VERSION)" -o $(pkg_linux_arm64)
	env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -mod=vendor -ldflags "-X main.version=$(VERSION)" -o $(pkg_darwin_arm64)
	env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -mod=vendor -ldflags "-X main.version=$(VERSION)" -o $(pkg_windows)

	go install github.com/crazy-max/xgo@v0.7.5
	if [ -f /.dockerenv ]; then tar cz $(GOPATH) | docker run -v $(GOPATH):/gopath -i ubuntu tar xz -C /gopath --strip-components 2; fi
	cd ../.. && $(GOPATH)/bin/xgo -branch $(shell git rev-parse HEAD) -pkg cmd/convox -out cmd/convox/pkg/convox -go 1.17.x -targets 'darwin/amd64' -ldflags "-X main.version=$(VERSION)" .
	if [ -f /.dockerenv ]; then mkdir -p pkg && docker run -v $(GOPATH):/gopath -i ubuntu tar czv /gopath/src/github.com/convox/rack/cmd/convox/pkg | tar xzv -C pkg --strip-components 8; fi

release: package
	aws s3 cp $(pkg_darwin) s3://convox/release/$(VERSION)/cli/darwin/convox --acl public-read
	aws s3 cp $(pkg_darwin_arm64) s3://convox/release/$(VERSION)/cli/darwin/convox-arm64 --acl public-read
	aws s3 cp $(pkg_linux) s3://convox/release/$(VERSION)/cli/linux/convox --acl public-read
	aws s3 cp $(pkg_linux_arm64) s3://convox/release/$(VERSION)/cli/linux/convox-arm64 --acl public-read
	aws s3 cp $(pkg_windows) s3://convox/release/$(VERSION)/cli/windows/convox.exe --acl public-read
