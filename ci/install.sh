#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${IS_UPDATE}" == "false" ]; then
  convox rack install ${PROVIDER} --name ${RACK_NAME} --version ${VERSION} ${ARGS}
else
  convox rack install ${PROVIDER} --name ${RACK_NAME} ${ARGS}
fi
convox instances
