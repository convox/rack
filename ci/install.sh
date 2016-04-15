#!/bin/bash
set -ex -o pipefail

export CIRCLE_ARTIFACTS=${CIRCLE_ARTIFACTS:-/tmp}
export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}

export CONVOX_EMAIL=ci@convox.com
export STACK_NAME=convox-${CIRCLE_BUILD_NUM}

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

convox install | tee $CIRCLE_ARTIFACTS/convox-installer.log

grep -v "Created Unknown" $CIRCLE_ARTIFACTS/convox-installer.log
