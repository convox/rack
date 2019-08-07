all: generate

generate:
	# go get -u k8s.io/code-generator/...
	../../vendor/k8s.io/code-generator/generate-groups.sh all \
		github.com/convox/rack/pkg/atom/pkg/client \
		github.com/convox/rack/pkg/atom/pkg/apis \
		convox:v1,v2
