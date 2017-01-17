FIXTURES := $(shell find . -name '*.yml')
MANIFESTS := $(FIXTURES:%.yml=%.json)

.PHONY: all fixture

all: clean fixture $(MANIFESTS)

clean:
	@rm -f $(MANIFESTS) ./fixture

fixture:
	go build -o ./fixture fixture.go

%.json: %.yml
	env AWS_REGION=test PROVIDER=test CLUSTER=convox-test RACK=convox-test ./fixture $< > $@
