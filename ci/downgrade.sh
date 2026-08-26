#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${ACTION}" == "downgrade" ]; then
  LATEST=$(curl -fsS --connect-timeout 5 --max-time 20 https://api.github.com/repos/convox/rack/releases/latest | jq -r '.tag_name' || true)

  if [ -z "${LATEST}" ] || [ "${LATEST}" == "null" ]; then
    echo "could not resolve latest release tag from the github api"
    exit 1
  fi

  convox rack update "${LATEST}" --wait | tee downgrade-log.txt
  if grep -Fq "_FAILED" downgrade-log.txt; then
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
