PORT ?= 3000

.PHONY: default dev

all: build

build:
	go get ./...

dev:
	@forego run fig up

vendor:
	godep save -r -copy=true ./...
