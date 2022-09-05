#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${EXTEND}" == "update" ]; then
  # install the latest release, it will be updated to the release candidate
  convox rack install ${PROVIDER} --name ${RACK_NAME} ${ARGS}
else
  # install the release candidate, if EXTEND is "downgrade" it will be downgraded to the latest release
  convox rack install ${PROVIDER} --name ${RACK_NAME} --version ${VERSION} ${ARGS}
fi

convox instances
