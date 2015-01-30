PORT ?= 3000

.PHONY: default dev

all: build

build:
	go get ./...

vendor:
	godep save -r -copy=true ./...
