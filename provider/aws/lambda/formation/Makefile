.PHONY: all build clean main release

all: lambda.zip

clean:
	rm -f lambda.zip main

lambda.zip: index.js main
	zip -r lambda.zip main index.js

main: *.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-extldflags=-static" -o main

release: lambda.zip
	aws s3 cp lambda.zip s3://convox/release/$(VERSION)/lambda/formation.zip --acl public-read
	for region in $(shell aws ec2 describe-regions --query "Regions[].RegionName" --output text); do \
		aws s3 cp s3://convox/release/$(VERSION)/lambda/formation.zip s3://convox-$$region/release/$(VERSION)/lambda/formation.zip --acl public-read --region $$region; \
	done
