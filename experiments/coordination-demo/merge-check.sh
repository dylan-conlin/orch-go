#!/usr/bin/env bash
# Coordination Failure Controlled Demo — Merge Conflict Checker
#
# Usage: ./merge-check.sh RESULTS_DIR
#
# For each trial, attempts to merge haiku and opus changes into
# a single branch. Measures merge conflicts as coordination failure signal.

set -euo pipefail

RESULTS_DIR="${1:?Usage: merge-check.sh RESULTS_DIR}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
MERGE_FILE="$RESULTS_DIR/merge_results.csv"

echo "trial,merge_success,conflict_files,conflict_lines,notes" > "$MERGE_FILE"

echo "=== Merge Conflict Analysis ==="
echo ""

for trial_dir in "$RESULTS_DIR"/trial-*/; do
    trial_num=$(basename "$trial_dir" | sed 's/trial-//')
    haiku_diff="$trial_dir/haiku/full_diff.txt"
    opus_diff="$trial_dir/opus/full_diff.txt"

    if [ ! -f "$haiku_diff" ] || [ ! -f "$opus_diff" ]; then
        echo "  Trial $trial_num: SKIP (missing diffs)"
        echo "$trial_num,skip,0,0,missing diffs" >> "$MERGE_FILE"
        continue
    fi

    if [ ! -s "$haiku_diff" ] || [ ! -s "$opus_diff" ]; then
        echo "  Trial $trial_num: SKIP (empty diffs)"
        echo "$trial_num,skip,0,0,empty diffs" >> "$MERGE_FILE"
        continue
    fi

    # Create temporary worktrees to test merge
    merge_dir="/tmp/coord-merge-trial${trial_num}-$$"
    baseline=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

    cd "$PROJECT_DIR"

    # Create branch for haiku changes
    haiku_branch="merge-test-haiku-$$"
    git worktree add -b "$haiku_branch" "${merge_dir}-haiku" "$baseline" 2>/dev/null
    cd "${merge_dir}-haiku"
    git apply "$haiku_diff" 2>/dev/null && \
        git add -A && git commit -m "haiku changes" --no-verify 2>/dev/null || true

    # Create branch for opus changes
    cd "$PROJECT_DIR"
    opus_branch="merge-test-opus-$$"
    git worktree add -b "$opus_branch" "${merge_dir}-opus" "$baseline" 2>/dev/null
    cd "${merge_dir}-opus"
    git apply "$opus_diff" 2>/dev/null && \
        git add -A && git commit -m "opus changes" --no-verify 2>/dev/null || true

    # Attempt merge
    cd "${merge_dir}-haiku"
    merge_output=$(git merge "$opus_branch" --no-edit 2>&1) || true
    merge_status=$?

    if echo "$merge_output" | grep -q "CONFLICT"; then
        conflict_files=$(echo "$merge_output" | grep "CONFLICT" | wc -l | tr -d ' ')
        conflict_lines=$(git diff --check 2>/dev/null | wc -l | tr -d ' ' || echo "0")
        echo "  Trial $trial_num: CONFLICT ($conflict_files files, ~$conflict_lines lines)"
        echo "$trial_num,conflict,$conflict_files,$conflict_lines,merge conflict" >> "$MERGE_FILE"
        git merge --abort 2>/dev/null || true
    elif [ $merge_status -eq 0 ]; then
        echo "  Trial $trial_num: CLEAN MERGE"
        # Test if merged code still works
        go test ./pkg/display/ -v > "${trial_dir}/merge_test_output.txt" 2>&1 || true
        if grep -q "^ok" "${trial_dir}/merge_test_output.txt" 2>/dev/null; then
            echo "$trial_num,success,0,0,clean merge + tests pass" >> "$MERGE_FILE"
        else
            echo "$trial_num,semantic_conflict,0,0,merges but tests fail" >> "$MERGE_FILE"
            echo "    ⚠️  Semantic conflict: merge succeeded but tests fail"
        fi
    else
        echo "  Trial $trial_num: MERGE FAILED (non-conflict)"
        echo "$trial_num,error,0,0,merge failed: $merge_output" >> "$MERGE_FILE"
    fi

    # Cleanup
    cd "$PROJECT_DIR"
    git worktree remove "${merge_dir}-haiku" --force 2>/dev/null || true
    git worktree remove "${merge_dir}-opus" --force 2>/dev/null || true
    git branch -D "$haiku_branch" 2>/dev/null || true
    git branch -D "$opus_branch" 2>/dev/null || true
done

echo ""
echo "Merge results written to: $MERGE_FILE"
