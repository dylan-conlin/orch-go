#!/usr/bin/env bash
# Benchmark orch status performance with varying session counts
# Usage: ./scripts/benchmark_status.sh [session_count]
#
# This script:
# 1. Creates N test OpenCode sessions
# 2. Benchmarks orch status (compact mode)
# 3. Benchmarks orch status --all
# 4. Cleans up test sessions

set -euo pipefail

# Configuration
SESSION_COUNT=${1:-100}
OPENCODE_BIN=${OPENCODE_BIN:-opencode}
TEST_SESSION_PREFIX="benchmark-test"
RESULTS_FILE="benchmark_results_$(date +%Y%m%d_%H%M%S).txt"

echo "=== orch status Performance Benchmark ==="
echo "Session count: $SESSION_COUNT"
echo "Results will be written to: $RESULTS_FILE"
echo

# Cleanup function
cleanup() {
	echo
	echo "Cleaning up test sessions..."

	# Get all sessions and filter for our test sessions
	sessions=$($OPENCODE_BIN session list --json 2>/dev/null | jq -r '.[].id' || echo "")

	if [ -z "$sessions" ]; then
		echo "No sessions found to clean up"
		return
	fi

	# Count how many we're cleaning
	count=0
	for session_id in $sessions; do
		# Get session details to check title
		title=$($OPENCODE_BIN session get "$session_id" --json 2>/dev/null | jq -r '.title' || echo "")

		if [[ "$title" == "$TEST_SESSION_PREFIX"* ]]; then
			echo "Deleting test session: $session_id ($title)"
			$OPENCODE_BIN session delete "$session_id" 2>/dev/null || true
			((count++))
		fi
	done

	echo "Cleaned up $count test sessions"
}

# Trap cleanup on exit
trap cleanup EXIT

# Step 1: Create test sessions
echo "Step 1: Creating $SESSION_COUNT test sessions..."
session_ids=()
created=0
failed=0

for i in $(seq 1 $SESSION_COUNT); do
	title="${TEST_SESSION_PREFIX}-${i}"

	# Create a minimal session with title
	# Use a simple message to avoid LLM calls
	session_id=$($OPENCODE_BIN run --headless --title "$title" "echo test" 2>/dev/null | grep -o 'ses_[a-zA-Z0-9]*' | head -1 || echo "")

	if [ -n "$session_id" ]; then
		session_ids+=("$session_id")
		((created++))

		# Show progress every 10 sessions
		if [ $((i % 10)) -eq 0 ]; then
			echo "  Created $created/$SESSION_COUNT sessions..."
		fi
	else
		((failed++))
	fi

	# Small delay to avoid overwhelming OpenCode
	sleep 0.1
done

echo "Created $created sessions (failed: $failed)"
echo

# Verify sessions exist
echo "Verifying sessions are visible to OpenCode API..."
all_sessions=$($OPENCODE_BIN session list --json 2>/dev/null | jq -r '.[].id' | wc -l)
echo "Total sessions in OpenCode: $all_sessions"
echo

# Step 2: Benchmark orch status (compact mode)
echo "Step 2: Benchmarking 'orch status' (compact mode)..."
{
	echo "=== Benchmark Results ==="
	echo "Date: $(date)"
	echo "Total OpenCode sessions: $all_sessions"
	echo "Test sessions created: $created"
	echo
	echo "--- orch status (compact mode) ---"
} >"$RESULTS_FILE"

# Run 5 times and take average
total_time=0
for run in {1..5}; do
	start=$(date +%s.%N)
	orch status >/dev/null 2>&1
	end=$(date +%s.%N)

	duration=$(echo "$end - $start" | bc)
	echo "  Run $run: ${duration}s"
	echo "Run $run: ${duration}s" >>"$RESULTS_FILE"

	total_time=$(echo "$total_time + $duration" | bc)
done

avg_time=$(echo "scale=3; $total_time / 5" | bc)
echo "  Average: ${avg_time}s"
echo "Average: ${avg_time}s" >>"$RESULTS_FILE"
echo

# Step 3: Benchmark orch status --all
echo "Step 3: Benchmarking 'orch status --all'..."
{
	echo
	echo "--- orch status --all ---"
} >>"$RESULTS_FILE"

total_time_all=0
for run in {1..5}; do
	start=$(date +%s.%N)
	orch status --all >/dev/null 2>&1
	end=$(date +%s.%N)

	duration=$(echo "$end - $start" | bc)
	echo "  Run $run: ${duration}s"
	echo "Run $run: ${duration}s" >>"$RESULTS_FILE"

	total_time_all=$(echo "$total_time_all + $duration" | bc)
done

avg_time_all=$(echo "scale=3; $total_time_all / 5" | bc)
echo "  Average: ${avg_time_all}s"
echo "Average: ${avg_time_all}s" >>"$RESULTS_FILE"
echo

# Summary
{
	echo
	echo "=== Summary ==="
	echo "orch status (compact): ${avg_time}s avg"
	echo "orch status --all:     ${avg_time_all}s avg"
	echo "Difference:            $(echo "scale=3; $avg_time_all - $avg_time" | bc)s"
	echo
	echo "Performance target: <2s for compact mode"
	if (($(echo "$avg_time < 2.0" | bc -l))); then
		echo "✅ PASS: Compact mode is under 2s target"
	else
		echo "❌ FAIL: Compact mode exceeds 2s target"
	fi
} | tee -a "$RESULTS_FILE"

echo
echo "Results saved to: $RESULTS_FILE"
