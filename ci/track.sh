#!/bin/bash

# Run the args, and track the command, exit code and time elapsed
#
# usage:
#  track delete-all-apps.sh
#  track echo hello
#  track false

SECONDS=0

track(){
  curl -vi https://api.segment.io/v1/track      \
    -H "Content-Type: application/json" -X POST \
    --user $SEGMENT_WRITE_KEY:                  \
    -d "{
      \"userId\": \"circleci\",
      \"event\": \"command\",
      \"properties\": {
        \"branch\": \"$CIRCLE_BRANCH\",
        \"code\": $2,
        \"cmd\": \"$1\",
        \"region\": \"$AWS_REGION\",
        \"seconds\": $3
      }
    }"
}

set -x

"$@"

track $1 $? $SECONDS