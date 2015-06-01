# always rebuild
.PHONY: formation

all: build

build: formation lambda.zip

release: build
	aws s3 cp lambda.zip s3://convox/lambda-formation.zip --acl public-read

formation: *.go
	GOOS=linux GOARCH=amd64 go build

lambda.zip: lambda.js formation
	zip lambda.zip lambda.js formation
