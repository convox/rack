#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

# install utilities
sudo apt-get update && sudo apt-get -y install awscli jq

# install kubectl
curl -Ls https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/amd64/kubectl -o /tmp/kubectl && \
	sudo mv /tmp/kubectl /usr/bin/kubectl && sudo chmod +x /usr/bin/kubectl

# download appropriate cli version
curl -o ${GOPATH}/bin/convox https://convox.s3.amazonaws.com/release/${VERSION}/cli/linux/convox
chmod +x ${GOPATH}/bin/convox

# set ci@convox.com as id
mkdir -p ~/.convox/
echo ci@convox.com > ~/.convox/id
