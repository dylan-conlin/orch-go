#!/usr/bin/env bash
# Unit tests for analyze.sh
#
# Creates synthetic result directories with known merge outcomes and scores,
# then verifies analyze.sh produces correct aggregation.
#
# Usage: ./test_analyze.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

PASS=0
FAIL=0

assert_contains() {
    local label="$1"
    local pattern="$2"
    local file="$3"
    if grep -q "$pattern" "$file" 2>/dev/null; then
        echo "  PASS: $label"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $label (pattern '$pattern' not found)"
        FAIL=$((FAIL + 1))
    fi
}

# --- Test 1: Merge result aggregation ---
echo "=== Test 1: Merge result counts ==="
RESULTS="$TEST_DIR/test1"

# Create 5 trials: 2 success, 1 conflict, 1 build_fail, 1 semantic_conflict
for i in 1 2 3 4 5; do
    trial_dir="$RESULTS/no-coord/simple/trial-$i"
    mkdir -p "$trial_dir/agent-a" "$trial_dir/agent-b"
done

echo "success,0,clean merge + tests pass" > "$RESULTS/no-coord/simple/trial-1/merge_result.csv"
echo "success,0,clean merge + tests pass" > "$RESULTS/no-coord/simple/trial-2/merge_result.csv"
echo "conflict,1,merge conflict" > "$RESULTS/no-coord/simple/trial-3/merge_result.csv"
echo "build_fail,0,merged but build fails" > "$RESULTS/no-coord/simple/trial-4/merge_result.csv"
echo "semantic_conflict,0,merged but tests fail" > "$RESULTS/no-coord/simple/trial-5/merge_result.csv"

# Create scores.csv for analyze.sh to consume
cat > "$RESULTS/scores.csv" << 'EOF'
condition,task_type,trial,agent,completion,build,tests_pass,file_discipline,spec_match,duration_s,total
no-coord,simple,1,a,1,1,1,1,1,30,5
no-coord,simple,1,b,1,1,1,1,1,25,5
no-coord,simple,2,a,1,1,1,1,0,40,4
no-coord,simple,2,b,1,1,0,1,1,35,4
no-coord,simple,3,a,1,0,0,1,1,50,3
no-coord,simple,3,b,0,0,0,0,0,60,0
no-coord,simple,4,a,1,1,1,1,1,20,5
no-coord,simple,4,b,1,1,1,1,1,22,5
no-coord,simple,5,a,1,1,1,1,1,18,5
no-coord,simple,5,b,1,1,1,1,1,19,5
EOF

bash "$SCRIPT_DIR/analyze.sh" "$RESULTS" > /dev/null 2>&1

# Check analysis.md content
assert_contains "conflict count" "| no-coord | simple | 5 | 1 | 2 | 1 | 1 | 0 |" "$RESULTS/analysis.md"

# --- Test 2: Individual agent scores ---
echo "=== Test 2: Score aggregation ==="
# agent-a scores: 5,4,3,5,5 = 22 total, 3 perfect (5/5)
# agent-b scores: 5,4,0,5,5 = 19 total, 3 perfect
assert_contains "agent-a perfect count" "| no-coord | simple | a |" "$RESULTS/analysis.md"
assert_contains "agent-b in output" "| no-coord | simple | b |" "$RESULTS/analysis.md"

# --- Test 3: Duration summary present ---
echo "=== Test 3: Duration section exists ==="
# Create duration files
for i in 1 2 3 4 5; do
    echo "30" > "$RESULTS/no-coord/simple/trial-$i/agent-a/duration_seconds"
    echo "25" > "$RESULTS/no-coord/simple/trial-$i/agent-b/duration_seconds"
done

bash "$SCRIPT_DIR/analyze.sh" "$RESULTS" > /dev/null 2>&1
assert_contains "duration section" "## Duration Summary" "$RESULTS/analysis.md"
assert_contains "duration data" "no-coord/simple" "$RESULTS/analysis.md"

# --- Test 4: Multiple conditions ---
echo "=== Test 4: Multiple conditions ==="
RESULTS="$TEST_DIR/test4"

for cond in no-coord placement; do
    for i in 1 2 3; do
        trial_dir="$RESULTS/$cond/simple/trial-$i"
        mkdir -p "$trial_dir/agent-a" "$trial_dir/agent-b"
        echo "30" > "$trial_dir/agent-a/duration_seconds"
        echo "25" > "$trial_dir/agent-b/duration_seconds"
    done
done

# no-coord: 0 success, 3 conflict
echo "conflict,1,merge conflict" > "$RESULTS/no-coord/simple/trial-1/merge_result.csv"
echo "conflict,2,merge conflict" > "$RESULTS/no-coord/simple/trial-2/merge_result.csv"
echo "conflict,1,merge conflict" > "$RESULTS/no-coord/simple/trial-3/merge_result.csv"

# placement: 3 success, 0 conflict
echo "success,0,clean merge" > "$RESULTS/placement/simple/trial-1/merge_result.csv"
echo "success,0,clean merge" > "$RESULTS/placement/simple/trial-2/merge_result.csv"
echo "success,0,clean merge" > "$RESULTS/placement/simple/trial-3/merge_result.csv"

# Minimal scores.csv
cat > "$RESULTS/scores.csv" << 'EOF'
condition,task_type,trial,agent,completion,build,tests_pass,file_discipline,spec_match,duration_s,total
no-coord,simple,1,a,1,0,0,1,1,30,3
placement,simple,1,a,1,1,1,1,1,30,5
EOF

bash "$SCRIPT_DIR/analyze.sh" "$RESULTS" > /dev/null 2>&1

assert_contains "no-coord all conflicts" "| no-coord | simple | 3 | 3 | 0 |" "$RESULTS/analysis.md"
assert_contains "placement all success" "| placement | simple | 3 | 0 | 3 |" "$RESULTS/analysis.md"

# --- Summary ---
echo ""
echo "=== Results ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
echo "All tests passed."
