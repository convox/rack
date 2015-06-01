# always rebuild
.PHONY: bootstrap

all: build

build: bootstrap lambda.zip

create: build
	bin/create
	
release: build
	bin/update

delete:
	bin/delete

bootstrap: *.go
	GOOS=linux GOARCH=amd64 go build

lambda.zip: lambda.js bootstrap
	zip lambda.zip lambda.js bootstrap
