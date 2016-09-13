#!/bin/bash

set -e
set -x

#GOOS=linux go build -o main
# Using Docker linux image to compile golang natively
# you can't use CGO to link to system libraries.
# cross compiling makes you lose native DNS resolution and some other
# linux system libraries with CGO
# dependency obviously to install Docker 

# building vendor dependencies
docker run --rm -it -v "$PWD":/app -w /app qapi/gocker vendor

# compiling natively
docker run --rm -v "$PWD":/app -w /app qapi/gocker cross


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
