.PHONY: all build data test vendor

all: build

build:
	docker build -t convox/build .

data:
	go-bindata data/

test: build
	go test -v ./...

vendor:
	godep save -r -copy=true ./...
