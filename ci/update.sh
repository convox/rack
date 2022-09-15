#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${ACTION}" == "update" ]; then
  convox rack update "${VERSION}" --wait
fi
