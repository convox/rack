.PHONY: all build dev release vendor

VERSION=edge

all: build

build:
	docker build -t convox/api .

templates:
	go-bindata -pkg=models -prefix=models -o=models/templates.go models/templates/...

test:
	go get -t ./...
	env PROVIDER=test go test -v -cover ./...

test-coverage:
	gocov test -v ./... | gocov-html > /tmp/coverage.html
	open /tmp/coverage.html

test-deps:
	go get github.com/axw/gocov/gocov
	go get gopkg.in/matm/v1/gocov-html
