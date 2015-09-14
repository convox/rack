.PHONY: all build dev release vendor

VERSION=edge

all: build

build:
	docker build -t convox/api .

release:
	cd cmd/formation && make release VERSION=$(VERSION)
	jq '.Parameters.Version.Default |= "$(VERSION)"' dist/kernel.json > /tmp/kernel.json
	aws s3 cp /tmp/kernel.json s3://convox/release/$(VERSION)/formation.json --acl public-read

test:
	go get -t ./...
	go test -v -cover ./...

test-coverage:
	gocov test -v ./... | gocov-html > /tmp/coverage.html
	open /tmp/coverage.html

test-deps:
	go get github.com/axw/gocov/gocov
	go get gopkg.in/matm/v1/gocov-html
