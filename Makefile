all: build

build:
	docker build -t convox/agent .

test: build
	docker run --env-file .env convox/agent testapp testps i-0000000
