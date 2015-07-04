PORT ?= 3000

.PHONY: all build dev release vendor

all: build

build:
	docker build -t convox/kernel .

dev:
	@export $(shell cat .env); docker-compose up

release:
	convox run --app release release kernel

vendor:
	godep save -r -copy=true ./...
