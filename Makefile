.PHONY: all build builder generate mocks release templates test

all: build

build:
	go install ./...

builder:
	docker build -t convox/build:$(USER) -f cmd/build/Dockerfile .
	docker push convox/build:$(USER)

generate:
	go run cmd/generate/main.go controllers > pkg/api/controllers.go
	go run cmd/generate/main.go routes > pkg/api/routes.go
	go run cmd/generate/main.go sdk > sdk/methods.go

mocks:
	go get -u github.com/vektra/mockery/.../
	make -C pkg/structs mocks
	mockery -case underscore -dir pkg/start -outpkg sdk -output pkg/mock/start -name Interface
	mockery -case underscore -dir sdk -outpkg sdk -output pkg/mock/sdk -name Interface
	mockery -case underscore -dir vendor/github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface -outpkg aws -output pkg/mock/aws -name CloudWatchAPI
	mockery -case underscore -dir vendor/github.com/convox/stdcli -outpkg stdcli -output pkg/mock/stdcli -name Executor


release:
	make -C cmd/convox release VERSION=$(VERSION)
	make -C provider release VERSION=$(VERSION)
	docker build -t convox/rack:$(VERSION) .
	docker push convox/rack:$(VERSION)

templates:
	go get -u github.com/jteeuwen/go-bindata/...
	make -C pkg/sync templates

test:
	env PROVIDER=test go test -v -coverpkg ./... -covermode atomic -coverprofile coverage.txt ./...
