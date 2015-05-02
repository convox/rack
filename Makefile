all: build

build:
	docker build -t convox/build .

test: build
	docker run --env-file .env convox/build sinatra https://github.com/convox-examples/sinatra

vendor:
	godep save -r -copy=true ./...
