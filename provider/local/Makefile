all: all build release

all: build

build:
	go install .

release:
	go run ../../cmd/template/main.go local $(VERSION) | aws s3 cp - s3://convox/release/$(VERSION)/provider/local/k8s/rack.yml --acl public-read
