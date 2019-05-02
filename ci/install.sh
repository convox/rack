#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

convox rack install ${PROVIDER} --name ${RACK_NAME} --version ${VERSION} ${ARGS}
convox instances
