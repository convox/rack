.DEFAULT_GOAL=build
.PHONY: build test get run tags clean reset

clean:
	go clean .
	go env

get: clean
	rm -rf $(GOPATH)/src/github.com/aws/aws-sdk-go
	git clone --depth=1 https://github.com/aws/aws-sdk-go $(GOPATH)/src/github.com/aws/aws-sdk-go

	rm -rf $(GOPATH)/src/github.com/go-ini/ini
	git clone --depth=1 https://github.com/go-ini/ini $(GOPATH)/src/github.com/go-ini/ini

	rm -rf $(GOPATH)/src/github.com/jmespath/go-jmespath
	git clone --depth=1 https://github.com/jmespath/go-jmespath $(GOPATH)/src/github.com/jmespath/go-jmespath

	rm -rf $(GOPATH)/src/github.com/Sirupsen/logrus
	git clone --depth=1 https://github.com/Sirupsen/logrus $(GOPATH)/src/github.com/Sirupsen/logrus

	rm -rf $(GOPATH)/src/github.com/mjibson/esc
	git clone --depth=1 https://github.com/mjibson/esc $(GOPATH)/src/github.com/mjibson/esc

	rm -rf $(GOPATH)/src/github.com/crewjam/go-cloudformation
	git clone --depth=1 https://github.com/crewjam/go-cloudformation $(GOPATH)/src/github.com/crewjam/go-cloudformation

	rm -rf $(GOPATH)/src/github.com/spf13/cobra
	git clone --depth=1 https://github.com/spf13/cobra $(GOPATH)/src/github.com/spf13/cobra

	rm -rf $(GOPATH)/src/github.com/spf13/pflag
	git clone --depth=1 https://github.com/spf13/pflag $(GOPATH)/src/github.com/spf13/pflag

	rm -rf $(GOPATH)/src/github.com/asaskevich/govalidator
	git clone --depth=1 https://github.com/asaskevich/govalidator $(GOPATH)/src/github.com/asaskevich/govalidator
	
	rm -rf $(GOPATH)/src/github.com/fzipp/gocyclo
	git clone --depth=1 https://github.com/fzipp/gocyclo $(GOPATH)/src/github.com/fzipp/gocyclo

travisget: 
	rm -rf $(GOPATH)/src/github.com/mweagle/cloudformationresources
	git clone --depth=1 https://github.com/mweagle/cloudformationresources $(GOPATH)/src/github.com/mweagle/cloudformationresources
	
reset:
		git reset --hard
		git clean -f -d

generate: 
	go generate -x
	@echo "Generate complete: `date`"

validate: 
	go run $(GOPATH)/src/github.com/fzipp/gocyclo/gocyclo.go -over 15 .
	# Disable composites until https://github.com/golang/go/issues/9171 is resolved.  Currently
	# failing due to gocf.IAMPoliciesList literal initialization
	go tool vet -composites=false *.go
	go tool vet -composites=false ./explore
	go tool vet -composites=false ./aws/
	
format:
	go fmt .

travisci: get travisget generate validate
	go build .

build: format generate validate
	go build .
	@echo "Build complete"

docs:
	@echo ""
	@echo "Sparta godocs: http://localhost:8090/pkg/github.com/mweagle/Sparta"
	@echo
	godoc -v -http=:8090 -index=true

test: build
	go test -v .
	go test -v ./aws/...

run: build
	./sparta

tags:
	gotags -tag-relative=true -R=true -sort=true -f="tags" -fields=+l .

provision: build
	go run ./applications/hello_world.go --level info provision --s3Bucket $(S3_BUCKET)

execute: build
	./sparta execute

describe: build
	rm -rf ./graph.html
	go test -v -run TestDescribe
