.PHONY: all build clean package package-deps package-export package-gopath release

pkg_darwin  = pkg/convox-darwin-amd64
pkg_darwin_arm64  = pkg/convox-darwin-arm64
pkg_linux   = pkg/convox-linux-amd64
pkg_linux_arm64   = pkg/convox-linux-amd64
pkg_windows = pkg/convox-windows-4.0-amd64.exe

sources = $(shell find . -name '*.go')

all: build

build:
	go install ./...

clean:
	rm -f pkg/convox-*

package: package-gopath $(pkg_darwin) $(pkg_linux) $(pkg_windows) package-export

# copy finished packages out if running inside a container
package-export:
	if [ -f /.dockerenv ]; then mkdir -p pkg && docker run -v $(GOPATH):/gopath -i ubuntu tar czv /gopath/src/github.com/convox/rack/cmd/convox/pkg | tar xzv -C pkg --strip-components 8; fi

# copy gopath in if running inside a container
package-gopath:
	mkdir -p pkg
	if [ -f /.dockerenv ]; then tar cz $(GOPATH) | docker run -v $(GOPATH):/gopath -i ubuntu tar xz -C /gopath --strip-components 2; fi

release: package
	aws s3 cp $(pkg_darwin) s3://convox/release/$(VERSION)/cli/darwin/convox --acl public-read
	aws s3 cp $(pkg_darwin_arm64) s3://convox/release/$(VERSION)/cli/darwin/convox-arm64 --acl public-read
	aws s3 cp $(pkg_linux) s3://convox/release/$(VERSION)/cli/linux/convox --acl public-read
	aws s3 cp $(pkg_linux_arm64) s3://convox/release/$(VERSION)/cli/linux/convox-arm64 --acl public-read
	aws s3 cp $(pkg_windows) s3://convox/release/$(VERSION)/cli/windows/convox.exe --acl public-read

$(pkg_darwin): $(sources)
	# cd to root folder because we need go.mod
	cd ../.. && env GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o pkg/convox-darwin-amd64
	cd ../.. && env GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o pkg/convox-darwin-arm64

$(pkg_linux): $(sources)
	# cd to root folder because we need go.mod
	cd ../.. && env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o pkg/convox-linux-amd64
	cd ../.. && env GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o pkg/convox-linux-arm64

$(pkg_windows): $(sources)
	# cd to root folder because we need go.mod
	cd ../../ && env GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o pkg/convox-windows-4.0-amd64.exe
