all: build

build:
	docker build -t convox/agent .

test: 
	go get -t ./...
	go test -v -cover ./...

release: build
	docker tag -f convox/agent:latest convox/agent:0.3
	docker push convox/agent:0.3
	AWS_DEFAULT_PROFILE=release aws s3 cp convox.conf s3://convox/agent/0.3/convox.conf --acl public-read
