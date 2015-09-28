.PHONY: all test

all: test

test-deps:
	go get -t -u ./...

test:
	docker info >/dev/null
	go get -t ./...
	go test -v -cover ./...
