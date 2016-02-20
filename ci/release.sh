#!/bin/bash
set -x

# don't auto-publish until upgrade path is automatically tested
# curl -vik -X POST $RELEASE_URL/publish -d token=$RELEASE_TOKEN -d version=$VERSION