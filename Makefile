.PHONY: all build data install release test vendor

all: build

build:
	go build -o convox/convox ./convox

coverage:
	gocov test -v ./... | gocov-html > /tmp/coverage.html
	open /tmp/coverage.html

data:
	go-bindata -o convox/asset.go -prefix convox convox/data
	go-bindata -o manifest/asset.go -prefix manifest -pkg manifest manifest/data

deps:
	go get github.com/axw/gocov/gocov
	go get gopkg.in/matm/v1/gocov-html

install:
	go get ./convox

release: build
	equinox release --config=.equinox.yaml --version=$(shell convox/convox --version | cut -d' ' -f3) ./convox

test:
	go test -v -cover ./...

vendor:
	godep save -r ./...
