.PHONY: all build dev release vendor

VERSION=latest

all: build

build:
	docker build -t convox/kernel .

dev:
	@export $(shell cat .env); docker-compose up

release:
	cd cmd/formation && make release VERSION=$(VERSION)
	aws s3 cp dist/kernel.json s3://convox/release/$(VERSION)/formation.json --acl public-read
	aws s3 cp dist/kernel.json s3://convox/release/latest/formation.json --acl public-read

vendor:
	godep save -r -copy=true ./...
