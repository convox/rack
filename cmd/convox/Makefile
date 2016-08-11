.PHONY: all build data install release templates test vendor

all: build

data:
	go-bindata -o convox/asset.go -prefix convox convox/data
	go-bindata -o manifest/asset.go -prefix manifest -pkg manifest manifest/data

install:
	go get .

release:
	equinox release --config=.equinox.yaml --version=$(shell convox/convox --version | cut -d' ' -f3) .

templates:
	go-bindata -pkg=templates -prefix=templates/ -o=templates/templates.go templates/...

test:
	go get -t ./...
	go test -v -cover ./...

test-coverage:
	gocov test -v ./... | gocov-html > /tmp/coverage.html
	open /tmp/coverage.html

test-deps:
	go get github.com/axw/gocov/gocov
	go get gopkg.in/matm/v1/gocov-html

vendor-changes:
	cd ../changes && env GOOS=linux GOARCH=amd64 go build .
	go-bindata -o changes/assets.go -pkg="changes" ../changes/changes
