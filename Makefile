.PHONY: all templates test test-deps vendor

all: test

release:
	cd api/cmd/formation && make release VERSION=$(VERSION)
	docker build -t convox/api:$(VERSION) .
	docker push convox/api:$(VERSION)
	mkdir -p /tmp/release/$(VERSION)
	cd /tmp/release/$(VERSION)
	jq '.Parameters.Version.Default |= "$(VERSION)"' api/dist/kernel.json > kernel.json
	aws s3 cp kernel.json s3://convox/release/$(VERSION)/formation.json --acl public-read

templates:
	make -C api templates
	make -C cmd/convox templates

test-deps:
	go get -t -u ./...

test:
	docker info >/dev/null
	go get -t ./...
	env PROVIDER=test go test -v -cover ./...

vendor:
	godep restore
	godep save -r ./...
