all: build

build:
	docker build -t convox/builder .

release: build
	cat dist/Dockerfile | docker build -t convox/builder:release -
	docker save convox/builder:release | docker run -i proximo/squash -verbose -t convox/builder:release | docker load
	docker push convox/builder:release

test: build
	docker run --env-file .env convox/builder https://github.com/convox-examples/sinatra sinatra
