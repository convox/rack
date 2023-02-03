#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

if [ "${ACTION}" == "update" ]; then
  # install the latest release, if ACTION is "update" it will be updated to the release candidate later on.
  convox rack install ${PROVIDER} --name ${RACK_NAME} ${ARGS}
else
  # install the release candidate, if ACTION is "downgrade" it will be downgraded to the latest release later
  convox rack install ${PROVIDER} --name ${RACK_NAME} --version ${VERSION} ${ARGS}
fi

convox instances

# set ci@convox.com as id
# convox config dir path
echo ci@convox.com > ~/.config/convox2/id
