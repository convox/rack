.PHONY: all build clean formation

all: build

build: formation.zip

clean:
	rm -f formation formation.zip

formation: *.go
	GOOS=linux GOARCH=amd64 go build

formation.zip: lambda.js formation
	zip formation.zip lambda.js formation

release: formation.zip
	for region in us-east-1 us-west-2 eu-west-1 ap-northeast-1 ap-southeast-2; do \
		aws s3 cp formation.zip s3://convox-$$region/release/$(VERSION)/formation.zip --acl public-read; \
	done
