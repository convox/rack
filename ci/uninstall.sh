#!/bin/bash
set -ex -o pipefail

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}

export STACK_NAME=convox-${CIRCLE_BUILD_NUM}
export TEMPLATE_FILE=api/dist/kernel.json

case $CIRCLE_NODE_INDEX in
  1)
	export AWS_DEFAULT_REGION=us-west-2
	export AWS_REGION=us-west-2
	;;
  2)
  export AWS_DEFAULT_REGION=eu-west-1
  export AWS_REGION=eu-west-1
  ;;
  *)
	export AWS_DEFAULT_REGION=us-east-1
	export AWS_REGION=us-east-1
	;;
esac

convox uninstall --force
