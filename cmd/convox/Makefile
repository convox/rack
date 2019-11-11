.PHONY: all build clean package package-deps package-gopath release

pkg_darwin  = pkg/convox-darwin-10.6-amd64
pkg_linux   = pkg/convox-linux-amd64
pkg_windows = pkg/convox-windows-4.0-amd64.exe

sources = $(shell find . -name '*.go')

all: build

build:
	go install ./...

clean:
	rm -f pkg/convox-*

package: package-deps package-gopath $(pkg_darwin) $(pkg_linux) $(pkg_windows)

package-deps:
	go get -u github.com/karalabe/xgo
	mkdir -p pkg

# set up gopath in docker volume if running inside a container
package-gopath:
	if [ -f /.dockerenv ]; then tar cz $(GOPATH) | docker run -v $(GOPATH):/gopath -i ubuntu tar xz -C /gopath --strip-components 2; fi

release: package
	aws s3 cp $(pkg_darwin) s3://convox/release/$(VERSION)/cli/darwin/convox --acl public-read
	aws s3 cp $(pkg_linux) s3://convox/release/$(VERSION)/cli/linux/convox --acl public-read
	aws s3 cp $(pkg_windows) s3://convox/release/$(VERSION)/cli/windows/convox.exe --acl public-read

$(pkg_darwin): $(sources)
	$(GOPATH)/bin/xgo -branch $(shell git rev-parse HEAD) -out pkg/convox -targets 'darwin/amd64' -ldflags "-X main.version=$(VERSION)" .

$(pkg_linux): $(sources)
	$(GOPATH)/bin/xgo -branch $(shell git rev-parse HEAD) -out pkg/convox -targets 'linux/amd64' -ldflags "-X main.version=$(VERSION)" .

$(pkg_windows): $(sources)
	$(GOPATH)/bin/xgo -branch $(shell git rev-parse HEAD) -out pkg/convox -targets 'windows/amd64' -ldflags "-X main.version=$(VERSION)" .