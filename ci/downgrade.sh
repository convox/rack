#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${ACTION}" == "downgrade" ]; then
  convox rack update "${LATEST}" --wait
fi
