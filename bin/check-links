#!/bin/bash

urls=$(egrep -ri --exclude-dir=vendor 'DocsLink' ../* \
    | awk '{ print $3}' \
    | grep http \
    | cut -d '"' -f 2)

for url in $urls; do
    curl -sL -w "%{http_code} %{url_effective}\\n" $url -o /dev/null
done
