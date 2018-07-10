#!/bin/bash
set -ex -o pipefail

# configure client id if on Travis CI
if [ ! -d "~/.convox/" ]; then
	mkdir -p ~/.convox/
	echo ci@convox.com > ~/.convox/id
fi
