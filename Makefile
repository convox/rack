.PHONY: all build builder clean clean-package compress dev generate mocks package release release-cli release-image release-provider release-version version-gen test test-docker

commands = build monitor rack
injects  = convox-env

assets   = $(wildcard assets/*)
binaries = $(addprefix $(GOPATH)/bin/, $(commands))
sources  = $(shell find . -name '*.go')
statics  = $(addprefix $(GOPATH)/bin/, $(injects))

DEV ?= true

ifeq ($(DEV), false)
VERSION := $(shell date +%Y%m%d%H%M%S)
else ifndef VERSION
VERSION := $(shell date +%Y%m%d%H%M%S)-dev
endif

all: build

build: $(binaries) $(statics)

builder:
	docker buildx build --platform linux/amd64 -t convox/build:$(VERSION) --no-cache --pull --push -f cmd/build/Dockerfile .
	docker buildx build --platform linux/arm64 -t convox/build:$(VERSION)-arm64 --no-cache --pull --push -f cmd/build/Dockerfile.arm .

clean: clean-package
	make -C cmd/convox clean

clean-package:
	find . -name '*-packr.go' -delete

compress: $(binaries) $(statics)
	upx-ucl -1 $^

dev:
	test -n "$(RACK)" # RACK
	docker build --target development -t convox/rack:dev .
ifdef UPLOAD
	docker push convox/rack:dev
	kubectl patch deployment/api -p '{"spec":{"template":{"spec":{"containers":[{"name":"main","imagePullPolicy":"Always"}]}}}}' -n $(RACK)
	kubectl patch deployment/router -p '{"spec":{"template":{"spec":{"containers":[{"name":"main","imagePullPolicy":"Always"}]}}}}' -n convox-system
endif
	convox rack update dev --wait
	kubectl delete pod --all -n convox-system
	kubectl delete pod --all -n $(RACK)
	kubectl rollout status deployment/api -n $(RACK)
	kubectl rollout status deployment/router -n convox-system
	convox rack logs

generate:
	go run cmd/generate/main.go controllers > pkg/api/controllers.go
	go run cmd/generate/main.go routes > pkg/api/routes.go
	go run cmd/generate/main.go sdk > sdk/methods.go

generate-provider:
	go run cmd/generate/main.go controllers > pkg/api/controllers.go
	go run cmd/generate/main.go routes > pkg/api/routes.go
	go run cmd/generate/main.go sdk > sdk/methods.go

mocks: generate-provider
	go get -u github.com/vektra/mockery/.../
	make -C pkg/structs mocks
	mockery -case underscore -dir pkg/start -outpkg sdk -output pkg/mock/start -name Interface
	mockery -case underscore -dir sdk -outpkg sdk -output pkg/mock/sdk -name Interface
	mockery -case underscore -dir vendor/github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface -outpkg aws -output pkg/mock/aws -name CloudWatchAPI
	mockery -case underscore -dir vendor/github.com/convox/stdcli -outpkg stdcli -output pkg/mock/stdcli -name Executor

package:
	go run -mod=vendor vendor/github.com/gobuffalo/packr/packr/main.go

regions:
	@aws-vault exec convox-release-standard go run provider/aws/cmd/regions/main.go
	@aws-vault exec convox-release-govcloud go run provider/aws/cmd/regions/main.go

release:
	test -n "$(VERSION)" # VERSION
	git tag $(VERSION) -m $(VERSION)
	git push origin refs/tags/$(VERSION)

release-all: release-version release-cli release-image builder release-provider

release-cli: release-version package
	make -C cmd/convox release VERSION=$(VERSION)

release-image: release-version package
	docker buildx build --platform linux/amd64 -t convox/rack:$(VERSION) --no-cache --pull --push .
	docker buildx build --platform linux/arm64 -t convox/rack:$(VERSION)-arm64 --no-cache --pull --push -f Dockerfile.arm .

release-provider: release-version package
	make -C provider release VERSION=$(VERSION)

release-version:
	test -n "$(VERSION)" # VERSION

version-gen:
	@echo $(shell date +%Y%m%d%H%M%S)

test:
	env PROVIDER=test go test -covermode atomic -coverprofile coverage.txt ./...

test-docker:
	docker build -t convox/rack:test --target development .
	docker run -it convox/rack:test make test

$(binaries): $(GOPATH)/bin/%: $(sources)
	env CGO_ENABLED=0 GOOS=linux go build -mod=vendor -tags=hidraw -ldflags="-extldflags=-static" -o $@ ./cmd/$*

$(statics): $(GOPATH)/bin/%: $(sources)
	env CGO_ENABLED=0 go install --ldflags '-extldflags "-static" -s -w' ./cmd/$*
