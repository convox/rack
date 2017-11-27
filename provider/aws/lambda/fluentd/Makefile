.PHONY: all build clean release

all: lambda.zip

clean:
	rm -f lambda.zip main

lambda.zip: index.js main
	zip -r lambda.zip main index.js

main: *.go
	GOOS=linux GOARCH=amd64 go build -o main

release: lambda.zip
	for region in $(shell cat ../../../../REGIONS); do \
		aws s3 cp lambda.zip s3://goodeggs-convox-$$region/release/$(VERSION)/lambda/fluentd.zip --acl public-read --region $$region; \
	done
