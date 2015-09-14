all: test

test:
	go test -v -cover ./...

vendor:
	godep save -r -copy=true ./...
