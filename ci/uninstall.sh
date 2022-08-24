#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

# cleanup
if [ "${IS_UPDATE}" == "true" ]; then
  convox apps delete ci2 --wait
fi

convox rack uninstall ${PROVIDER} ${RACK_NAME} --force
