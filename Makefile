.PHONY: all api app build convox crypt service test vendor

all: api app build convox crypt service

api:
	go get ./api

app:
	go get ./cmd/app

build:
	go get ./cmd/build

convox:
	go get ./cmd/convox

crypt:
	go get ./cmd/crypt

service:
	go get ./cmd/service

test:
	go test -v -cover ./...

vendor:
	godep save -r -copy=true ./...
