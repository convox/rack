#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

# install utilities
sudo apt-get update && sudo apt-get -y install unzip jq curl

# install aws cli
curl -f "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# install docker
curl -fs https://download.docker.com/linux/static/stable/x86_64/docker-29.3.1.tgz | sudo tar -C /usr/bin --strip-components 1 -xz

# install docker-buildx plugin (required by Docker 29.x for BuildKit builds)
BUILDX_VERSION=0.22.0
sudo mkdir -p /usr/local/lib/docker/cli-plugins
sudo curl -fsL https://github.com/docker/buildx/releases/download/v${BUILDX_VERSION}/buildx-v${BUILDX_VERSION}.linux-amd64 \
  -o /usr/local/lib/docker/cli-plugins/docker-buildx
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-buildx

# install kubectl
curl -fLs https://storage.googleapis.com/kubernetes-release/release/v1.28.15/bin/linux/amd64/kubectl -o /tmp/kubectl && \
	sudo mv /tmp/kubectl /usr/bin/kubectl && sudo chmod +x /usr/bin/kubectl

# install aws-iam-authenticator
curl -fLs https://amazon-eks.s3-us-west-2.amazonaws.com/1.12.7/2019-03-27/bin/linux/amd64/aws-iam-authenticator -o /tmp/aws-iam-authenticator && \
	sudo mv /tmp/aws-iam-authenticator /usr/bin/aws-iam-authenticator && sudo chmod +x /usr/bin/aws-iam-authenticator

# download appropriate cli version, waiting for the release artifact to finish uploading
for i in $(seq 1 12); do
	if curl -fsSL -o ${GOPATH}/bin/convox https://convox.s3.amazonaws.com/release/${VERSION}/cli/linux/convox; then
		break
	fi

	if [ "${i}" -eq 12 ]; then
		echo "cli not available at release/${VERSION}/cli/linux/convox after 2m"
		exit 1
	fi

	sleep 10
done

chmod +x ${GOPATH}/bin/convox
