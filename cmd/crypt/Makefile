all: build

build:
	docker build -t convox/crypt .

test: build
	cat eg/env | docker run --env-file .env -i convox/crypt encrypt $(KEY) | docker run --env-file .env -i convox/crypt decrypt $(KEY)
