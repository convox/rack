#!/bin/bash
set -ex -o pipefail

root="$(cd $(dirname ${0:-})/..; pwd)"

# cli
convox2 version

# app
cd $root/examples/httpd
convox2 apps create ci2 --wait
convox2 apps | grep ci2
convox2 apps info ci2 | grep running
convox2 deploy -a ci2 --wait
convox2 apps info ci2 | grep running
