.PHONY: all build test vendor

all: build

build:
	docker build -t convox/build .

test:
	go test -v -run TestDockerRunning && go test -v ./...

vendor:
	godep save -r -copy=true ./...
