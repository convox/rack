#!/bin/bash

# $1 is the http address
# $2 is the error message

result=$(curl -ks --max-time 10 --fail $1)

code=$?
echo "curl on $1 return exit code $code"

if [ $code -eq 0 ]
then
  echo "$2"
  exit 1
fi
