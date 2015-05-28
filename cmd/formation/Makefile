# always rebuild
.PHONY: formation

all: build

build: formation lambda.zip

upload: build
	@export $(shell cat .env); bin/upload

formation: *.go
	GOOS=linux GOARCH=amd64 go build

lambda.zip: lambda.js formation
	zip lambda.zip lambda.js formation
