.PHONY: all build builder compress dev generate mocks package release test

commands = build monitor rack router
injects  = convox-env

# commands = build rack router
# injects  =

assets   = $(wildcard assets/*)
binaries = $(addprefix $(GOPATH)/bin/, $(commands))
sources  = $(shell find . -name '*.go')
statics  = $(addprefix $(GOPATH)/bin/, $(injects))

all: build

build: $(binaries) $(statics)

builder:
	docker build -t convox/build:$(USER) -f cmd/build/Dockerfile .
	docker push convox/build:$(USER)

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
	kubectl patch deployment/api -p '{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"convox/rack:dev","env":[{"name":"VERSION","value":"dev"}]}]}}}}' -n $(RACK)
	kubectl patch deployment/router -p '{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"convox/rack:dev","env":[{"name":"VERSION","value":"dev"}]}]}}}}' -n convox-system
	kubectl delete pod --all -n convox-system
	kubectl delete pod --all -n $(RACK)
	kubectl rollout status deployment/api -n $(RACK)
	kubectl rollout status deployment/router -n convox-system
	convox rack logs

generate:
	go run cmd/generate/main.go controllers > pkg/api/controllers.go
	go run cmd/generate/main.go routes > pkg/api/routes.go
	go run cmd/generate/main.go sdk > sdk/methods.go

mocks: generate
	go get -u github.com/vektra/mockery/.../
	make -C pkg/structs mocks
	mockery -case underscore -dir pkg/start -outpkg sdk -output pkg/mock/start -name Interface
	mockery -case underscore -dir sdk -outpkg sdk -output pkg/mock/sdk -name Interface
	mockery -case underscore -dir vendor/github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface -outpkg aws -output pkg/mock/aws -name CloudWatchAPI
	mockery -case underscore -dir vendor/github.com/convox/stdcli -outpkg stdcli -output pkg/mock/stdcli -name Executor

package:
	$(GOPATH)/bin/packr

release: package
	make -C cmd/convox release VERSION=$(VERSION)
	make -C provider release VERSION=$(VERSION)
	docker build --pull -t convox/rack:$(VERSION) .
	docker push convox/rack:$(VERSION)

test:
	env PROVIDER=test go test -coverpkg ./... -covermode atomic -coverprofile coverage.txt ./...

$(binaries): $(GOPATH)/bin/%: $(sources)
	go install --ldflags="-s -w" ./cmd/$*

$(statics): $(GOPATH)/bin/%: $(sources)
	env CGO_ENABLED=0 go install --ldflags '-extldflags "-static" -s -w' ./cmd/$*
