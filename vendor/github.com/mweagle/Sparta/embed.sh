#!/bin/bash -ex
rm -fv ./resources/provision/node_modules.zip
rm -rf ./resources/provision/node_modules

pushd ./resources/provision
npm install
popd

pushd ./resources/provision/
zip -r ./node_modules.zip ./node_modules/*
popd

# Create the embedded version
rm -rf ./resources/provision/node_modules
go run $GOPATH/src/github.com/mjibson/esc/main.go \
  -o ./CONSTANTS.go \
  -private \
  -pkg sparta \
  ./resources

# Cleanup the zip file that we just embedded
unzip -vl ./resources/provision/node_modules.zip
rm -fv ./resources/provision/node_modules.zip

# Create a secondary CONSTANTS_AWSBINARY.go file with empty content.  The next step will insert the
# build tags at the head of each file so that they are mutually exclusive, similar to the
# lambdabinaryshims.go file
go run $GOPATH/src/github.com/mjibson/esc/main.go \
  -o ./CONSTANTS_AWSBINARY.go \
  -private \
  -pkg sparta \
  ./resources/awsbinary/README.md

# Tag the builds...
go run ./resources/awsbinary/insertTags.go ./CONSTANTS !lambdabinary
go run ./resources/awsbinary/insertTags.go ./CONSTANTS_AWSBINARY lambdabinary