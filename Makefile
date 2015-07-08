.PHONY: all build data install release test vendor

all: build

build:
	go build -o convox/convox ./convox

data:
	go-bindata -o convox/asset.go -prefix convox convox/data
	go-bindata -o manifest/asset.go -prefix manifest -pkg manifest manifest/data

install:
	go get ./convox

release: build
	equinox release --config=.equinox.yaml --version=$(shell convox/convox --version | cut -d' ' -f3) ./convox

test:
	go test -v ./...

vendor:
	godep save -r ./...
