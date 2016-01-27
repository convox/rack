#!/bin/bash

curl -k -X POST $RELEASE_URL/publish -d token=$RELEASE_TOKEN -d version=$VERSION