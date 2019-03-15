.PHONY: all clean release run test upload

TEST_FUNCTION:=

all: lambda.zip

lambda.zip: handler
	zip -r lambda.zip handler

handler: *.go
	GOOS=linux GOARCH=amd64 go build -o handler

clean:
	rm -f lambda.zip handler

release: lambda.zip
	for region in $(shell cat ../../../../REGIONS); do \
		aws s3 cp lambda.zip s3://convox-$$region/release/$(VERSION)/lambda/autoscale.zip --acl public-read --region $$region; \
	done

run:
	aws lambda invoke --function-name test-autoscale --log-type Tail /tmp/output.txt | jq -r .LogResult | base64 -D

test:
	bin/test $(RACK) $(INSTANCE)

upload: lambda.zip
	aws lambda update-function-code --function-name test-autoscale --zip fileb://lambda.zip
