#!/usr/bin/env bash
# Unit tests for score.sh
#
# Creates synthetic trial directories with known outcomes
# and verifies score.sh produces expected scores.
#
# Usage: ./test_score.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

PASS=0
FAIL=0

assert_score() {
    local label="$1"
    local expected="$2"
    local actual="$3"
    if [ "$expected" = "$actual" ]; then
        echo "  PASS: $label"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $label"
        echo "    expected: $expected"
        echo "    actual:   $actual"
        FAIL=$((FAIL + 1))
    fi
}

# --- Test 1: Perfect simple agent-a (all factors = 1) ---
echo "=== Test 1: Perfect simple agent-a ==="
RESULTS="$TEST_DIR/test1"
agent_dir="$RESULTS/no-coord/simple/trial-1/agent-a"
mkdir -p "$agent_dir"

# completion: has diff
echo "diff content" > "$agent_dir/full_diff.txt"
echo "abc123 agent work" > "$agent_dir/commits.txt"
# build: empty file means success
touch "$agent_dir/build_output.txt"
# tests: ok line
echo "ok  pkg/display 0.5s" > "$agent_dir/all_tests.txt"
# file discipline: 2 display files
printf " pkg/display/display.go     | 20 ++++\n pkg/display/display_test.go | 30 ++++\n 2 files changed\n" > "$agent_dir/diff_stat.txt"
# spec match: has FormatBytes
echo "func FormatBytes(bytes int64) string {" > "$agent_dir/display.go"
# duration
echo "45" > "$agent_dir/duration_seconds"

bash "$SCRIPT_DIR/score.sh" "$RESULTS" > /dev/null 2>&1

# CSV: condition,task_type,trial,agent,completion,build,tests_pass,file_discipline,spec_match,duration_s,total
line=$(tail -1 "$RESULTS/scores.csv")
assert_score "total=5/5" "no-coord,simple,1,a,1,1,1,1,1,45,5" "$line"

# --- Test 2: Zero-score agent (no output at all) ---
echo "=== Test 2: Zero-score agent ==="
RESULTS="$TEST_DIR/test2"
agent_dir="$RESULTS/no-coord/simple/trial-1/agent-a"
mkdir -p "$agent_dir"

# All files empty or missing
touch "$agent_dir/full_diff.txt"   # empty = no changes
echo "build error" > "$agent_dir/build_output.txt"  # non-empty = build fail
echo "FAIL" > "$agent_dir/all_tests.txt"  # no "ok" line
touch "$agent_dir/diff_stat.txt"   # empty
# No display.go at all

bash "$SCRIPT_DIR/score.sh" "$RESULTS" > /dev/null 2>&1

line=$(tail -1 "$RESULTS/scores.csv")
assert_score "total=0/5" "no-coord,simple,1,a,0,0,0,0,0,0,0" "$line"

# --- Test 3: Partial score (completion + build only) ---
echo "=== Test 3: Partial score ==="
RESULTS="$TEST_DIR/test3"
agent_dir="$RESULTS/placement/complex/trial-3/agent-b"
mkdir -p "$agent_dir"

echo "abc123 agent work" > "$agent_dir/commits.txt"
echo "diff content" > "$agent_dir/full_diff.txt"
touch "$agent_dir/build_output.txt"  # empty = build ok
echo "FAIL  pkg/display" > "$agent_dir/all_tests.txt"  # tests fail
printf " pkg/display/display.go | 10 +\n pkg/display/table.go | 5 +\n pkg/display/display_test.go | 15 +\n 3 files changed\n" > "$agent_dir/diff_stat.txt"
echo "func SomethingElse() {}" > "$agent_dir/display.go"  # wrong function
echo "60" > "$agent_dir/duration_seconds"

bash "$SCRIPT_DIR/score.sh" "$RESULTS" > /dev/null 2>&1

line=$(tail -1 "$RESULTS/scores.csv")
# completion=1, build=1, tests=0, files=1 (3 display files, complex allows 4), spec=0
assert_score "total=3/5" "placement,complex,3,b,1,1,0,1,0,60,3" "$line"

# --- Test 4: Agent-b simple spec match ---
echo "=== Test 4: Agent-b spec match ==="
RESULTS="$TEST_DIR/test4"
agent_dir="$RESULTS/context-share/simple/trial-2/agent-b"
mkdir -p "$agent_dir"

echo "diff" > "$agent_dir/full_diff.txt"
touch "$agent_dir/build_output.txt"
echo "ok  pkg/display 0.3s" > "$agent_dir/all_tests.txt"
printf " pkg/display/display.go | 20 +\n pkg/display/display_test.go | 20 +\n 2 files changed\n" > "$agent_dir/diff_stat.txt"
echo "func FormatRate(bytesPerSec float64) string {" > "$agent_dir/display.go"
echo "30" > "$agent_dir/duration_seconds"

bash "$SCRIPT_DIR/score.sh" "$RESULTS" > /dev/null 2>&1

line=$(tail -1 "$RESULTS/scores.csv")
assert_score "total=5/5" "context-share,simple,2,b,1,1,1,1,1,30,5" "$line"

# --- Test 5: File discipline fails (too many files) ---
echo "=== Test 5: File discipline violation ==="
RESULTS="$TEST_DIR/test5"
agent_dir="$RESULTS/no-coord/simple/trial-1/agent-a"
mkdir -p "$agent_dir"

echo "diff" > "$agent_dir/full_diff.txt"
touch "$agent_dir/build_output.txt"
echo "ok  pkg/display 0.3s" > "$agent_dir/all_tests.txt"
# 3 files for simple task (max is 2)
printf " pkg/display/display.go | 20 +\n pkg/display/display_test.go | 20 +\n pkg/display/extra.go | 10 +\n 3 files changed\n" > "$agent_dir/diff_stat.txt"
echo "func FormatBytes(bytes int64) string {" > "$agent_dir/display.go"
echo "20" > "$agent_dir/duration_seconds"

bash "$SCRIPT_DIR/score.sh" "$RESULTS" > /dev/null 2>&1

line=$(tail -1 "$RESULTS/scores.csv")
assert_score "file_discipline=0" "no-coord,simple,1,a,1,1,1,0,1,20,4" "$line"

# --- Test 6: CSV header correctness ---
echo "=== Test 6: CSV header ==="
RESULTS="$TEST_DIR/test6"
mkdir -p "$RESULTS"
# Run with empty results to just get header
bash "$SCRIPT_DIR/score.sh" "$RESULTS" > /dev/null 2>&1
header=$(head -1 "$RESULTS/scores.csv")
assert_score "header has 11 columns (no no_regression)" \
    "condition,task_type,trial,agent,completion,build,tests_pass,file_discipline,spec_match,duration_s,total" \
    "$header"

# --- Summary ---
echo ""
echo "=== Results ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
echo "All tests passed."
