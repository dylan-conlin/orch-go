#!/usr/bin/env bash
# Coordination Failure Controlled Demo — Complex Task Scorer
#
# Usage: ./score-complex.sh RESULTS_DIR
#
# Scores each trial result across 10 dimensions:
#   F0: Completion (did the agent finish?)
#   F1: Compilation (does the code compile?)
#   F2: Test pass (do new tests pass?)
#   F3: Test regression (do existing tests still pass?)
#   F4: File discipline (only expected files modified/created?)
#   F5: VisualWidth spec match (correct signature + uses StripANSI?)
#   F6: RenderTable spec match (correct signature?)
#   F7: Doc comments (all public functions documented?)
#   F8: Multi-file (created table.go AND table_test.go?)
#   F9: No external deps (only stdlib imports?)

set -euo pipefail

RESULTS_DIR="${1:?Usage: score-complex.sh RESULTS_DIR}"
SCORE_FILE="$RESULTS_DIR/scores.csv"

echo "model,trial,F0,F1,F2,F3,F4,F5,F6,F7,F8,F9,duration_s,total_score,max_score" > "$SCORE_FILE"

score_trial() {
    local trial_dir="$1"
    local model=$(basename "$trial_dir")
    local trial_num=$(basename "$(dirname "$trial_dir")" | sed 's/trial-//')

    local f0=0  # Completion
    local f1=0  # Compilation
    local f2=0  # New tests pass
    local f3=0  # No regression
    local f4=0  # File discipline
    local f5=0  # VisualWidth uses StripANSI
    local f6=0  # RenderTable exists
    local f7=0  # Doc comments
    local f8=0  # Multi-file (table.go + table_test.go)
    local f9=0  # No external deps
    local duration=0

    # F0: Completion — did the agent produce any changes?
    if [ -f "$trial_dir/full_diff.txt" ] && [ -s "$trial_dir/full_diff.txt" ]; then
        f0=1
    elif [ -f "$trial_dir/commits.txt" ] && [ -s "$trial_dir/commits.txt" ]; then
        f0=1
    fi

    # F1: Compilation — does the code build?
    if [ -f "$trial_dir/build_output.txt" ]; then
        if [ ! -s "$trial_dir/build_output.txt" ]; then
            f1=1  # Empty output = successful build
        fi
    fi

    # F2: New tests pass
    if [ -f "$trial_dir/all_tests.txt" ]; then
        if grep -q "^ok" "$trial_dir/all_tests.txt" 2>/dev/null; then
            f2=1
        fi
    fi

    # F3: No regression — do all existing tests still pass?
    # Check for FAIL in test output — if no FAIL and there's output, existing tests pass
    if [ -f "$trial_dir/all_tests.txt" ]; then
        if grep -q "^ok" "$trial_dir/all_tests.txt" 2>/dev/null && \
           ! grep -q "^FAIL" "$trial_dir/all_tests.txt" 2>/dev/null; then
            f3=1
        fi
    fi

    # F4: File discipline — only expected files modified?
    # Expected: display.go (modified), display_test.go (modified),
    #           table.go (new), table_test.go (new)
    if [ -f "$trial_dir/diff_stat.txt" ] && [ -s "$trial_dir/diff_stat.txt" ]; then
        local modified_files=$(grep -c '|' "$trial_dir/diff_stat.txt" 2>/dev/null || echo "0")
        local display_files=$(grep -c 'display' "$trial_dir/diff_stat.txt" 2>/dev/null || echo "0")
        # Allow up to 4 display-related files + maybe go.mod/go.sum for deps
        if [ "$modified_files" -le 4 ] && [ "$display_files" -ge 1 ]; then
            f4=1
        fi
    elif [ -f "$trial_dir/git_status.txt" ] && [ -s "$trial_dir/git_status.txt" ]; then
        local modified_files=$(wc -l < "$trial_dir/git_status.txt" | tr -d ' ')
        if [ "$modified_files" -le 5 ]; then
            f4=1
        fi
    fi

    # F5: VisualWidth spec — correct signature AND uses StripANSI
    if [ -f "$trial_dir/display.go" ]; then
        if grep -q 'func VisualWidth(s string) int' "$trial_dir/display.go" 2>/dev/null; then
            # Check it uses StripANSI
            if grep -A 10 'func VisualWidth' "$trial_dir/display.go" | grep -q 'StripANSI' 2>/dev/null; then
                f5=1
            fi
        fi
    fi

    # F6: RenderTable spec — correct signature exists
    if [ -f "$trial_dir/table.go" ]; then
        if grep -q 'func RenderTable(headers \[\]string, rows \[\]\[\]string) string' "$trial_dir/table.go" 2>/dev/null; then
            f6=1
        fi
    fi

    # F7: Doc comments — all public functions have comments
    local doc_score=1
    for file in "$trial_dir/display.go" "$trial_dir/table.go"; do
        if [ -f "$file" ]; then
            # Find public functions without preceding comment
            # Look for func lines starting with uppercase that don't have // on the line before
            local funcs=$(grep -n '^func [A-Z]' "$file" 2>/dev/null || true)
            while IFS= read -r func_line; do
                if [ -z "$func_line" ]; then continue; fi
                local line_num=$(echo "$func_line" | cut -d: -f1)
                local prev_line=$((line_num - 1))
                if [ "$prev_line" -gt 0 ]; then
                    local prev_content=$(sed -n "${prev_line}p" "$file" 2>/dev/null || echo "")
                    if ! echo "$prev_content" | grep -q '//' 2>/dev/null; then
                        doc_score=0
                        break 2
                    fi
                fi
            done <<< "$funcs"
        fi
    done
    f7=$doc_score

    # F8: Multi-file — created both table.go AND table_test.go
    if [ -f "$trial_dir/table.go" ] && [ -f "$trial_dir/table_test.go" ]; then
        f8=1
    fi

    # F9: No external deps — only stdlib imports
    local ext_dep=0
    for file in "$trial_dir/display.go" "$trial_dir/table.go"; do
        if [ -f "$file" ]; then
            # Check for non-stdlib imports (contains a dot in import path)
            if grep -A 20 '^import' "$file" | grep -q '"[a-z]*\.[a-z]' 2>/dev/null; then
                ext_dep=1
            fi
        fi
    done
    if [ "$ext_dep" -eq 0 ]; then
        f9=1
    fi

    # Duration
    if [ -f "$trial_dir/duration_seconds" ]; then
        duration=$(cat "$trial_dir/duration_seconds")
    fi

    local total=$((f0 + f1 + f2 + f3 + f4 + f5 + f6 + f7 + f8 + f9))

    echo "$model,$trial_num,$f0,$f1,$f2,$f3,$f4,$f5,$f6,$f7,$f8,$f9,$duration,$total,10" >> "$SCORE_FILE"
    printf "  %-6s: F0=%d F1=%d F2=%d F3=%d F4=%d F5=%d F6=%d F7=%d F8=%d F9=%d  total=%d/10  (%ds)\n" \
        "$model" "$f0" "$f1" "$f2" "$f3" "$f4" "$f5" "$f6" "$f7" "$f8" "$f9" "$total" "$duration"
}

echo "Scoring results in: $RESULTS_DIR"
echo ""
echo "  Legend:"
echo "    F0=completion  F1=compilation  F2=tests-pass  F3=no-regression"
echo "    F4=file-discipline  F5=VisualWidth-spec  F6=RenderTable-spec"
echo "    F7=doc-comments  F8=multi-file  F9=no-ext-deps"
echo ""

# Find and score all trial directories
for trial_dir in "$RESULTS_DIR"/trial-*/*/; do
    if [ -d "$trial_dir" ]; then
        score_trial "$trial_dir"
    fi
done

echo ""
echo "Scores written to: $SCORE_FILE"
