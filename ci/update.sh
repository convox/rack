#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${ACTION}" == "update" ]; then
  convox rack update "${VERSION}" --wait | tee update-log.txt

  if grep -Fq "_FAILED" update-log.txt; then
    exit 1;
  else
    echo ok;
  fi

  version=$(convox rack | grep Version | awk -F '  +' '{print $2}')
  if [ "${version}" != "${VERSION}" ]; then
    exit 1;
  fi

  status=$(convox rack | grep Status | awk -F '  +' '{print $2}')
  if [ "${status}" != "running" ]; then
    exit 1;
  fi
fi
