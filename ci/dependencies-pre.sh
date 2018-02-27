#!/bin/bash
set -ex -o pipefail

VERSION=${VERSION:-ci}

# make artifacts dir
mkdir -p /tmp/artifacts

# install utilities
sudo apt-get install python-pip
curl -O http://stedolan.github.io/jq/download/linux64/jq && chmod +x jq && sudo mv jq /usr/local/bin
sudo pip install awscli --upgrade

# download appropriate cli version
curl -o $GOPATH/bin/convox https://convox.s3.amazonaws.com/release/$VERSION/cli/linux/convox

# configure client id if on CircleCI
if [ ! -d "~/.convox/" ]; then
	mkdir -p ~/.convox/
	echo ci@convox.com > ~/.convox/id
fi
