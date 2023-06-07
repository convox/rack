#!/bin/bash
set -e

echo "version:$VERSION"

# check for version
[ -z "$VERSION" ] && echo "VERSION is empty" && exit 1

# link cli
aws s3 cp s3://convox/release/${VERSION}/cli/linux/convox s3://convox/cli/linux/convox --acl public-read --copy-props none
aws s3 cp s3://convox/release/${VERSION}/cli/linux/convox-arm64 s3://convox/cli/linux/convox-arm64 --acl public-read --copy-props none
aws s3 cp s3://convox/release/${VERSION}/cli/darwin/convox s3://convox/cli/darwin/convox --acl public-read --copy-props none
aws s3 cp s3://convox/release/${VERSION}/cli/darwin/convox-arm64 s3://convox/cli/darwin/convox-arm64 --acl public-read --copy-props none
aws s3 cp s3://convox/release/${VERSION}/cli/windows/convox.exe s3://convox/cli/windows/convox.exe --acl public-read --copy-props none
