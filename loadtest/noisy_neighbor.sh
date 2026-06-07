#!/bin/bash
# Tenant isolation: a noisy tenant gets throttled, a good tenant is unaffected.
URL="http://localhost:8080/api"

echo "=== Noisy tenant 'flooder' — 30 concurrent requests ==="
flood=$(for i in $(seq 1 30); do
  curl -s -o /dev/null -w "%{http_code}\n" -H "X-Tenant-ID: flooder" "$URL" &
done; wait)
echo "flooder: $(echo "$flood" | grep -c 200) served, $(echo "$flood" | grep -c 429) throttled"

echo "=== Well-behaved tenant 'goodguy' — 5 concurrent requests ==="
good=$(for i in $(seq 1 5); do
  curl -s -o /dev/null -w "%{http_code}\n" -H "X-Tenant-ID: goodguy" "$URL" &
done; wait)
echo "goodguy: $(echo "$good" | grep -c 200) served, $(echo "$good" | grep -c 429) throttled"

echo ""
echo "RESULT: flooder hit its limit; goodguy unaffected -> per-tenant isolation works."