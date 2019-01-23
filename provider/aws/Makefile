.PHONY: all build release

all: build

build:
	go install .

release:
	make -C lambda/autoscale release VERSION=$(VERSION)
	make -C lambda/formation release VERSION=$(VERSION)
	make -C lambda/lifecycle release VERSION=$(VERSION)
	make -C lambda/syslog release VERSION=$(VERSION)
	cat formation/rack.yml | sed -e "s/# VERSION-DEFAULT/Default: \"$(VERSION)\"/g" > rack.yml
	cfn-flip -j rack.yml rack.json
	aws s3 cp rack.json s3://convox/release/$(VERSION)/rack.json --acl public-read
	aws s3 cp rack.yml s3://convox/release/$(VERSION)/rack.yml --acl public-read
	aws s3 cp formation s3://convox/release/$(VERSION)/formation --recursive --acl public-read
