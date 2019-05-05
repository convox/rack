#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${VERSION}" != "$(convox api get /system | jq -r '.version')" ]; then
  convox rack update "${VERSION}" --wait
fi
