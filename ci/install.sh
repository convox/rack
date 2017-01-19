#!/bin/bash
set -ex -o pipefail

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}
export CONVOX_EMAIL=ci@convox.com
export STACK_NAME=convox-${CIRCLE_BUILD_NUM}
export $($(dirname $0)/region.sh ${CIRCLE_NODE_INDEX})

convox install | tee $CIRCLE_ARTIFACTS/convox-installer.log

convox rack params set Autoscale=No

convox instances
