all: build

build:
	docker build -t convox/agent .

test: build
	docker run -v /var/lib/boot2docker:/host/boot2docker --env-file .env convox/agent -log /host/boot2docker/docker.log -cwgroup test -cwstream test testapp testps i-0000000
