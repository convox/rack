all: build

build:
	docker build -t convox/build .

test: build
	docker run --env-file .env convox/build https://github.com/convox-examples/sinatra sinatra
