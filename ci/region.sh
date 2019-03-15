#!/bin/sh

case "$1" in
  0)
    echo AWS_DEFAULT_REGION=us-east-1
    echo AWS_REGION=us-east-1
    echo RACK_BUILD_INSTANCE=t2.small
    ;;
  1)
    echo AWS_DEFAULT_REGION=us-west-2
    echo AWS_REGION=us-west-2
    echo RACK_PRIVATE=true
    ;;
  2)
    echo AWS_DEFAULT_REGION=eu-west-1
    echo AWS_REGION=eu-west-1
    ;;
  *)
    echo AWS_DEFAULT_REGION=unknown
    echo AWS_REGION=unknown
    ;;
esac

