.PHONY: all release templates

all: templates

release:
	make -C lambda/fluentd release VERSION=$(VERSION)
	make -C lambda/formation release VERSION=$(VERSION)
	make -C lambda/lifecycle release VERSION=$(VERSION)
	mkdir -p /tmp/release/$(VERSION)
	cd /tmp/release/$(VERSION)
	jq '.Parameters.Version.Default |= "$(VERSION)"' formation/rack.json > formation.json
	aws s3 cp formation.json s3://convox/release/$(VERSION)/formation.json --acl public-read
	aws s3 cp formation.json s3://convox/release/$(VERSION)/rack.json --acl public-read
	aws s3 cp formation s3://convox/release/$(VERSION)/formation --recursive --acl public-read

templates:
	go-bindata -nometadata -pkg=aws -o templates.go ./templates/...
