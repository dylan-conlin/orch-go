#!/usr/bin/env bash
# UserPromptSubmit hook: inject comprehension queue status (two-state)
# Only fires for orchestrator sessions — workers don't need this.
# ~100ms per invocation — negligible overhead

# Skip for worker sessions (spawned agents have SPAWN_CONTEXT.md)
# Interactive sessions at project root without SPAWN_CONTEXT.md are orchestrators.
if [ -f "${CLAUDE_PROJECT_DIR:-.}/SPAWN_CONTEXT.md" ]; then
  exit 0
fi

THRESHOLD=${COMPREHENSION_THRESHOLD:-5}

# Count unread (needs orchestrator review) — includes legacy pending
unread=$(bd list -l comprehension:unread -s closed --no-pager 2>/dev/null | grep -c "^orch-" || true)
legacy=$(bd list -l comprehension:pending -s closed --no-pager 2>/dev/null | grep -c "^orch-" || true)
unread_total=$((unread + legacy))

# Count processed (needs Dylan to read brief)
processed=$(bd list -l comprehension:processed -s closed --no-pager 2>/dev/null | grep -c "^orch-" || true)

if [ "$unread_total" -ge "$THRESHOLD" ]; then
  msg="comprehension:unread queue: ${unread_total} items — OVER THRESHOLD (${THRESHOLD}). Drain before releasing new work."
elif [ "$unread_total" -gt 0 ] || [ "$processed" -gt 0 ]; then
  msg="comprehension:unread queue: ${unread_total} items"
  if [ "$processed" -gt 0 ]; then
    msg="${msg}, comprehension:processed: ${processed} briefs to read"
  fi
else
  msg="comprehension:pending queue: 0 items"
fi

cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "${msg}"
  }
}
EOF
