.PHONY: all build clean release

default: lambda.zip

clean:
	rm lambda.zip main

lambda.zip: index.js main
	zip -r lambda.zip main index.js

main: *.go
	GOOS=linux GOARCH=amd64 go build -o main

release: lambda.zip
	for region in us-east-1 us-west-2 eu-west-1 ap-northeast-1 ap-southeast-2; do \
		aws s3 cp lambda.zip s3://convox-$$region/release/$(VERSION)/fluentd.zip --acl public-read; \
	done
