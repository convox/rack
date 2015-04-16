PORT ?= 3000

.PHONY: default dev

all: build

dev:
	@export $(shell cat .env); docker-compose up

ami:
	docker pull convox/build
	docker run -v $(shell pwd):/build --env-file .env convox/build convox /build

build:
	docker build -t convox/kernel .

release: squash
	docker push convox/kernel
	# TODO: version numbering
	docker tag -f convox/kernel convox/kernel:v1
	docker push convox/kernel:v1

vendor:
	godep save -r -copy=true ./...
