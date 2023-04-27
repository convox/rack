#!/bin/bash

# TODO: create and deploy the app "full-convox-yaml"

# timers
sleep 30

timerLog=$(convox logs -a ci2 --no-follow --since 1m | grep service/example)
if ! [[ $timerLog == *"Hello Timer"* ]]; then
  echo "failed"; exit 1;
fi
