# always rebuild
.PHONY: formation

VERSION=latest

all: build

build: formation lambda.zip

clean:
	rm -f formation lambda.zip

release: build
	aws s3 cp lambda.zip s3://convox/lambda/formation.$(VERSION).zip --acl public-read

formation: *.go
	GOOS=linux GOARCH=amd64 go build

lambda.zip: lambda.js formation
	zip lambda.zip lambda.js formation
