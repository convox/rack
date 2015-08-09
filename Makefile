.PHONY: all build dev release vendor

VERSION=latest

all: build

build:
	docker build -t convox/kernel .

dev:
	@export $(shell cat .env); docker-compose up

release:
	cd cmd/formation && make release VERSION=$(VERSION)
	jq '.Parameters.Version.Default |= "$(VERSION)"' dist/kernel.json > /tmp/kernel.json
	aws s3 cp /tmp/kernel.json s3://convox/release/$(VERSION)/formation.json --acl public-read
ifeq ($(LATEST),yes)
	aws s3 cp /tmp/kernel.json s3://convox/release/latest/formation.json --acl public-read
	echo $(VERSION) > /tmp/version && aws s3 cp /tmp/version s3://convox/release/latest/version --acl public-read
endif

vendor:
	godep save -r -copy=true ./...
