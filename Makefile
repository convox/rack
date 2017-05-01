.PHONY: all templates test test-deps vendor

REGIONS = us-east-1 us-east-2 us-west-1 us-west-2 eu-west-1 eu-central-1 ap-northeast-1 ap-southeast-1 ap-southeast-2

all: templates

builder:
	docker build -t convox/build:$(USER) -f api/cmd/build/Dockerfile .
	docker push convox/build:$(USER)

fixtures:
	make -C api/models/fixtures

release:
	make -C provider release VERSION=$(VERSION)
	docker build -t convox/api:$(VERSION) .
	docker push convox/api:$(VERSION)

templates:
	go get -u github.com/jteeuwen/go-bindata/...
	make -C api templates
	make -C cmd templates
	make -C provider templates
	make -C sync templates

test:
	env PROVIDER=test CONVOX_WAIT= bin/test

listamis:
	@$(foreach region,$(REGIONS),aws ec2 describe-images --filters "Name=name,Values=${name}" --region="$(region)" | jq --raw-output '"${region}: \(.Images[0].ImageId)"' ;)

vendor:
	godep save ./...
