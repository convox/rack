all: build

build:
	docker build -t convox/builder .

test: build
	docker run --env-file .env convox/builder https://github.com/convox-examples/sinatra sinatra-example
