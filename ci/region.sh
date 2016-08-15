#!/bin/sh

case "$1" in
  0)
    echo AWS_DEFAULT_REGION=us-east-1
    echo AWS_REGION=us-east-1
    ;;
  1)
    echo AWS_DEFAULT_REGION=us-west-2
    echo AWS_REGION=us-west-2
    ;;
  2)
    echo AWS_DEFAULT_REGION=eu-west-1
    echo AWS_REGION=eu-west-1
    ;;
  3)
    echo AWS_DEFAULT_REGION=ap-southeast-2
    echo AWS_REGION=ap-southeast-2
    ;;
  4)
    echo AWS_DEFAULT_REGION=eu-central-1
    echo AWS_REGION=eu-central-1
    ;;
  5)
    echo AWS_DEFAULT_REGION=ap-northeast-1
    echo AWS_REGION=ap-northeast-1
    ;;
  6)
    echo AWS_DEFAULT_REGION=ap-southeast-1
    echo AWS_REGION=ap-southeast-1
    ;;
  *)
    echo AWS_DEFAULT_REGION=unknown
    echo AWS_REGION=unknown
    ;;
esac

