.PHONY: all clean release test upload

all: lambda.zip

clean:
	rm -f lambda.zip handler

release: lambda.zip
	aws s3 cp lambda.zip s3://convox/release/$(VERSION)/lambda/lifecycle.zip --acl public-read
	for region in $(shell cat ../../REGIONS); do \
		aws s3 cp s3://convox/release/$(VERSION)/lambda/lifecycle.zip s3://convox-$$region/release/$(VERSION)/lambda/lifecycle.zip --acl public-read --region $$region; \
	done

test:
	bin/test $(RACK) $(INSTANCE)

upload: lambda.zip
	bin/upload $(RACK)

handler: *.go
	GOOS=linux GOARCH=amd64 go build -o handler

lambda.zip: handler
	zip -r lambda.zip handler
