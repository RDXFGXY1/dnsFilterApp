#!/usr/bin/env bash
set -euo pipefail

URL="${DNS_FILTER_HEALTH_URL:-http://127.0.0.1:8080/healthz}"
ATTEMPTS="${DNS_FILTER_HEALTH_ATTEMPTS:-30}"

for ((i=1; i<=ATTEMPTS; i++)); do
  if curl --fail --silent --show-error --max-time 2 "$URL" | grep -q '"ready":true'; then
    printf 'DNS Filter is healthy at %s\n' "$URL"
    exit 0
  fi
  sleep 1
done

printf 'DNS Filter did not become healthy after %s seconds.\n' "$ATTEMPTS" >&2
exit 1
