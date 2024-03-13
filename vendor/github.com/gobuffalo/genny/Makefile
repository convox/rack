TAGS ?= "sqlite"
GO_BIN ?= go

install:
	packr2
	$(GO_BIN) install -v ./genny

tidy:
ifeq ($(GO111MODULE),on)
	$(GO_BIN) mod tidy
else
	echo skipping go mod tidy
endif

deps:
	$(GO_BIN) get github.com/gobuffalo/release
	$(GO_BIN) get github.com/gobuffalo/packr/v2/packr2
	$(GO_BIN) get -tags ${TAGS} -t ./...
	make tidy

build:
	packr2
	$(GO_BIN) build -v .
	make tidy

test:
	packr2
	$(GO_BIN) test -tags ${TAGS} ./...
	make tidy

ci-test:
	$(GO_BIN) test -tags ${TAGS} -race ./...
	make tidy

lint:
	gometalinter --vendor ./... --deadline=1m --skip=internal

update:
	packr2 clean
	$(GO_BIN) get -u -tags ${TAGS}
	make tidy
	packr2
	make test
	make install
	make tidy

release-test:
	$(GO_BIN) test -tags ${TAGS} -race ./...

release:
	make tidy
	release -y -f version.go
	make tidy
