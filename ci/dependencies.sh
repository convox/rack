#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

# install utilities
sudo apt-get update && sudo apt-get -y install awscli jq

# install docker
curl -s https://download.docker.com/linux/static/stable/x86_64/docker-18.09.6.tgz | sudo tar -C /usr/bin --strip-components 1 -xz

# install kubectl
curl -Ls https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/linux/amd64/kubectl -o /tmp/kubectl && \
	sudo mv /tmp/kubectl /usr/bin/kubectl && sudo chmod +x /usr/bin/kubectl

# install aws-iam-authenticator
curl -o /tmp/aws-iam-authenticator aws-iam-authenticator https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/linux/amd64/aws-iam-authenticator && \
	sudo mv /tmp/aws-iam-authenticator /usr/bin/aws-iam-authenticator && sudo chmod +x /usr/bin/aws-iam-authenticator

# download appropriate cli version
curl -o ${GOPATH}/bin/convox https://convox.s3.amazonaws.com/release/${VERSION}/cli/linux/convox
chmod +x ${GOPATH}/bin/convox

# set ci@convox.com as id
mkdir -p ~/.convox/
echo ci@convox.com > ~/.convox/id
