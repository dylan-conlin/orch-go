#!/usr/bin/env bash
# UserPromptSubmit hook: inject comprehension:pending queue count
# ~50ms per invocation — negligible overhead

THRESHOLD=${COMPREHENSION_THRESHOLD:-5}

# Count non-empty lines excluding tip/warning lines from bd
count=$(bd list -l comprehension:pending -s closed --no-pager 2>/dev/null | grep -c "^orch-" || true)

if [ "$count" -ge "$THRESHOLD" ]; then
  msg="comprehension:pending queue: ${count} items — OVER THRESHOLD (${THRESHOLD}). Drain before releasing new work."
else
  msg="comprehension:pending queue: ${count} items"
fi

cat <<EOF
{"additionalContext": "${msg}"}
EOF
