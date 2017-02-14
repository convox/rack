#!/bin/bash
set -e

for f in $HOME/.profile.d/*; do source $f; done

if [ "$#" -gt 0 ]; then
	exec "$@"
else
	exec /bin/bash
fi
