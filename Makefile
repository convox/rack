all: build

build:
	docker build -t convox/builder .

squash: build
	cat dist/Dockerfile | docker build -t convox/builder:squash -
	docker save convox/builder:squash | docker run -i convox/squash -verbose -t convox/builder:squash | docker load

release: squash
	docker tag -f convox/builder:squash convox/builder:latest
	docker tag -f convox/builder:squash convox/builder:v1
	docker push convox/builder:latest
	docker push convox/builder:v1

test: build
	docker run --env-file .env convox/builder https://github.com/convox-examples/sinatra sinatra
