PORT ?= 3000

.PHONY: default dev

all: build

dev:
	@export $(shell cat .env); docker-compose up

ami:
	docker pull convox/build
	docker run -v $(shell pwd):/build --env-file ~/.convox/.env.release convox/build -public convox /build

build:
	docker build -t convox/kernel .

release:
	bin/release
	aws s3 cp dist/kernel.json s3://convox/kernel.json --acl public-read

vendor:
	godep save -r -copy=true ./...

ssh:
	export AWS_DEFAULT_PROFILE=release; aws ec2 describe-instances --filters 'Name=tag:Name,Values=convox-web' 'Name=instance-state-name,Values=running' --query 'Reservations[0].Instances[0].PublicIpAddress'
