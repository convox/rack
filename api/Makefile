.PHONY: all build dev release vendor

VERSION=edge

all: build

build:
	docker build -t convox/kernel .

dev:
	@export $(shell cat .env); docker-compose up

release:
	cd cmd/formation && make release VERSION=$(VERSION)
	jq '.Parameters.Version.Default |= "$(VERSION)"' dist/kernel.json > /tmp/kernel.json
	aws s3 cp /tmp/kernel.json s3://convox/release/$(VERSION)/formation.json --acl public-read

test:
	go test -v ./...

vendor:
	env GO15VENDOREXPERIMENT=1 godep save ./...
