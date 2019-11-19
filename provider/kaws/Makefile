all: all build generate release

all: build

build:
	go install .

generate:
	../../vendor/k8s.io/code-generator/generate-groups.sh all \
		github.com/convox/rack/provider/kaws/pkg/client \
		github.com/convox/rack/provider/kaws/pkg/apis \
		convox:v1

release:
	go run ../../cmd/template/main.go kaws $(VERSION) | aws s3 cp - s3://convox/release/$(VERSION)/provider/kaws/k8s/rack.yml --acl public-read
	sed -e 's/Version: dev/Version: $(VERSION)/g' formation/rack.yml | aws s3 cp - s3://convox/release/$(VERSION)/provider/kaws/cfn/rack.yml --acl public-read
