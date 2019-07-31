.PHONY: all clean release upload

all: lambda.zip

lambda.zip: handler
	zip -r lambda.zip handler

handler: *.go
	GOOS=linux GOARCH=amd64 go build -o handler

clean:
	rm -f lambda.zip handler

release: lambda.zip
	aws s3 cp lambda.zip s3://convox/release/$(VERSION)/lambda/syslog.zip --acl public-read
	for region in $(shell cat ../../REGIONS); do \
		aws s3 cp s3://convox/release/$(VERSION)/lambda/syslog.zip s3://convox-$$region/release/$(VERSION)/lambda/syslog.zip --acl public-read --region $$region; \
	done

upload: lambda.zip
	aws lambda update-function-code --function-name $(FUNCTION) --zip fileb://lambda.zip