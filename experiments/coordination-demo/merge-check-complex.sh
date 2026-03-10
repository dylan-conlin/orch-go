#!/usr/bin/env bash
# Coordination Failure Controlled Demo — Merge Conflict Checker (Complex Task)
#
# Usage: ./merge-check-complex.sh RESULTS_DIR
#
# Attempts to merge haiku and opus changes. Reports conflict type per file:
# - textual conflict (git can't auto-merge)
# - both-added conflict (both created same new file)
# - semantic conflict (merges clean but tests fail)
# - clean (merges clean and tests pass)

set -euo pipefail

RESULTS_DIR="${1:?Usage: merge-check-complex.sh RESULTS_DIR}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
MERGE_FILE="$RESULTS_DIR/merge_results.csv"

echo "trial,merge_result,conflict_files,conflict_detail,tests_after_merge" > "$MERGE_FILE"

echo "=== Merge Conflict Analysis (Complex Task) ==="
echo ""

for trial_dir in "$RESULTS_DIR"/trial-*/; do
    trial_num=$(basename "$trial_dir" | sed 's/trial-//')
    haiku_diff="$trial_dir/haiku/full_diff.txt"
    opus_diff="$trial_dir/opus/full_diff.txt"

    if [ ! -f "$haiku_diff" ] || [ ! -f "$opus_diff" ]; then
        echo "  Trial $trial_num: SKIP (missing diffs)"
        echo "$trial_num,skip,0,missing diffs,n/a" >> "$MERGE_FILE"
        continue
    fi

    if [ ! -s "$haiku_diff" ] || [ ! -s "$opus_diff" ]; then
        echo "  Trial $trial_num: SKIP (empty diffs)"
        echo "$trial_num,skip,0,empty diffs,n/a" >> "$MERGE_FILE"
        continue
    fi

    # Create temporary worktrees to test merge
    merge_base="/tmp/coord-merge-complex-trial${trial_num}-$$"
    baseline=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

    cd "$PROJECT_DIR"

    # Create branch for haiku changes
    haiku_branch="merge-complex-haiku-$$"
    git worktree add -b "$haiku_branch" "${merge_base}-haiku" "$baseline" 2>/dev/null
    cd "${merge_base}-haiku"
    git apply "$haiku_diff" 2>/dev/null && \
        git add -A && git commit -m "haiku changes" --no-verify 2>/dev/null || true

    # Create branch for opus changes
    cd "$PROJECT_DIR"
    opus_branch="merge-complex-opus-$$"
    git worktree add -b "$opus_branch" "${merge_base}-opus" "$baseline" 2>/dev/null
    cd "${merge_base}-opus"
    git apply "$opus_diff" 2>/dev/null && \
        git add -A && git commit -m "opus changes" --no-verify 2>/dev/null || true

    # Attempt merge (haiku base, merge opus into it)
    cd "${merge_base}-haiku"
    merge_output=$(git merge "$opus_branch" --no-edit 2>&1) || true

    if echo "$merge_output" | grep -q "CONFLICT"; then
        # Identify conflict types
        conflict_files=$(echo "$merge_output" | grep "CONFLICT" | wc -l | tr -d ' ')
        conflict_detail=""

        # Check for "both added" conflicts (new files created by both)
        both_added=$(echo "$merge_output" | grep "both added" | wc -l | tr -d ' ')
        content_conflicts=$(echo "$merge_output" | grep "content" | wc -l | tr -d ' ')

        if [ "$both_added" -gt 0 ]; then
            conflict_detail="both-added:${both_added}"
        fi
        if [ "$content_conflicts" -gt 0 ]; then
            if [ -n "$conflict_detail" ]; then
                conflict_detail="${conflict_detail}+content:${content_conflicts}"
            else
                conflict_detail="content:${content_conflicts}"
            fi
        fi

        echo "  Trial $trial_num: CONFLICT ($conflict_files files)"
        echo "    Detail: $conflict_detail"
        echo "    Conflicting files:"
        echo "$merge_output" | grep "CONFLICT" | sed 's/^/      /'

        echo "$trial_num,conflict,$conflict_files,$conflict_detail,n/a" >> "$MERGE_FILE"
        git merge --abort 2>/dev/null || true
    else
        # Clean merge — test if it works
        echo "  Trial $trial_num: CLEAN MERGE"
        test_output=$(go test ./pkg/display/ -v 2>&1) || true
        if echo "$test_output" | grep -q "^ok"; then
            echo "    Tests: PASS"
            echo "$trial_num,clean,0,clean merge,pass" >> "$MERGE_FILE"
        else
            echo "    Tests: FAIL (semantic conflict)"
            echo "    Failed tests:"
            echo "$test_output" | grep "FAIL" | head -5 | sed 's/^/      /'
            echo "$trial_num,semantic_conflict,0,merges but tests fail,fail" >> "$MERGE_FILE"
        fi
        echo "$test_output" > "$trial_dir/merge_test_output.txt"
    fi

    # Cleanup
    cd "$PROJECT_DIR"
    git worktree remove "${merge_base}-haiku" --force 2>/dev/null || true
    git worktree remove "${merge_base}-opus" --force 2>/dev/null || true
    git branch -D "$haiku_branch" 2>/dev/null || true
    git branch -D "$opus_branch" 2>/dev/null || true
done

echo ""
echo "Merge results written to: $MERGE_FILE"
