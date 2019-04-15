.PHONY: all build clean release

VERSION=latest

default: lambda.zip

clean:
	rm lambda.zip main

lambda.zip: index.js main
	zip -r lambda.zip main index.js

main: *.go
	GOOS=linux go build -o main

release: lambda.zip
	aws s3 cp lambda.zip s3://convox/release/$(VERSION)/lambda/syslog.zip --acl public-read
	for region in $(shell cat ../../../../REGIONS); do \
		aws s3 cp s3://convox/release/$(VERSION)/lambda/syslog.zip s3://convox-$$region/release/$(VERSION)/lambda/syslog.zip --acl public-read --region $$region; \
	done
