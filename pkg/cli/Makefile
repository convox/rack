.PHONY: all build clean release

all: build

build:
	go install ./...

clean:
	rm -f pkg/convox-*

release: clean
	go get -u github.com/karalabe/xgo
	tar cz $(GOPATH) | docker run -v $(GOPATH):/gopath -i ubuntu tar xz -C /gopath --strip-components 2
	$(GOPATH)/bin/xgo -branch $(shell git rev-parse HEAD) -out pkg/convox -targets 'darwin/amd64,linux/amd64,windows/amd64' -ldflags "-X main.version=$(VERSION)" .
	mkdir -p pkg && docker run -v $(GOPATH):/gopath -i ubuntu tar czv /gopath/src/github.com/convox/rack/cmd/convox/pkg | tar xzv -C pkg --strip-components 8
	aws s3 cp pkg/convox-darwin-10.6-amd64 s3://convox/release/$(VERSION)/cli/darwin/convox --acl public-read
	aws s3 cp pkg/convox-linux-amd64 s3://convox/release/$(VERSION)/cli/linux/convox --acl public-read
	aws s3 cp pkg/convox-windows-4.0-amd64.exe s3://convox/release/$(VERSION)/cli/windows/convox.exe --acl public-read
