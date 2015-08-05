.PHONY: all build clean release formation

VERSION=latest

all: build

build: formation lambda.zip

clean:
	rm -f formation lambda.zip

release: build
	# lambda regions: https://docs.aws.amazon.com/general/latest/gr/rande.html#lambda_region
	for region in us-east-1 us-west-2 eu-west-1 ap-northeast-1; do \
		aws s3 cp lambda.zip s3://convox-$$region/release/$(VERSION)/formation.zip --acl public-read; \
		aws s3 cp lambda.zip s3://convox-$$region/release/latest/formation.zip  --acl public-read; \
	done
	aws s3 cp lambda.zip s3://convox/release/$(VERSION)/formation.zip --acl public-read
	aws s3 cp lambda.zip s3://convox/release/latest/formation.zip  --acl public-read

formation: *.go
	GOOS=linux GOARCH=amd64 go build

lambda.zip: lambda.js formation
	zip lambda.zip lambda.js formation
