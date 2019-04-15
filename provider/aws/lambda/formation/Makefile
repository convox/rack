.PHONY: all build clean main release

all: lambda.zip

clean:
	rm -f lambda.zip main

lambda.zip: index.js main
	zip -r lambda.zip main index.js

main: *.go
	GOOS=linux GOARCH=amd64 go build -o main

release: lambda.zip
	aws s3 cp lambda.zip s3://convox/release/$(VERSION)/lambda/formation.zip --acl public-read
	for region in $(shell cat ../../../../REGIONS); do \
		aws s3 cp s3://convox/release/$(VERSION)/lambda/formation.zip s3://convox-$$region/release/$(VERSION)/lambda/formation.zip --acl public-read --region $$region; \
	done
