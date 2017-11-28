#!/bin/bash
set -e

VERSION=${TRAVIS_COMMIT:7}
VERSION=${VERSION:-"$(git rev-parse --short HEAD)"}
export VERSION

make release
