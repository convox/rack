all: build

build:
	docker build --no-cache -t convox/agent .

test: build
	docker run -v /var/lib/boot2docker:/host/boot2docker --env-file .env convox/agent -log /host/boot2docker/docker.log -cwgroup test -cwstream test -tick 2 testapp testps i-0000000

vendor:
	godep save -r -copy=true ./...

release: build
	docker tag -f convox/agent:latest convox/agent:0.51
	docker push convox/agent:0.51
	AWS_DEFAULT_PROFILE=release aws s3 cp convox.conf s3://convox/agent/0.51/convox.conf --acl public-read
