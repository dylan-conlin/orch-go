#!/usr/bin/env bash
# Score individual agent results for modification experiment
#
# Usage: ./score-modification.sh RESULTS_DIR

set -euo pipefail

RESULTS_DIR="${1:?Usage: score-modification.sh RESULTS_DIR}"
SCORE_FILE="$RESULTS_DIR/scores.csv"

echo "condition,trial,agent,completion,build,tests_pass,file_discipline,spec_match,duration_s,total" > "$SCORE_FILE"

score_agent() {
    local agent_dir="$1"
    local condition="$2"
    local trial="$3"
    local agent="$4"

    local f_completion=0
    local f_build=0
    local f_tests=0
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

    # Tests pass: do all tests pass?
    if [ -f "$agent_dir/all_tests.txt" ]; then
        if grep -q "^ok" "$agent_dir/all_tests.txt" 2>/dev/null; then
            f_tests=1
        elif grep -q "PASS" "$agent_dir/all_tests.txt" 2>/dev/null; then
            f_tests=1
        fi
    fi

    # File discipline: only expected files modified?
    if [ -f "$agent_dir/diff_stat.txt" ] && [ -s "$agent_dir/diff_stat.txt" ]; then
        local modified=$(grep -v '\.beads/' "$agent_dir/diff_stat.txt" | grep -c '|' 2>/dev/null || echo "0")
        local display_files=$(grep -c 'display' "$agent_dir/diff_stat.txt" 2>/dev/null || echo "0")
        # For modification tasks: should be <= 2 (display.go + display_test.go)
        if [ "$modified" -le 2 ] && [ "$display_files" -ge 1 ]; then
            f_files=1
        fi
    fi

    # Spec match: did the agent make the right modifications?
    if [ -f "$agent_dir/display.go" ]; then
        if [ "$agent" = "a" ]; then
            # Agent A: FormatDuration should now handle weeks
            if grep -q 'func FormatDuration' "$agent_dir/display.go" 2>/dev/null; then
                # Check for week-related logic (w suffix or week variable)
                if grep -qE '(weeks|"[0-9]+w"|week)' "$agent_dir/display.go" 2>/dev/null; then
                    f_spec=1
                fi
            fi
        else  # agent b
            # Agent B: Truncate should now use runes
            if grep -q 'func Truncate' "$agent_dir/display.go" 2>/dev/null; then
                # Check for rune-based logic
                if grep -qE '(\[\]rune|utf8|rune)' "$agent_dir/display.go" 2>/dev/null; then
                    f_spec=1
                fi
            fi
        fi
    fi

    # Duration
    if [ -f "$agent_dir/duration_seconds" ]; then
        duration=$(cat "$agent_dir/duration_seconds")
    fi

    local total=$((f_completion + f_build + f_tests + f_files + f_spec))

    echo "$condition,$trial,$agent,$f_completion,$f_build,$f_tests,$f_files,$f_spec,$duration,$total" >> "$SCORE_FILE"
    printf "  %-14s t%-2s agent-%s: comp=%d build=%d test=%d file=%d spec=%d  %d/5  (%ds)\n" \
        "$condition" "$trial" "$agent" \
        "$f_completion" "$f_build" "$f_tests" "$f_files" "$f_spec" "$total" "$duration"
}

echo "=== Individual Agent Scores ==="
echo "Legend: comp=completion build=build test=tests_pass file=file_discipline spec=spec_match"
echo ""

for condition_dir in "$RESULTS_DIR"/*/; do
    condition=$(basename "$condition_dir")
    [ ! -d "$condition_dir" ] && continue
    # Skip non-condition dirs
    [[ "$condition" =~ \. ]] && continue

    for trial_dir in "$condition_dir"trial-*/; do
        trial=$(basename "$trial_dir" | sed 's/trial-//')
        [ ! -d "$trial_dir" ] && continue

        for agent_dir in "$trial_dir"agent-*/; do
            agent=$(basename "$agent_dir" | sed 's/agent-//')
            [ ! -d "$agent_dir" ] && continue
            score_agent "$agent_dir" "$condition" "$trial" "$agent"
        done
    done
done

echo ""
echo "Scores written to: $SCORE_FILE"
