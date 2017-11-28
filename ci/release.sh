#!/bin/bash
set -e

VERSION="$(git rev-parse --short HEAD)"
export VERSION

make release
