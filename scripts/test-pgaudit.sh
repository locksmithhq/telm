#!/usr/bin/env bash
# Calls every pgaudit-test endpoint in a loop so traces, SQL audit logs,
# and their correlations continuously appear in telm.
#
# Usage:
#   ./scripts/test-pgaudit.sh              # default: localhost:9001, 2s interval
#   BASE=http://myhost:9001 INTERVAL=1 ./scripts/test-pgaudit.sh

set -euo pipefail

BASE="${BASE:-http://localhost:9001}"
INTERVAL="${INTERVAL:-2}"

# Pretty-print JSON if jq is available, otherwise pass through
fmt() { command -v jq &>/dev/null && jq -c '.' || cat; }

echo "pgaudit-test loop  base=$BASE  interval=${INTERVAL}s"
echo "Open telm → Logs → filter service=postgres-audit to see audit entries"
echo "Each log with a trace_id links to a trace in the Traces tab."
echo "Press Ctrl-C to stop."
echo "────────────────────────────────────────────────────────────"

i=0
while true; do
  i=$((i + 1))
  ts=$(date '+%H:%M:%S')

  echo -n "[$ts] #$i  /select → "
  curl -sf "$BASE/select" | fmt

  echo -n "[$ts] #$i  /insert → "
  curl -sf "$BASE/insert" | fmt

  echo -n "[$ts] #$i  /ddl    → "
  curl -sf "$BASE/ddl" | fmt

  echo ""
  sleep "$INTERVAL"
done
