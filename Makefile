.PHONY: all builder fixtures mocks release templates test vendor

all: templates

builder:
	docker build -t convox/build:$(USER) -f cmd/build/Dockerfile .
	docker push convox/build:$(USER)

fixtures:
	make -C api/models/fixtures

mocks:
	go get -u github.com/vektra/mockery/.../
	make -C structs mocks

release:
	make -C cmd/convox release VERSION=$(VERSION)
	make -C provider release VERSION=$(VERSION)
	docker build -t convox/rack:$(VERSION) .
	docker push convox/rack:$(VERSION)

templates:
	go get -u github.com/jteeuwen/go-bindata/...
	make -C cmd templates
	make -C provider templates
	make -C sync templates

test:
	env PROVIDER=test CONVOX_WAIT= bin/test

vendor:
	godep save ./...
