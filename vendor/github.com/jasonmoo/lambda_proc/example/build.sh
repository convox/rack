#!/bin/bash

set -e
set -x

GOOS=linux go build -o main

zip -r lambda.zip main index.js

# upload lambda.zip as lambda function
# echoes back values received as input on event object

# Sample event data:
# {
#   "key3": "value3",
#   "key2": "value2",
#   "key1": "value1"
# }

# Lambda Proc output:
# {
#   "proc_req_id": 0,
#   "error": null,
#   "data": {
#     "key1": "value1",
#     "key2": "value2",
#     "key3": "value3"
#   }
# }