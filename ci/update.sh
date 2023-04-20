#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${ACTION}" == "update" ]; then
  convox rack update "${VERSION}" --wait | tee update-log.txt
fi

sleep 5

if grep -Fxq "_FAILED" update-log.txt; then
  echo "ok"
else
  exit 1;
fi

version=$(convox rack | grep Version | awk -F '  +' '{print $2}')
if [ "${version}" != "${VERSION}" ]; then
  exit 1;
fi

status=$(convox rack | grep Status | awk -F '  +' '{print $2}')
if [ "${status}" != "running" ]; then
  exit 1;
fi
