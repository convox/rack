LAST_TAG := $(shell git describe --abbrev=0 --tags)

USER := convox
REPO := cli
EXECUTABLE := convox

# only include the amd64 binaries, otherwise the github release will become
# too big
UNIX_EXECUTABLES := \
	darwin/amd64/$(EXECUTABLE) \
	freebsd/amd64/$(EXECUTABLE) \
	linux/amd64/$(EXECUTABLE)
WIN_EXECUTABLES := \
	windows/amd64/$(EXECUTABLE).exe

COMPRESSED_EXECUTABLES=$(UNIX_EXECUTABLES:%=%.tar.bz2) $(WIN_EXECUTABLES:%.exe=%.zip)
COMPRESSED_EXECUTABLE_TARGETS=$(COMPRESSED_EXECUTABLES:%=bin/%)

UPLOAD_CMD = github-release upload -u $(USER) -r $(REPO) -t $(LAST_TAG) -n $(subst /,-,$(FILE)) -f bin/$(FILE)

all: $(EXECUTABLE)

# amd64
bin/freebsd/amd64/$(EXECUTABLE):
	cd convox; GOARCH=amd64 GOOS=freebsd go build -ldflags "-X main.version $(LAST_TAG)" -o "$@"
bin/darwin/amd64/$(EXECUTABLE):
	cd convox; GOARCH=amd64 GOOS=darwin  go build -ldflags "-X main.version $(LAST_TAG)" -o "$@"
bin/linux/amd64/$(EXECUTABLE):
	cd convox; GOARCH=amd64 GOOS=linux   go build -ldflags "-X main.version $(LAST_TAG)" -o "$@"
bin/windows/amd64/$(EXECUTABLE).exe:
	cd convox; GOARCH=amd64 GOOS=windows go build -ldflags "-X main.version $(LAST_TAG)" -o "$@"

# compressed artifacts, makes a huge difference (Go executable is ~9MB,
# after compressing ~2MB)
%.tar.bz2: %
	cd convox; tar -jcvf "$<.tar.bz2" "$<"
%.zip: %.exe
	cd convox; zip "$@" "$<"

# git tag -a v$(RELEASE) -m 'release $(RELEASE)'
release: $(COMPRESSED_EXECUTABLE_TARGETS)
	git push && git push --tags
	github-release release -u $(USER) -r $(REPO) \
		-t $(LAST_TAG) -n $(LAST_TAG) || true
	cd convox; $(foreach FILE,$(COMPRESSED_EXECUTABLES),$(UPLOAD_CMD);)

install:
	cd convox; go install

clean:
	cd convox; rm $(EXECUTABLE) || true
	cd convox; rm -rf bin/

.PHONY: clean release dep install

build: 
	go get ./convox

vendor:
	godep save -r ./...

test:
	go test -v ./...

all: test
