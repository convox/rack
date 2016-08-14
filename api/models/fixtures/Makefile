FIXTURES := $(shell find . -name '*.yml')
MANIFESTS := $(FIXTURES:%.yml=%.json)

.PHONY: all

all: fixture $(MANIFESTS)

fixture: fixture.go
	go build -o fixture fixture.go

%.json: %.yml
	env AWS_REGION=test PROVIDER=test RACK=convox-test fixture $< > $@
