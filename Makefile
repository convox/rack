.PHONY: all test

all: test

publish:
	docker tag -f convox/api:$(VERSION) convox/api:latest
	docker push convox/api:latest

release:
	docker build -t convox/api:$(VERSION) .
	docker push convox/api:$(VERSION)
	cd api && make release

test-deps:
	go get -t -u ./...

test:
	docker info >/dev/null
	go get -t ./...
	go test -v -cover ./...

vendor:
	godep save -r ./...
