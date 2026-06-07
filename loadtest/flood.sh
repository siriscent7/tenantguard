#!/bin/bash
# Concurrent flood test — the correct way to stress a rate limiter.
URL="http://localhost:8080/api"
TENANT="${1:-acme}"
N="${2:-20}"

results=$(for i in $(seq 1 $N); do
  curl -s -o /dev/null -w "%{http_code}\n" -H "X-Tenant-ID: $TENANT" "$URL" &
done; wait)

served=$(echo "$results" | grep -c 200)
throttled=$(echo "$results" | grep -c 429)
echo "Tenant '$TENANT': $N concurrent requests -> $served served (200), $throttled throttled (429)"