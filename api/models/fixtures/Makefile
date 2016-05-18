FIXTURES := $(shell find . -name '*.yml')
MANIFESTS := $(FIXTURES:%.yml=%.json)

.PHONY: all

all: $(MANIFESTS)

%.json: %.yml
	env AWS_REGION=us-test-2 PROVIDER=test RACK=convox-test fixture $< > $@
