PORT ?= 3000

.PHONY: default dev

all: build

dev:
	@export $(shell cat .env); docker-compose up

ami:
	docker run -v $(shell pwd):/build --env-file .env convox/builder /build convox

build:
	docker build -t convox/kernel .

squash: build
	cat dist/Dockerfile | docker build -t convox/kernel:squash -
	docker save convox/kernel:squash | docker run -i convox/squash -verbose -t convox/kernel:squash | docker load

# TODO: make version dynamic
release: squash
	docker tag -f convox/kernel:squash convox/kernel:latest
	docker tag -f convox/kernel:squash convox/kernel:v1
	docker push convox/kernel:v1
	docker push convox/kernel:latest

vendor:
	godep save -r -copy=true ./...
