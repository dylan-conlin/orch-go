#!/usr/bin/env bash
# Score individual agent results across all conditions
#
# Usage: ./score.sh RESULTS_DIR

set -euo pipefail

RESULTS_DIR="${1:?Usage: score.sh RESULTS_DIR}"
SCORE_FILE="$RESULTS_DIR/scores.csv"

echo "condition,task_type,trial,agent,completion,build,tests_pass,no_regression,file_discipline,spec_match,duration_s,total" > "$SCORE_FILE"

score_agent() {
    local agent_dir="$1"
    local condition="$2"
    local task_type="$3"
    local trial="$4"
    local agent="$5"

    local f_completion=0
    local f_build=0
    local f_tests=0
    local f_regression=0
    local f_files=0
    local f_spec=0
    local duration=0

    # Completion: did the agent produce changes?
    if [ -f "$agent_dir/full_diff.txt" ] && [ -s "$agent_dir/full_diff.txt" ]; then
        f_completion=1
    elif [ -f "$agent_dir/commits.txt" ] && [ -s "$agent_dir/commits.txt" ]; then
        f_completion=1
    fi

    # Build: does the code compile?
    if [ -f "$agent_dir/build_output.txt" ]; then
        if [ ! -s "$agent_dir/build_output.txt" ]; then
            f_build=1
        fi
    fi

    # Tests pass: do new tests pass?
    if [ -f "$agent_dir/all_tests.txt" ]; then
        if grep -q "^ok" "$agent_dir/all_tests.txt" 2>/dev/null; then
            f_tests=1
            f_regression=1
        elif grep -q "PASS" "$agent_dir/all_tests.txt" 2>/dev/null; then
            f_tests=1
            f_regression=1
        fi
    fi

    # File discipline: only expected files modified?
    if [ -f "$agent_dir/diff_stat.txt" ] && [ -s "$agent_dir/diff_stat.txt" ]; then
        # Count files modified (lines with |)
        # Exclude .beads/ from count (environment side-effect, not agent behavior)
        local modified=$(grep -v '\.beads/' "$agent_dir/diff_stat.txt" | grep -c '|' 2>/dev/null || echo "0")
        # For simple tasks: should be ≤2 (display.go + display_test.go)
        # For complex tasks: could be ≤4 (+ table.go + table_test.go)
        local max_files=2
        if [ "$task_type" = "complex" ]; then
            max_files=4
        fi
        # Only count display-related files
        local display_files=$(grep -c 'display' "$agent_dir/diff_stat.txt" 2>/dev/null || echo "0")
        if [ "$modified" -le "$max_files" ] && [ "$display_files" -ge 1 ]; then
            f_files=1
        fi
    fi

    # Spec match: does the expected function exist?
    if [ -f "$agent_dir/display.go" ]; then
        if [ "$agent" = "a" ]; then
            case "$task_type" in
                simple)
                    grep -q 'func FormatBytes' "$agent_dir/display.go" 2>/dev/null && f_spec=1
                    ;;
                complex)
                    grep -q 'func VisualWidth' "$agent_dir/display.go" 2>/dev/null && \
                    grep -q 'func PadToWidth' "$agent_dir/display.go" 2>/dev/null && f_spec=1
                    ;;
            esac
        else  # agent b
            case "$task_type" in
                simple)
                    grep -q 'func FormatRate' "$agent_dir/display.go" 2>/dev/null && f_spec=1
                    ;;
                complex)
                    grep -q 'func FormatTable' "$agent_dir/display.go" 2>/dev/null && f_spec=1
                    ;;
            esac
        fi
    fi

    # Duration
    if [ -f "$agent_dir/duration_seconds" ]; then
        duration=$(cat "$agent_dir/duration_seconds")
    fi

    local total=$((f_completion + f_build + f_tests + f_regression + f_files + f_spec))

    echo "$condition,$task_type,$trial,$agent,$f_completion,$f_build,$f_tests,$f_regression,$f_files,$f_spec,$duration,$total" >> "$SCORE_FILE"
    printf "  %-14s %-8s t%-2s agent-%s: comp=%d build=%d test=%d regr=%d file=%d spec=%d  %d/6  (%ds)\n" \
        "$condition" "$task_type" "$trial" "$agent" \
        "$f_completion" "$f_build" "$f_tests" "$f_regression" "$f_files" "$f_spec" "$total" "$duration"
}

echo "=== Individual Agent Scores ==="
echo "Legend: comp=completion build=build test=tests_pass regr=no_regression file=file_discipline spec=spec_match"
echo ""

for condition_dir in "$RESULTS_DIR"/*/; do
    condition=$(basename "$condition_dir")
    # Skip non-condition dirs
    [[ "$condition" == "metadata.json" ]] && continue
    [ ! -d "$condition_dir" ] && continue

    for task_dir in "$condition_dir"*/; do
        task_type=$(basename "$task_dir")
        [ ! -d "$task_dir" ] && continue

        for trial_dir in "$task_dir"trial-*/; do
            trial=$(basename "$trial_dir" | sed 's/trial-//')
            [ ! -d "$trial_dir" ] && continue

            for agent_dir in "$trial_dir"agent-*/; do
                agent=$(basename "$agent_dir" | sed 's/agent-//')
                [ ! -d "$agent_dir" ] && continue
                score_agent "$agent_dir" "$condition" "$task_type" "$trial" "$agent"
            done
        done
    done
done

echo ""
echo "Scores written to: $SCORE_FILE"
