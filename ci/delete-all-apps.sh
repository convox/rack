#!/bin/bash
set -x

for app_name in $(convox api get /apps | jq -r '.[].name'); do
  convox apps delete $app_name

  while convox apps info --app $app_name | grep -i deleting; do
    echo "app deleting"
    sleep 10
  done

  sleep 5
done
