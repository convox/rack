all: build

build: 
	go get ./convox

vendor:
	godep save -r ./...

test:
	go test -v ./...
