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
