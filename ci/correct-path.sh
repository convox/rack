#!/bin/bash

if [[ "$TRAVIS_REPO_SLUG" != "convox/rack" ]]; then
    cd ../..
    mkdir -p convox
    mv $TRAVIS_REPO_SLUG convox/rack

    export TRAVIS_BUILD_DIR="$PWD/convox/rack"
    cd $TRAVIS_BUILD_DIR
fi
