#!/bin/bash
set -ex -o pipefail

source $(dirname $0)/env.sh

convox rack update "${LATEST}" --wait
convox instances
