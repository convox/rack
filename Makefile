all: build

build:
	docker build -t convox/service .
