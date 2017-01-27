.PHONY: all release templates vendor

all: templates

release:
	equinox release --config=.equinox.yaml --version=$(shell convox/convox --version | cut -d' ' -f3) .

templates:
	go-bindata -pkg=templates -modtime=1475420665 -prefix=templates -o=appify/templates/templates.go -ignore=templates.go appify/templates/...
