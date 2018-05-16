.PHONY: all build builder mocks release templates test

all: build

build:
	go install .

builder:
	docker build -t convox/build:$(USER) -f cmd/build/Dockerfile .
	docker push convox/build:$(USER)

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
	env PROVIDER=test bin/test
