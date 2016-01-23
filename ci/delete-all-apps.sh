#!/bin/bash
set -x

# convox apps doesn't have cli-formatted output
for app_name in $(convox api get /apps | jq -r '.[].name'); do
  convox apps delete $app_name

  # waiting for app delete
  while convox apps info --app $app_name | grep -i deleting; do
    echo "app deleting"
    sleep 20
  done
done
