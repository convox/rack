.PHONY: all templates test test-deps vendor

all: templates

builder:
	docker build -t goodeggs/convox-build:$(USER) -f api/cmd/build/Dockerfile .
	docker push goodeggs/convox-build:$(USER)

fixtures:
	make -C api/models/fixtures

release:
	make -C provider release VERSION=$(VERSION)
	docker build -t goodeggs/convox-api:$(VERSION) .
	docker push goodeggs/convox-api:$(VERSION)

templates:
	go get -u github.com/jteeuwen/go-bindata/...
	make -C api templates
	make -C cmd templates
	make -C provider templates
	make -C sync templates

test:
	env PROVIDER=test CONVOX_WAIT= bin/test

vendor:
	godep save ./...
