.PHONY: all build install release test vendor

all: build

build:
	go build -o convox/convox ./convox

install:
	go get ./convox

release: build
	equinox release --config=.equinox.yaml --version=$(shell convox/convox --version | cut -d' ' -f3) ./convox

test:
	go test -v ./...

vendor:
	godep save -r ./...
