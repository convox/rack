#!/bin/bash

export_secret() {
  echo "${2:-$1}=$(echo $SECRETS | jq -r ".${1}")" >> $GITHUB_ENV
}

export_secret AWS_ACCESS_KEY_ID
export_secret AWS_REGION AWS_DEFAULT_REGION # we also want to export AWS_DEFAULT_REGION for the aws cli config
export_secret AWS_REGION REGION
export_secret AWS_SECRET_ACCESS_KEY
export_secret SLACK_WEBHOOK_URL
export_secret DISCOURSE_API_KEY
