all: build

build: formation lambda.zip

upload: build
	@bin/upload

formation: *.go
	GOOS=linux GOARCH=amd64 go build

lambda.zip: lambda.js formation
	zip lambda.zip lambda.js formation
