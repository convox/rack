all: build

build: 
	go get cmd/convox

vendor:
	godep save -r ./...
