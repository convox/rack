#!/bin/bash
set -ex -o pipefail

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp/artifacts}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}
export STACK_NAME=convox-${CIRCLE_BUILD_NUM}
export $($(dirname $0)/region.sh ${CIRCLE_NODE_INDEX})

convox uninstall $STACK_NAME $AWS_REGION --force
