#!/usr/bin/env bash
# Tests for frustration-boundary.sh UserPromptSubmit hook
# Run: bash .claude/hooks/frustration-boundary_test.sh

set -euo pipefail

HOOK="$(cd "$(dirname "$0")" && pwd)/frustration-boundary.sh"
PASS=0
FAIL=0
TMPDIR_BASE=$(mktemp -d)

cleanup() {
  rm -rf "$TMPDIR_BASE"
}
trap cleanup EXIT

assert_eq() {
  local test_name="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    PASS=$((PASS + 1))
    echo "  PASS: $test_name"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: $test_name"
    echo "    expected: $expected"
    echo "    actual:   $actual"
  fi
}

assert_contains() {
  local test_name="$1" needle="$2" haystack="$3"
  if echo "$haystack" | grep -qF "$needle"; then
    PASS=$((PASS + 1))
    echo "  PASS: $test_name"
  else
    FAIL=$((FAIL + 1))
    echo "  FAIL: $test_name"
    echo "    expected to contain: $needle"
    echo "    actual: $haystack"
  fi
}

# Fixed window name for test isolation
TEST_WINDOW="test-window"

# Helper: run hook with given message, return stdout
run_hook() {
  local msg="$1"
  local project_dir="${2:-$TMPDIR_BASE/project}"
  mkdir -p "$project_dir/.orch/session/$TEST_WINDOW" 2>/dev/null || true
  echo "$msg" | CLAUDE_PROJECT_DIR="$project_dir" FRUSTRATION_THRESHOLD="${FRUSTRATION_THRESHOLD:-3}" FRUSTRATION_WINDOW_NAME="$TEST_WINDOW" bash "$HOOK" 2>/dev/null || true
}

# Helper: get counter value
get_counter() {
  local project_dir="${1:-$TMPDIR_BASE/project}"
  cat "$project_dir/.orch/session/$TEST_WINDOW/frustration_count" 2>/dev/null || echo "0"
}

# Helper: reset counter
reset_counter() {
  local project_dir="${1:-$TMPDIR_BASE/project}"
  rm -f "$project_dir/.orch/session/$TEST_WINDOW/frustration_count" 2>/dev/null || true
}

# ============================================================
echo "=== Test Group 1: Worker session skip ==="

project_with_spawn="$TMPDIR_BASE/worker-project"
mkdir -p "$project_with_spawn"
touch "$project_with_spawn/SPAWN_CONTEXT.md"

output=$(run_hook "this isn't working" "$project_with_spawn")
assert_eq "worker session produces no output" "" "$output"

# ============================================================
echo "=== Test Group 2: Empty message skip ==="

reset_counter
output=$(run_hook "")
assert_eq "empty message produces no output" "" "$output"

# ============================================================
echo "=== Test Group 3: SKIP_FRUSTRATION_BOUNDARY ==="

reset_counter
output=$(echo "this isn't working" | CLAUDE_PROJECT_DIR="$TMPDIR_BASE/project" SKIP_FRUSTRATION_BOUNDARY=1 bash "$HOOK" 2>/dev/null || true)
assert_eq "skip env var produces no output" "" "$output"

# ============================================================
echo "=== Test Group 4: Frustration signal detection ==="

# Test explicit frustration patterns
for pattern in "this isn't working" "we keep going in circles" "I'm frustrated" "this keeps failing" "still not working"; do
  reset_counter
  run_hook "$pattern" > /dev/null
  count=$(get_counter)
  assert_eq "detects explicit: '$pattern'" "1" "$count"
done

# Test correction patterns
for pattern in "no, not that" "I already said this" "as i mentioned before" "you're not listening"; do
  reset_counter
  run_hook "$pattern" > /dev/null
  count=$(get_counter)
  assert_eq "detects correction: '$pattern'" "1" "$count"
done

# Test abandon patterns
for pattern in "let's start over" "forget all of that" "scrap this" "I give up"; do
  reset_counter
  run_hook "$pattern" > /dev/null
  count=$(get_counter)
  assert_eq "detects abandon: '$pattern'" "1" "$count"
done

# ============================================================
echo "=== Test Group 5: Non-frustration messages ==="

for msg in "implement the login feature" "looks good, merge it" "can you read that file" "nice work" "let me think about this"; do
  reset_counter
  run_hook "$msg" > /dev/null
  count=$(get_counter)
  assert_eq "ignores clean: '$msg'" "0" "$count"
done

# ============================================================
echo "=== Test Group 6: Counter accumulation ==="

reset_counter
run_hook "this isn't working" > /dev/null
assert_eq "count after 1st signal" "1" "$(get_counter)"

run_hook "still not working" > /dev/null
assert_eq "count after 2nd signal" "2" "$(get_counter)"

run_hook "I'm frustrated" > /dev/null
assert_eq "count after 3rd signal" "3" "$(get_counter)"

# Clean messages should not increment counter
run_hook "try a different approach" > /dev/null
assert_eq "clean message doesn't increment" "3" "$(get_counter)"

# ============================================================
echo "=== Test Group 7: Threshold behavior ==="

reset_counter
# Below threshold: no output
output=$(run_hook "this isn't working")
assert_eq "signal 1: no output" "" "$output"

output=$(run_hook "still not working")
assert_eq "signal 2: no output" "" "$output"

# At threshold: boundary proposal
output=$(run_hook "I'm frustrated")
assert_contains "signal 3: has additionalContext" "additionalContext" "$output"
assert_contains "signal 3: mentions frustration boundary" "Frustration boundary" "$output"
assert_contains "signal 3: mentions threshold count" "3 frustration signals" "$output"

# Above threshold: still fires
output=$(run_hook "this keeps failing")
assert_contains "signal 4: still fires" "4 frustration signals" "$output"

# ============================================================
echo "=== Test Group 8: Custom threshold ==="

reset_counter
output=$(FRUSTRATION_THRESHOLD=1 run_hook "this isn't working" "$TMPDIR_BASE/project")
assert_contains "threshold=1 fires on first signal" "1 frustration signals" "$output"

# ============================================================
echo "=== Test Group 9: JSON output validity ==="

reset_counter
# Get to threshold
run_hook "this isn't working" > /dev/null
run_hook "still broken" > /dev/null
output=$(run_hook "I'm frustrated")

# Validate JSON
if echo "$output" | python3 -m json.tool > /dev/null 2>&1; then
  PASS=$((PASS + 1))
  echo "  PASS: output is valid JSON"
else
  FAIL=$((FAIL + 1))
  echo "  FAIL: output is not valid JSON"
  echo "    output: $output"
fi

# Check JSON structure
hook_event=$(echo "$output" | python3 -c "import json,sys; print(json.load(sys.stdin)['hookSpecificOutput']['hookEventName'])" 2>/dev/null || echo "PARSE_ERROR")
assert_eq "hookEventName is UserPromptSubmit" "UserPromptSubmit" "$hook_event"

# ============================================================
echo "=== Test Group 10: Case insensitivity ==="

reset_counter
run_hook "THIS ISN'T WORKING" > /dev/null
assert_eq "uppercase detected" "1" "$(get_counter)"

reset_counter
run_hook "This Isn't Working" > /dev/null
assert_eq "mixed case detected" "1" "$(get_counter)"

# ============================================================
echo ""
echo "Results: $PASS passed, $FAIL failed"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
