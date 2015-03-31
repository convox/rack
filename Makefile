all: build

build:
	docker build -t convox/builder .

release: build
	docker push convox/builder
	# TODO: version numbering
	# docker tag -f convox/builder:latest convox/builder:v1
	# docker push convox/builder:v1

test: build
	docker run --env-file .env convox/builder https://github.com/convox-examples/sinatra sinatra
