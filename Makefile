.PHONY: all api app build convox crypt service test vendor

all: test

test:
	go get -t ./...
	go test -v -cover ./...

vendor:
	godep save -r -copy=true ./...
