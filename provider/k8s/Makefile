all: generate

generate:
	# go get -u k8s.io/code-generator/...
	$(GOPATH)/src/k8s.io/code-generator/generate-groups.sh all \
		github.com/convox/rack/provider/k8s/pkg/client \
		github.com/convox/rack/provider/k8s/pkg/apis \
		convox:v1
