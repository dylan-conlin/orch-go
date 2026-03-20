#!/usr/bin/env bash
# Coordination Failure Controlled Demo — Complex/Ambiguous Task Runner
#
# Usage: ./run-complex.sh
#
# Spawns the complex/ambiguous task (table renderer) with both Haiku and Opus
# in isolated git worktrees. Scores individual results, then checks merge.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_DIR="$SCRIPT_DIR/results/complex-${TIMESTAMP}"
TASK_FILE="$SCRIPT_DIR/task-prompt-complex.md"
TIMEOUT_MINUTES=10
MODELS=("haiku" "opus")

mkdir -p "$RESULTS_DIR"

echo "=== Coordination Demo — Complex/Ambiguous Task ==="
echo "Project:  $PROJECT_DIR"
echo "Task:     $TASK_FILE"
echo "Models:   ${MODELS[*]}"
echo "Results:  $RESULTS_DIR"
echo "Timeout:  ${TIMEOUT_MINUTES}m per agent"
echo ""

# Get current commit for consistent baseline
BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)
echo "Baseline: $BASELINE_COMMIT"

# Read the task prompt
TASK_PROMPT=$(cat "$TASK_FILE")

run_agent() {
    local model="$1"
    local trial_dir="$RESULTS_DIR/trial-1/${model}"
    local worktree_dir="/tmp/coord-complex-${model}-$$"
    local branch_name="experiment/coord-complex-${model}-$$"

    mkdir -p "$trial_dir"

    echo "[${model}] Starting..."

    # Create worktree from baseline commit
    cd "$PROJECT_DIR"
    git worktree add -b "$branch_name" "$worktree_dir" "$BASELINE_COMMIT" 2>/dev/null

    # Record start time
    local start_time=$(date +%s)
    echo "$start_time" > "$trial_dir/start_time"

    # Map model aliases to full names
    local model_full=""
    case "$model" in
        haiku) model_full="claude-haiku-4-5-20251001" ;;
        opus)  model_full="claude-opus-4-5-20251101" ;;
    esac

    # Run claude with the task prompt
    cd "$worktree_dir"
    timeout "${TIMEOUT_MINUTES}m" env -u CLAUDECODE BEADS_NO_DAEMON=1 claude \
        --model "$model_full" \
        --dangerously-skip-permissions \
        -p "$TASK_PROMPT" \
        > "$trial_dir/stdout.log" 2>"$trial_dir/stderr.log" || true

    # Record end time
    local end_time=$(date +%s)
    echo "$end_time" > "$trial_dir/end_time"
    local duration=$((end_time - start_time))
    echo "$duration" > "$trial_dir/duration_seconds"

    echo "[${model}] Completed in ${duration}s"

    # Capture git state
    cd "$worktree_dir"
    git diff --stat HEAD > "$trial_dir/diff_stat.txt" 2>/dev/null || true
    git diff --numstat HEAD > "$trial_dir/numstat.txt" 2>/dev/null || true
    git diff HEAD > "$trial_dir/full_diff.txt" 2>/dev/null || true
    git log --oneline "$BASELINE_COMMIT"..HEAD > "$trial_dir/commits.txt" 2>/dev/null || true
    git status --short > "$trial_dir/git_status.txt" 2>/dev/null || true

    # Run tests and capture results
    cd "$worktree_dir"
    go test ./pkg/display/ -v > "$trial_dir/all_tests.txt" 2>&1 || true
    go build ./... > "$trial_dir/build_output.txt" 2>&1 || true

    # Copy modified/created files for analysis
    cp "$worktree_dir/pkg/display/display.go" "$trial_dir/display.go" 2>/dev/null || true
    cp "$worktree_dir/pkg/display/display_test.go" "$trial_dir/display_test.go" 2>/dev/null || true
    cp "$worktree_dir/pkg/display/table.go" "$trial_dir/table.go" 2>/dev/null || true
    cp "$worktree_dir/pkg/display/table_test.go" "$trial_dir/table_test.go" 2>/dev/null || true

    # Cleanup worktree
    cd "$PROJECT_DIR"
    git worktree remove "$worktree_dir" --force 2>/dev/null || true
    git branch -D "$branch_name" 2>/dev/null || true

    echo "[${model}] Results saved to $trial_dir"
}

# Run both agents (sequentially to avoid resource contention)
for model in "${MODELS[@]}"; do
    run_agent "$model"
done

echo ""
echo "=== Scoring ==="
bash "$SCRIPT_DIR/score-complex.sh" "$RESULTS_DIR"

# Measure accretion
echo ""
echo "=== Accretion ==="
bash "$SCRIPT_DIR/measure-accretion.sh" "$RESULTS_DIR"

echo ""
echo "=== Done ==="
echo "Results: $RESULTS_DIR"
