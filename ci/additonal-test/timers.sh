#!/bin/bash

set -e -o pipefail

attempts=30

# the timer fires every minute; poll rather than sampling a single window
for i in $(seq 1 ${attempts}); do
  timerLog=$(convox logs -a ci2 --no-follow --since 5m | grep service/example || true)

  if [[ $timerLog == *"Hello Timer"* ]]; then
    echo "timer output found: ${timerLog}"
    exit 0
  fi

  if [ "${i}" -eq "${attempts}" ]; then
    break
  fi

  echo "waiting for timer output from service/example..."
  sleep 10
done

echo "no 'Hello Timer' output from service/example in ${attempts} attempts"
exit 1
