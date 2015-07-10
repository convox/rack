all: build

build:
	docker build -t convox/app .

vendor:
	godep save -r -copy=true ./...

test:
	go test -v ./...