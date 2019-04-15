.PHONY: all build release

all: build

build:
	go install .

release:
	make -C lambda/autoscale release VERSION=$(VERSION)
	make -C lambda/formation release VERSION=$(VERSION)
	make -C lambda/lifecycle release VERSION=$(VERSION)
	make -C lambda/syslog release VERSION=$(VERSION)
	jq '.Parameters.Version.Default |= "$(VERSION)"' formation/rack.json | aws s3 cp - s3://convox/release/$(VERSION)/formation.json --acl public-read
	jq '.Parameters.Version.Default |= "$(VERSION)"' formation/rack.json | aws s3 cp - s3://convox/release/$(VERSION)/rack.json --acl public-read
	jq '.Parameters.Version.Default |= "$(VERSION)"' formation/rack.json | aws s3 cp - s3://convox/release/$(VERSION)/provider/aws/rack.json --acl public-read
	aws s3 cp formation s3://convox/release/$(VERSION)/formation --recursive --acl public-read
