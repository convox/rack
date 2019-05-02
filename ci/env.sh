#!/bin/bash
set -e -o pipefail

if [ ! -f /tmp/convox-rack-name ]; then
  echo "${CIRCLE_BUILD_NUM}-$(date +"%H%M")" > /tmp/convox-rack-name
fi

export AWS_REGION=us-east-1
export RACK=$(cat /tmp/convox-rack-name)
export VERSION=${VERSION:-${CIRCLE_TAG}}
