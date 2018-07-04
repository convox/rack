#!/bin/bash
set -ex -o pipefail

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp/artifacts}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}
export $($(dirname $0)/region.sh ${CIRCLE_NODE_INDEX})

convox rack install aws --name convox-${CIRCLE_BUILD_NUM} | tee $CIRCLE_ARTIFACTS/convox-installer.log
convox instances
