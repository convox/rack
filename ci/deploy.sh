#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

repo=$1
app=$2
check=$3

mkdir -p /tmp/app
cd /tmp/app

git clone $repo .

convox deploy --app $app --wait

curl -ks --connect-timeout 5 --max-time 3 --retry 10 --retry-max-time 30 --retry-connrefused $check
