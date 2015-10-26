.PHONY: all test

all: test

publish:
	docker tag -f convox/api:$(VERSION) convox/api:latest
	docker push convox/api:latest
	cd api/cmd/formation && make publish VERSION=$(VERSION)

release:
	cd api/cmd/formation && make release VERSION=$(VERSION)
	docker build -t convox/api:$(VERSION) .
	docker push convox/api:$(VERSION)
	mkdir -p /tmp/release/$(VERSION)
	cd /tmp/release/$(VERSION)
	jq '.Parameters.Version.Default |= "$(VERSION)"' api/dist/kernel.json > kernel.json
	aws s3 cp kernel.json s3://convox/release/$(VERSION)/formation.json --acl public-read

test-deps:
	go get -t -u ./...

test:
	docker info >/dev/null
	go get -t ./...
	go test -v -cover ./...

vendor:
	godep save -r ./...
