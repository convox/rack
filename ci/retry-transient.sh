#!/bin/bash
# Retry a command only when it fails with a transport-level error. Anything else
# is a real failure and is returned immediately.

set -o pipefail

[ "$#" -gt 0 ] || { echo "usage: retry-transient.sh <command> [args...]"; exit 2; }

attempts=3
transient='send request failed|connection reset by peer|connection refused|broken pipe|use of closed network connection|TLS handshake timeout|i/o timeout|unexpected EOF|: EOF$|no such host'

log=$(mktemp) || exit 1
trap 'rm -f "${log}"' EXIT

for i in $(seq 1 ${attempts}); do
  "$@" 2>&1 | tee "${log}"
  rc=${PIPESTATUS[0]}

  if [ "${rc}" -eq 0 ]; then
    exit 0
  fi

  # only the tail, so transient noise from a phase that already succeeded cannot mask a real failure
  if ! tail -n 20 "${log}" | grep -Eq "${transient}"; then
    exit "${rc}"
  fi

  if [ "${i}" -eq "${attempts}" ]; then
    break
  fi

  echo "transient error on attempt ${i}, retrying in 30s: $*"
  sleep 30
done

echo "still failing after ${attempts} attempts with a transient error: $*"
exit "${rc}"
