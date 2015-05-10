all: build

build: 
	go get .

test: build
	export $(shell cat .env)
	cat eg/env | $(GOPATH)/bin/env encrypt | $(GOPATH)/bin/env decrypt

vendor:
	godep save -r ./...
