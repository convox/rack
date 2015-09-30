#!/bin/bash
set -ex

export CIRCLE_BUILD_NUM=${CIRCLE_BUILD_NUM:-0}

export APP_NAME=httpd-${CIRCLE_BUILD_NUM}
export STACK_NAME=convox-${CIRCLE_BUILD_NUM}

convox logs --app $STACK_NAME > $CIRCLE_ARTIFACTS/convox.log &

cd ci/examples/httpd

convox apps create $APP_NAME

while convox apps info --app $APP_NAME | grep -i creating; do
  echo "app creating"
  sleep 10
done

convox logs --app $APP_NAME > $CIRCLE_ARTIFACTS/$APP_NAME.log &

convox deploy --app $APP_NAME

sleep 240

curl -m2 http://$(convox apps info --app httpd-${CIRCLE_BUILD_NUM} | egrep -o 'httpd.*.amazonaws.com'):3000

convox apps delete $APP_NAME

while convox apps info --app $APP_NAME | grep -i deleting; do
  echo "app deleting"
  sleep 10
done
