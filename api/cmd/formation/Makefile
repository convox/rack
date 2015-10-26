.PHONY: all build clean formation

all: build

build: formation.zip

clean:
	rm -f formation formation.zip

formation: *.go
	GOOS=linux GOARCH=amd64 go build

formation.zip: lambda.js formation
	zip formation.zip lambda.js formation
