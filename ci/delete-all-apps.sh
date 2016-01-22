#!/bin/bash
set -x

for app_name in $(cx api get /apps | jq '.[].name'); do
  convox apps delete $app_name

  while convox apps info --app $app_name | grep -i deleting; do
    echo "app deleting"
    sleep 10
  done
done
