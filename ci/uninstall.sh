#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

convox rack uninstall ${PROVIDER} ${RACK_NAME} --force
