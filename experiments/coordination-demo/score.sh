#!/usr/bin/env bash
# Coordination Failure Controlled Demo — Scorer
#
# Usage: ./score.sh RESULTS_DIR
#
# Scores each trial result across 6 failure dimensions:
#   F0: Completion (did the agent finish?)
#   F1: Compilation (does the code compile?)
#   F2: Test pass (do new tests pass?)
#   F3: Test regression (do existing tests still pass?)
#   F4: File discipline (only expected files modified?)
#   F5: Code quality (function matches spec?)

set -euo pipefail

RESULTS_DIR="${1:?Usage: score.sh RESULTS_DIR}"
SCORE_FILE="$RESULTS_DIR/scores.csv"

echo "model,trial,F0_completion,F1_compilation,F2_new_tests,F3_regression,F4_file_discipline,F5_spec_match,duration_s,total_score" > "$SCORE_FILE"

score_trial() {
    local trial_dir="$1"
    local model=$(basename "$trial_dir")
    local trial_num=$(basename "$(dirname "$trial_dir")" | sed 's/trial-//')

    local f0=0  # Completion
    local f1=0  # Compilation
    local f2=0  # New tests pass
    local f3=0  # No regression
    local f4=0  # File discipline
    local f5=0  # Spec match
    local duration=0

    # F0: Completion — did the agent produce any changes?
    if [ -f "$trial_dir/full_diff.txt" ] && [ -s "$trial_dir/full_diff.txt" ]; then
        f0=1
    elif [ -f "$trial_dir/commits.txt" ] && [ -s "$trial_dir/commits.txt" ]; then
        f0=1
    fi

    # F1: Compilation — does the code build?
    if [ -f "$trial_dir/build_output.txt" ]; then
        if ! grep -q "^" "$trial_dir/build_output.txt" 2>/dev/null || \
           [ ! -s "$trial_dir/build_output.txt" ]; then
            f1=1  # Empty output = successful build
        fi
    fi

    # F2: New tests pass — does TestFormatBytes pass?
    if [ -f "$trial_dir/test_output.txt" ]; then
        if grep -q "^ok" "$trial_dir/test_output.txt" 2>/dev/null; then
            f2=1
        elif grep -q "PASS" "$trial_dir/test_output.txt" 2>/dev/null; then
            f2=1
        fi
    fi

    # F3: No regression — do all existing tests still pass?
    if [ -f "$trial_dir/all_tests.txt" ]; then
        if grep -q "^ok" "$trial_dir/all_tests.txt" 2>/dev/null; then
            f3=1
        elif grep -q "PASS" "$trial_dir/all_tests.txt" 2>/dev/null; then
            f3=1
        fi
    fi

    # F4: File discipline — only display.go and display_test.go modified?
    if [ -f "$trial_dir/diff_stat.txt" ] && [ -s "$trial_dir/diff_stat.txt" ]; then
        local modified_files=$(grep -c '|' "$trial_dir/diff_stat.txt" 2>/dev/null || echo "0")
        local expected_files=$(grep -c 'display' "$trial_dir/diff_stat.txt" 2>/dev/null || echo "0")
        if [ "$modified_files" -le 2 ] && [ "$expected_files" -ge 1 ]; then
            f4=1
        fi
    elif [ -f "$trial_dir/git_status.txt" ] && [ -s "$trial_dir/git_status.txt" ]; then
        local modified_files=$(wc -l < "$trial_dir/git_status.txt" | tr -d ' ')
        if [ "$modified_files" -le 2 ]; then
            f4=1
        fi
    fi

    # F5: Spec match — does FormatBytes function exist with correct signature?
    if [ -f "$trial_dir/display.go" ]; then
        if grep -q 'func FormatBytes(bytes int64) string' "$trial_dir/display.go" 2>/dev/null; then
            f5=1
        elif grep -q 'func FormatBytes' "$trial_dir/display.go" 2>/dev/null; then
            f5=0  # Function exists but wrong signature
        fi
    fi

    # Duration
    if [ -f "$trial_dir/duration_seconds" ]; then
        duration=$(cat "$trial_dir/duration_seconds")
    fi

    local total=$((f0 + f1 + f2 + f3 + f4 + f5))

    echo "$model,$trial_num,$f0,$f1,$f2,$f3,$f4,$f5,$duration,$total" >> "$SCORE_FILE"
    printf "  %-6s trial %s: F0=%d F1=%d F2=%d F3=%d F4=%d F5=%d  total=%d/6  (%ds)\n" \
        "$model" "$trial_num" "$f0" "$f1" "$f2" "$f3" "$f4" "$f5" "$total" "$duration"
}

echo "Scoring results in: $RESULTS_DIR"
echo ""
echo "  Legend: F0=completion F1=compilation F2=new-tests F3=no-regression F4=file-discipline F5=spec-match"
echo ""

# Find and score all trial directories
for trial_dir in "$RESULTS_DIR"/trial-*/*/; do
    if [ -d "$trial_dir" ]; then
        score_trial "$trial_dir"
    fi
done

echo ""
echo "Scores written to: $SCORE_FILE"

# Summary statistics
echo ""
echo "=== Summary ==="
for model in haiku opus; do
    local_scores=$(grep "^$model," "$SCORE_FILE" | cut -d',' -f10)
    local_count=$(echo "$local_scores" | wc -l | tr -d ' ')
    local_sum=0
    local_durations=""
    while IFS=',' read -r m t f0 f1 f2 f3 f4 f5 dur total; do
        if [ "$m" = "$model" ]; then
            local_sum=$((local_sum + total))
            local_durations="$local_durations $dur"
        fi
    done < <(tail -n +2 "$SCORE_FILE")

    if [ "$local_count" -gt 0 ]; then
        local_avg=$((local_sum * 100 / local_count))
        printf "  %-6s: %d trials, avg score: %d.%02d/6\n" \
            "$model" "$local_count" "$((local_avg / 100))" "$((local_avg % 100))"
    fi
done

# Merge conflict analysis
echo ""
echo "=== Merge Conflict Analysis ==="
echo "(Run merge-check.sh to test if haiku + opus changes can be merged)"
