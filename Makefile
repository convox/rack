all: build

build:
	docker build -t convox/service .

release: build
	docker push convox/service
