#!/usr/bin/env bash
# UserPromptSubmit hook: detect frustration signals and propose session boundary
# Only fires for interactive orchestrator sessions — workers skip this.
# Design: .kb/investigations/2026-03-27-design-frustration-detection-session-boundary.md (Track 1)
#
# How it works:
#   1. Reads user message from stdin
#   2. Pattern-matches for frustration signals (keywords, corrections, abandon signals)
#   3. Tracks signal count in a counter file scoped to tmux window
#   4. At threshold (default 3): injects boundary proposal via additionalContext
#
# Env overrides:
#   FRUSTRATION_THRESHOLD=N    — signals before proposal (default: 3)
#   SKIP_FRUSTRATION_BOUNDARY=1 — disable hook entirely

set -euo pipefail

# --- Early exits ---

# Disable via env
if [ "${SKIP_FRUSTRATION_BOUNDARY:-0}" = "1" ]; then
  exit 0
fi

# Skip for worker sessions (spawned agents have SPAWN_CONTEXT.md)
if [ -f "${CLAUDE_PROJECT_DIR:-.}/SPAWN_CONTEXT.md" ]; then
  exit 0
fi

# Read user message from stdin
user_message=$(cat)

# Skip empty messages
if [ -z "$user_message" ]; then
  exit 0
fi

# --- Frustration Signal Detection ---

# Convert to lowercase for matching
msg_lower=$(echo "$user_message" | tr '[:upper:]' '[:lower:]')

signal_detected=0

# Signal category 1: Explicit frustration
explicit_patterns=(
  "this isn't working"
  "this isnt working"
  "this is not working"
  "nothing is working"
  "going in circles"
  "we keep going"
  "try something completely different"
  "completely different approach"
  "i'm frustrated"
  "im frustrated"
  "this is frustrating"
  "what the hell"
  "what the fuck"
  "why won't this"
  "why wont this"
  "this keeps failing"
  "keeps failing"
  "still broken"
  "still doesn't work"
  "still doesnt work"
  "still not working"
)

for pattern in "${explicit_patterns[@]}"; do
  if [[ "$msg_lower" == *"$pattern"* ]]; then
    signal_detected=1
    break
  fi
done

# Signal category 2: Repeated correction
if [ "$signal_detected" -eq 0 ]; then
  correction_patterns=(
    "no, not that"
    "no not that"
    "i already said"
    "i already told"
    "again, the problem"
    "again the problem"
    "as i mentioned"
    "like i said"
    "i keep telling"
    "how many times"
    "i just said"
    "that's not what i"
    "thats not what i"
    "that is not what i"
    "wrong again"
    "no, i meant"
    "no i meant"
    "you're not listening"
    "youre not listening"
  )

  for pattern in "${correction_patterns[@]}"; do
    if [[ "$msg_lower" == *"$pattern"* ]]; then
      signal_detected=1
      break
    fi
  done
fi

# Signal category 3: Session abandon intent
if [ "$signal_detected" -eq 0 ]; then
  abandon_patterns=(
    "start over"
    "start fresh"
    "forget all of that"
    "forget everything"
    "scrap this"
    "scrap all of"
    "throw this away"
    "let's just stop"
    "lets just stop"
    "i give up"
    "forget it"
  )

  for pattern in "${abandon_patterns[@]}"; do
    if [[ "$msg_lower" == *"$pattern"* ]]; then
      signal_detected=1
      break
    fi
  done
fi

# No signal found — exit silently
if [ "$signal_detected" -eq 0 ]; then
  exit 0
fi

# --- Counter Management ---

# Get tmux window name for session scoping (fallback to "default")
# FRUSTRATION_WINDOW_NAME overrides tmux detection (used in tests)
window_name="${FRUSTRATION_WINDOW_NAME:-$(tmux display-message -p '#{window_name}' 2>/dev/null || echo "default")}"
counter_dir="${CLAUDE_PROJECT_DIR:-.}/.orch/session/${window_name}"
counter_file="${counter_dir}/frustration_count"

# Create directory if needed
mkdir -p "$counter_dir" 2>/dev/null

# Reset counter if file is older than 4 hours (stale session)
if [ -f "$counter_file" ]; then
  file_mod=$(stat -f %m "$counter_file" 2>/dev/null || stat -c %Y "$counter_file" 2>/dev/null || echo 0)
  now=$(date +%s)
  file_age=$((now - file_mod))
  if [ "$file_age" -gt 14400 ]; then
    rm -f "$counter_file"
  fi
fi

# Read and increment counter
current_count=0
if [ -f "$counter_file" ]; then
  current_count=$(cat "$counter_file" 2>/dev/null || echo 0)
fi
current_count=$((current_count + 1))
echo "$current_count" > "$counter_file"

# --- Threshold Check & Boundary Proposal ---

THRESHOLD=${FRUSTRATION_THRESHOLD:-3}

if [ "$current_count" -ge "$THRESHOLD" ]; then
  # Build the proposal message — no double quotes or backslashes to keep JSON safe
  msg="Frustration boundary: ${current_count} frustration signals detected this session (threshold: ${THRESHOLD}). This conversation may be fighting the user. Propose: 'This conversation is fighting you. Want to save the question and start fresh?' If they agree, write FRUSTRATION_BOUNDARY.md using template at .orch/templates/FRUSTRATION_BOUNDARY.md to .orch/session/${window_name}/FRUSTRATION_BOUNDARY.md, then suggest exiting to start a new session."

  cat <<EOF
{
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "${msg}"
  }
}
EOF
fi
