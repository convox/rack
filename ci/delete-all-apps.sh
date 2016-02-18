#!/bin/bash
set -x

# Helpers to wait for stack and deployment status
# Timeout after 10m
wait_for_app_not_found() {
  local c=0

  sleep 10

  while convox apps info --app $1 2>&1 | grep -v "ERROR: no such app"; do
    let c=c+1
    [ $c -gt 30 ] && exit 1

    sleep 20
  done
}

# convox apps doesn't have cli-formatted output
# kick off deletes
for app_name in $(convox api get /apps | jq -r '.[].name'); do
  convox apps delete $app_name
done

# wait for app to 404 because the stack is gone
for app_name in $(convox api get /apps | jq -r '.[].name'); do
  wait_for_app_not_found $app_name
done