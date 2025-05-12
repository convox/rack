#!/bin/bash
set -ex -o pipefail

root="$(cd $(dirname ${0:-})/..; pwd)"

# cli
convox version

# only deploy the example/httpd if not on full-convox-yaml ci test
if [ "${ACTION}" == "full-convox-yaml" ]; then
  cd $root/examples/full-convox-yaml
else
  cd $root/examples/httpd
fi

convox apps create ci2 --wait
convox apps | grep ci2
convox apps info ci2 | grep running
convox deploy -a ci2 --wait
convox apps info ci2 | grep running

# deploy multi-stage build app
if [ "${ACTION}" != "full-convox-yaml" ]; then
  cd $root/examples/multi-stage-build
  convox apps create ci3 --wait
  convox apps | grep ci3
  convox apps info ci3 | grep running
  convox deploy -a ci3 --wait
  convox apps info ci3 | grep running
fi
