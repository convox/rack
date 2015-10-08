#!/bin/bash
set -ex

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}

export CONVOX_EMAIL=ci@convox.com
export STACK_NAME=convox-${CIRCLE_BUILD_NUM}
export TEMPLATE_FILE=api/dist/kernel.json

convox install --disable-encryption
