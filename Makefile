PORT ?= 3000

.PHONY: default dev

all: build

build:
	go get ./...

dev:
	@forego run fig up

vendor:
	godep save -r -copy=true ./...

docker-clean:
	docker rm -f `docker ps -a -q` ; docker rmi -f `docker images -q`