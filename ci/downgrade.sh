#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${ACTION}" == "downgrade" ] && [ "${VERSION}" != "$(convox api get /system | jq -r '.version')" ]; then
  convox rack update "${LATEST}" --wait
fi
