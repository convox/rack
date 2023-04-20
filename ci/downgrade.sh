#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${ACTION}" == "downgrade" ]; then
  convox rack update "${LATEST}" --wait | tee downgrade-log.txt

  if grep -Fxq "_FAILED" downgrade-log.txt; then
    exit 1;
  else
    echo ok;
  fi

  version=$(convox rack | grep Version | awk -F '  +' '{print $2}')
  if [ "${version}" != "${LATEST}" ]; then
    exit 1;
  fi

  status=$(convox rack | grep Status | awk -F '  +' '{print $2}')
  if [ "${status}" != "running" ]; then
    exit 1;
  fi
fi
