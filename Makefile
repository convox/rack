all: build

build:
	docker build -t convox/env .

test: build
	cat eg/env | docker run --env-file .env -i convox/env encrypt $(KEY) | docker run --env-file .env -i convox/env decrypt $(KEY)

vendor:
	godep save -r ./...
