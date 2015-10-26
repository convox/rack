.PHONY: all test

all: test

publish:
	docker tag -f convox/api:$(VERSION) convox/api:latest
	docker push convox/api:latest
	for region in us-east-1 us-west-2 eu-west-1 ap-northeast-1; do \
		aws s3 cp s3://convox/-$$region/release/$(VERSION)/formation.zip s3://convox-$$region/release/latest/formation.zip --acl public-read; \
	done

release:
	docker build -t convox/api:$(VERSION) .
	docker push convox/api:$(VERSION)
	rm -rf /tmp/release.$$PPID
	mkdir -p /tmp/release.$$PPID
	cd /tmp/release.$$PPID
	jq '.Parameters.Version.Default |= "$(VERSION)"' api/dist/kernel.json > kernel.json
	aws s3 cp kernel.json s3://convox/release/$(VERSION)/formation.json --acl public-read
	docker run -i convox/api:$(VERSION) cat api/cmd/formation/lambda.js > lambda.js
	docker run -i convox/api:$(VERSION) cat /go/bin/formation > formation
	zip formation.zip lambda.js formation
	for region in us-east-1 us-west-2 eu-west-1 ap-northeast-1; do \
		aws s3 cp formation.zip s3://convox-$$region/release/$(VERSION)/formation.zip --acl public-read; \
	done

test-deps:
	go get -t -u ./...

test:
	docker info >/dev/null
	go get -t ./...
	go test -v -cover ./...

vendor:
	godep save -r ./...
