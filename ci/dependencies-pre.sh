#!/bin/bash
set -ex -o pipefail

VERSION=${VERSION:-ci}

# make artifacts dir
mkdir -p /tmp/artifacts

# install utilities
sudo apt-get install python-pip
curl -O http://stedolan.github.io/jq/download/linux64/jq && chmod +x jq && sudo mv jq /usr/local/bin
sudo pip install awscli --upgrade

# build and install with VERSION
go get -d github.com/convox/rack/cmd/convox
(
	cd ${GOPATH%%:*}/src/github.com/convox/rack/cmd/convox
	[ -n "$CIRCLE_BRANCH" ] && git fetch && git reset --hard origin/$CIRCLE_BRANCH
	go install -ldflags "-X main.Version=$VERSION"
)

# configure client id if on CircleCI
if [ ! -d "~/.convox/" ]; then
	mkdir -p ~/.convox/
	echo ci@convox.com > ~/.convox/id
fi
