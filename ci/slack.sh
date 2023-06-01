#!/bin/bash

case "$TYPE" in
release)
  echo "rack release created: $VERSION"
  curl -s -X POST -d "payload={\"text\":\"rack release created: \`$VERSION\`\"}" $SLACK_WEBHOOK_URL
  ;;
publish)
  echo "rack release published: $VERSION"
  curl -s -X POST -d "payload={\"text\":\"rack release published: \`$VERSION\`\"}" $SLACK_WEBHOOK_URL
  ;;
esac
