#!/usr/bin/env bash
# Coordination Failure Controlled Demo — Runner
#
# Usage: ./run.sh [--trials N] [--task TASK_FILE]
#
# Spawns the same task with both Haiku and Opus models in isolated
# git worktrees, waits for completion, then scores results.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results/$(date +%Y%m%d-%H%M%S)"
TASK_FILE="${TASK_FILE:-$SCRIPT_DIR/task-prompt.md}"
TRIALS="${1:-1}"
TIMEOUT_MINUTES=30
MODELS=("haiku" "opus")

mkdir -p "$RESULTS_DIR"

echo "=== Coordination Failure Controlled Demo ==="
echo "Project:  $PROJECT_DIR"
echo "Task:     $TASK_FILE"
echo "Trials:   $TRIALS"
echo "Models:   ${MODELS[*]}"
echo "Results:  $RESULTS_DIR"
echo "Timeout:  ${TIMEOUT_MINUTES}m per agent"
echo ""

# Get current commit for consistent baseline
BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)
echo "Baseline: $BASELINE_COMMIT"

# Read the task prompt
TASK_PROMPT=$(cat "$TASK_FILE")

run_trial() {
    local trial_num="$1"
    local model="$2"
    local trial_dir="$RESULTS_DIR/trial-${trial_num}/${model}"
    local worktree_dir="/tmp/coord-demo-${model}-$$"
    local branch_name="experiment/coord-demo-${model}-trial${trial_num}-$$"

    mkdir -p "$trial_dir"

    echo "[Trial $trial_num/$model] Starting..."
    echo "[Trial $trial_num/$model] Worktree: $worktree_dir"
    echo "[Trial $trial_num/$model] Branch: $branch_name"

    # Create worktree from baseline commit
    cd "$PROJECT_DIR"
    git worktree add -b "$branch_name" "$worktree_dir" "$BASELINE_COMMIT" 2>/dev/null

    # Record start time
    local start_time=$(date +%s)
    echo "$start_time" > "$trial_dir/start_time"

    # Run claude with the task prompt
    # Using claude CLI directly (not orch spawn) for isolation
    cd "$worktree_dir"
    timeout "${TIMEOUT_MINUTES}m" claude \
        --model "$model" \
        --dangerously-skip-permissions \
        -p "$TASK_PROMPT" \
        > "$trial_dir/stdout.log" 2>"$trial_dir/stderr.log" || true

    # Record end time
    local end_time=$(date +%s)
    echo "$end_time" > "$trial_dir/end_time"
    local duration=$((end_time - start_time))
    echo "$duration" > "$trial_dir/duration_seconds"

    echo "[Trial $trial_num/$model] Completed in ${duration}s"

    # Capture git state
    cd "$worktree_dir"
    git diff --stat HEAD > "$trial_dir/diff_stat.txt" 2>/dev/null || true
    git diff HEAD > "$trial_dir/full_diff.txt" 2>/dev/null || true
    git log --oneline "$BASELINE_COMMIT"..HEAD > "$trial_dir/commits.txt" 2>/dev/null || true
    git status --short > "$trial_dir/git_status.txt" 2>/dev/null || true

    # Run tests and capture results
    cd "$worktree_dir"
    go test ./pkg/display/ -v -run TestFormatBytes > "$trial_dir/test_output.txt" 2>&1 || true
    go test ./pkg/display/ -v > "$trial_dir/all_tests.txt" 2>&1 || true
    go build ./... > "$trial_dir/build_output.txt" 2>&1 || true

    # Copy modified files for analysis
    cp "$worktree_dir/pkg/display/display.go" "$trial_dir/display.go" 2>/dev/null || true
    cp "$worktree_dir/pkg/display/display_test.go" "$trial_dir/display_test.go" 2>/dev/null || true

    # Cleanup worktree
    cd "$PROJECT_DIR"
    git worktree remove "$worktree_dir" --force 2>/dev/null || true
    git branch -D "$branch_name" 2>/dev/null || true

    echo "[Trial $trial_num/$model] Results saved to $trial_dir"
}

# Run trials
for trial in $(seq 1 "$TRIALS"); do
    for model in "${MODELS[@]}"; do
        run_trial "$trial" "$model"
    done
done

# Run scoring
echo ""
echo "=== Scoring ==="
bash "$SCRIPT_DIR/score.sh" "$RESULTS_DIR"

echo ""
echo "=== Done ==="
echo "Results: $RESULTS_DIR"
