#!/bin/bash
set -ex -o pipefail

root="$(cd $(dirname ${0:-})/..; pwd)"

# cli
convox version

# app
cd $root/examples/httpd
convox apps create ci2 --wait
convox apps | grep ci2
convox apps info ci2 | grep running
convox deploy -a ci2 --wait
convox apps info ci2 | grep running
