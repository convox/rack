#!/bin/bash

region="$1"
[ -z $region ] && echo "Please provide a region name, e.g. eu-west-2" && exit 1

echo aws s3api create-bucket \
    --create-bucket-configuration "LocationConstraint=${region}" \
    --region "$region" \
    --bucket "convox-${region}"
