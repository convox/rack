.PHONY: all build release

all: build

build:
	go install ./...

release:
	make -C aws release VERSION=$(VERSION)
	make -C kaws release VERSION=$(VERSION)
	make -C local release VERSION=$(VERSION)
