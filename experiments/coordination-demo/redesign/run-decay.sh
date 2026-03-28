#!/usr/bin/env bash
# Attractor Decay Experiment — How fast does placement degrade when codebase changes?
#
# Design: 3 phases of codebase mutation with ORIGINAL placement prompts:
#   Phase 1 (rename):      Rename FormatDurationShort -> FormatElapsed
#   Phase 2 (reorganize):  Move StripANSI to separate ansi.go file
#   Phase 3 (alternatives): Add new functions creating competing insertion points
#
# Each phase uses the ORIGINAL placement prompts that reference the pre-mutation codebase.
# N=3 trials per phase (simple task only) to measure degradation.
#
# Usage:
#   ./run-decay.sh                    # Run all phases, N=3
#   ./run-decay.sh --trials 5         # Run all phases, N=5
#   ./run-decay.sh --phase rename     # Run single phase

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/prompts"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_BASE="$SCRIPT_DIR/results"

# Defaults
TRIALS=3
PHASES=("rename" "reorganize" "alternatives")
MODEL="haiku"
MODEL_FULL="claude-haiku-4-5-20251001"
TIMEOUT_MINUTES=10

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --phase) PHASES=("$2"); shift 2 ;;
        --trials) TRIALS="$2"; shift 2 ;;
        --model) MODEL="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

# Resolve model
case "$MODEL" in
    haiku) MODEL_FULL="claude-haiku-4-5-20251001" ;;
    opus)  MODEL_FULL="claude-opus-4-5-20251101" ;;
    sonnet) MODEL_FULL="claude-sonnet-4-5-20250514" ;;
    *) MODEL_FULL="$MODEL" ;;
esac

RESULTS_DIR="$RESULTS_BASE/$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

echo "=== Attractor Decay Experiment ==="
echo "Project:    $PROJECT_DIR"
echo "Baseline:   $BASELINE_COMMIT"
echo "Model:      $MODEL ($MODEL_FULL)"
echo "Trials:     $TRIALS per phase"
echo "Phases:     ${PHASES[*]}"
echo "Results:    $RESULTS_DIR"
echo ""

# Save metadata
cat > "$RESULTS_DIR/metadata.json" << EOF
{
  "experiment": "attractor-decay",
  "timestamp": "$TIMESTAMP",
  "baseline_commit": "$BASELINE_COMMIT",
  "model": "$MODEL",
  "model_full": "$MODEL_FULL",
  "trials_per_phase": $TRIALS,
  "phases": $(printf '%s\n' "${PHASES[@]}" | jq -R . | jq -s .),
  "timeout_minutes": $TIMEOUT_MINUTES,
  "description": "Tests how placement coordination degrades when codebase changes underneath stale attractors"
}
EOF

# --- Mutation functions ---
# Each creates a branch with mutations applied to the baseline

apply_rename_mutation() {
    local wt="$1"
    cd "$wt"

    # Rename FormatDurationShort -> FormatElapsed in display.go
    sed -i '' 's/FormatDurationShort/FormatElapsed/g' pkg/display/display.go
    sed -i '' 's/FormatDurationShort/FormatElapsed/g' pkg/display/display_test.go

    # Also update any callers in the codebase so it compiles
    find . -name '*.go' -not -path './.git/*' -exec grep -l 'FormatDurationShort' {} \; | while read -r f; do
        sed -i '' 's/FormatDurationShort/FormatElapsed/g' "$f"
    done

    git add -A
    git commit -m "mutation: rename FormatDurationShort -> FormatElapsed" --no-verify 2>/dev/null || true
}

apply_reorganize_mutation() {
    local wt="$1"
    cd "$wt"

    # Create ansi.go with StripANSI and ansiRegex moved from display.go
    cat > pkg/display/ansi.go << 'GOEOF'
package display

import "regexp"

// ansiRegex matches ANSI escape sequences (colors, formatting, etc.)
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// StripANSI removes ANSI escape codes from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
GOEOF

    # Create ansi_test.go with TestStripANSI moved from display_test.go
    cat > pkg/display/ansi_test.go << 'GOEOF'
package display

import "testing"

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"\x1b[31mred text\x1b[0m", "red text"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"no ansi here", "no ansi here"},
		{"", ""},
	}
	for _, tt := range tests {
		got := StripANSI(tt.input)
		if got != tt.want {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
GOEOF

    # Remove StripANSI and ansiRegex from display.go
    # Use a Go-aware approach: remove the lines
    python3 -c "
import re
with open('pkg/display/display.go', 'r') as f:
    content = f.read()

# Remove ansiRegex var declaration (2 lines)
content = re.sub(r'// ansiRegex matches.*\n.*regexp\.MustCompile.*\n\n', '', content)

# Remove StripANSI function (3 lines + comment)
content = re.sub(r'// StripANSI removes.*\nfunc StripANSI.*\n\treturn.*\n\}\n\n', '', content)

# Remove regexp import if no longer needed
# Check if regexp is still used
if 'regexp' not in content.split('import')[1].split(')')[0] or 'regexp.' not in content.split(')')[1]:
    content = re.sub(r'\t\"regexp\"\n', '', content)

with open('pkg/display/display.go', 'w') as f:
    f.write(content)
"

    # Remove TestStripANSI from display_test.go
    python3 -c "
import re
with open('pkg/display/display_test.go', 'r') as f:
    content = f.read()

# Remove the entire TestStripANSI function
content = re.sub(r'func TestStripANSI\(t \*testing\.T\) \{.*?\n\}\n\n', '', content, flags=re.DOTALL)

with open('pkg/display/display_test.go', 'w') as f:
    f.write(content)
"

    # Verify it compiles
    go build ./pkg/display/ 2>/dev/null || {
        echo "WARNING: reorganize mutation broke build"
        return 1
    }

    git add -A
    git commit -m "mutation: move StripANSI to ansi.go" --no-verify 2>/dev/null || true
}

apply_alternatives_mutation() {
    local wt="$1"
    cd "$wt"

    # Add competing functions AFTER FormatDurationShort
    # These create semantically plausible alternative insertion points
    python3 -c "
with open('pkg/display/display.go', 'r') as f:
    content = f.read()

# Find the end of FormatDurationShort and add new functions after it
new_functions = '''

// FormatDurationCompact formats a duration into the most compact representation.
// Output: \"0s\", \"45s\", \"3m\", \"2h\", \"3d\".
func FormatDurationCompact(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf(\"%ds\", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf(\"%dm\", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf(\"%dh\", int(d.Hours()))
	}
	return fmt.Sprintf(\"%dd\", int(d.Hours())/24)
}

// FormatTimestamp formats a time as a human-readable timestamp string.
// Output: \"2026-03-22 14:30:05\"
func FormatTimestamp(t time.Time) string {
	return t.Format(\"2006-01-02 15:04:05\")
}
'''

# Insert after the closing brace of FormatDurationShort
# Find 'func FormatDurationShort' and then its closing }
import re
pattern = r'(func FormatDurationShort\(d time\.Duration\) string \{.*?\n\})'
match = re.search(pattern, content, re.DOTALL)
if match:
    insert_pos = match.end()
    content = content[:insert_pos] + new_functions + content[insert_pos:]

with open('pkg/display/display.go', 'w') as f:
    f.write(content)
"

    # Add tests for the new functions
    python3 -c "
with open('pkg/display/display_test.go', 'r') as f:
    content = f.read()

new_tests = '''
func TestFormatDurationCompact(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{30 * time.Second, \"30s\"},
		{0, \"0s\"},
		{5 * time.Minute, \"5m\"},
		{2 * time.Hour, \"2h\"},
		{48 * time.Hour, \"2d\"},
	}
	for _, tt := range tests {
		got := FormatDurationCompact(tt.input)
		if got != tt.want {
			t.Errorf(\"FormatDurationCompact(%v) = %q, want %q\", tt.input, got, tt.want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	t1 := time.Date(2026, 3, 22, 14, 30, 5, 0, time.UTC)
	got := FormatTimestamp(t1)
	want := \"2026-03-22 14:30:05\"
	if got != want {
		t.Errorf(\"FormatTimestamp() = %q, want %q\", got, want)
	}
}
'''

# Append before the final closing (just append at end)
content = content.rstrip() + '\n' + new_tests + '\n'

with open('pkg/display/display_test.go', 'w') as f:
    f.write(content)
"

    # Verify it compiles
    go build ./pkg/display/ 2>/dev/null || {
        echo "WARNING: alternatives mutation broke build"
        return 1
    }
    go test ./pkg/display/ 2>/dev/null || {
        echo "WARNING: alternatives mutation tests fail"
        return 1
    }

    git add -A
    git commit -m "mutation: add competing FormatDurationCompact and FormatTimestamp" --no-verify 2>/dev/null || true
}

# --- Agent runner (same as main run.sh) ---

run_agent() {
    local worktree="$1"
    local prompt="$2"
    local result_dir="$3"

    mkdir -p "$result_dir"

    local start_time=$(date +%s)
    echo "$start_time" > "$result_dir/start_time"

    cd "$worktree"
    timeout "${TIMEOUT_MINUTES}m" env -u CLAUDECODE BEADS_NO_DAEMON=1 claude \
        --model "$MODEL_FULL" \
        --dangerously-skip-permissions \
        -p "$prompt" \
        > "$result_dir/stdout.log" 2>"$result_dir/stderr.log" || true

    local end_time=$(date +%s)
    echo "$end_time" > "$result_dir/end_time"
    local duration=$((end_time - start_time))
    echo "$duration" > "$result_dir/duration_seconds"

    # Stage and commit unstaged agent work
    cd "$worktree"
    git add pkg/display/ 2>/dev/null || true
    git diff --cached --quiet 2>/dev/null || \
        git commit -m "agent work" --no-verify 2>/dev/null || true

    # Capture results
    cd "$worktree"
    local mutation_commit
    mutation_commit=$(git log --format='%H' --max-count=1 HEAD~1 2>/dev/null || echo "$BASELINE_COMMIT")
    git diff --stat "${mutation_commit}"..HEAD -- ':!.beads/' > "$result_dir/diff_stat.txt" 2>/dev/null || true
    git diff "${mutation_commit}"..HEAD -- ':!.beads/' > "$result_dir/full_diff.txt" 2>/dev/null || true
    git log --oneline "${mutation_commit}"..HEAD > "$result_dir/commits.txt" 2>/dev/null || true
    git status --short > "$result_dir/git_status.txt" 2>/dev/null || true

    go test ./pkg/display/ -v > "$result_dir/all_tests.txt" 2>&1 || true
    go build ./... > "$result_dir/build_output.txt" 2>&1 || true

    cp "$worktree/pkg/display/display.go" "$result_dir/display.go" 2>/dev/null || true
    cp "$worktree/pkg/display/display_test.go" "$result_dir/display_test.go" 2>/dev/null || true
    cp "$worktree/pkg/display/ansi.go" "$result_dir/ansi.go" 2>/dev/null || true
}

# --- Merge checker ---

check_merge() {
    local trial_dir="$1"
    local branch_a="$2"
    local branch_b="$3"

    local merge_wt="/tmp/decay-merge-$$"
    local merge_branch="decay-merge-test-$$"

    cd "$PROJECT_DIR"
    git worktree add -b "$merge_branch" "$merge_wt" "$branch_a" 2>/dev/null || {
        echo "no-merge,0,failed to create merge worktree" > "$trial_dir/merge_result.csv"
        return
    }

    cd "$merge_wt"
    local merge_out
    merge_out=$(git merge "$branch_b" --no-edit 2>&1) || true

    if echo "$merge_out" | grep -q "CONFLICT"; then
        local cf=$(echo "$merge_out" | grep -c "CONFLICT")
        echo "conflict,$cf,merge conflict" > "$trial_dir/merge_result.csv"
        echo "  Merge: CONFLICT ($cf files)"
        git merge --abort 2>/dev/null || true
    elif echo "$merge_out" | grep -q "Already up to date"; then
        echo "no_change,0,no changes" > "$trial_dir/merge_result.csv"
        echo "  Merge: NO CHANGE"
    else
        local build_ok=true
        go build ./... > "$trial_dir/merge_build.txt" 2>&1 || build_ok=false

        if [ "$build_ok" = false ]; then
            echo "build_fail,0,merged but build fails" > "$trial_dir/merge_result.csv"
            echo "  Merge: BUILD FAIL"
        else
            local test_out
            test_out=$(go test ./pkg/display/ -v 2>&1) || true
            if echo "$test_out" | grep -q "^ok"; then
                echo "success,0,clean merge + tests pass" > "$trial_dir/merge_result.csv"
                echo "  Merge: SUCCESS"
            else
                echo "semantic_conflict,0,merged but tests fail" > "$trial_dir/merge_result.csv"
                echo "  Merge: SEMANTIC CONFLICT"
            fi
            echo "$test_out" > "$trial_dir/merge_tests.txt"
        fi
    fi

    cd "$PROJECT_DIR"
    git worktree remove "$merge_wt" --force 2>/dev/null || true
    git branch -D "$merge_branch" 2>/dev/null || true
}

# --- Placement prompt builder (uses ORIGINAL prompts with stale attractors) ---

build_placement_prompt() {
    local role="$1"  # a or b

    local base_prompt
    base_prompt=$(cat "$PROMPTS_DIR/simple-${role}.md")

    local placement_note
    if [ "$role" = "a" ]; then
        # ORIGINAL attractor: references FormatDurationShort (may not exist after mutation)
        placement_note="

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`FormatDurationShort\` function in display.go.
Place your new test function(s) IMMEDIATELY after \`TestFormatDurationShort\` in display_test.go.

Do NOT place code anywhere else in these files."
    else
        # ORIGINAL attractor: references StripANSI in display.go (may be moved after mutation)
        placement_note="

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`StripANSI\` function in display.go (BEFORE \`FormatDuration\`).
Place your new test function(s) IMMEDIATELY after \`TestStripANSI\` in display_test.go (BEFORE \`TestFormatDuration\`).

Do NOT place code anywhere else in these files."
    fi

    echo "${base_prompt}${placement_note}"
}

# --- Main loop ---

total_trials=$((${#PHASES[@]} * TRIALS))
current=0

for phase in "${PHASES[@]}"; do
    echo ""
    echo "================================================================"
    echo "=== Phase: $phase (N=$TRIALS) ==="
    echo "================================================================"

    phase_dir="$RESULTS_DIR/$phase/simple"
    mkdir -p "$phase_dir"

    # Create a mutation branch from baseline
    mutation_branch="decay-mutation-${phase}-$$"
    mutation_wt="/tmp/decay-mutation-${phase}-$$"

    cd "$PROJECT_DIR"
    git worktree add -b "$mutation_branch" "$mutation_wt" "$BASELINE_COMMIT" 2>/dev/null

    # Apply the phase mutation
    echo "Applying mutation: $phase"
    case "$phase" in
        rename)       apply_rename_mutation "$mutation_wt" ;;
        reorganize)   apply_reorganize_mutation "$mutation_wt" ;;
        alternatives) apply_alternatives_mutation "$mutation_wt" ;;
    esac

    mutation_commit=$(cd "$mutation_wt" && git rev-parse HEAD)
    echo "Mutation commit: $mutation_commit"

    # Capture the mutated display.go for reference
    cp "$mutation_wt/pkg/display/display.go" "$RESULTS_DIR/${phase}_display.go" 2>/dev/null || true
    cp "$mutation_wt/pkg/display/display_test.go" "$RESULTS_DIR/${phase}_display_test.go" 2>/dev/null || true
    [ -f "$mutation_wt/pkg/display/ansi.go" ] && cp "$mutation_wt/pkg/display/ansi.go" "$RESULTS_DIR/${phase}_ansi.go" 2>/dev/null || true

    # Clean up mutation worktree (keep the branch)
    cd "$PROJECT_DIR"
    git worktree remove "$mutation_wt" --force 2>/dev/null || true

    for trial in $(seq 1 "$TRIALS"); do
        current=$((current + 1))
        echo ""
        echo "--- [$current/$total_trials] $phase trial $trial ---"

        trial_dir="$phase_dir/trial-$trial"
        mkdir -p "$trial_dir/agent-a" "$trial_dir/agent-b"

        # Create worktrees FROM the mutation commit (not baseline)
        wt_a="/tmp/decay-${phase}-a-t${trial}-$$"
        wt_b="/tmp/decay-${phase}-b-t${trial}-$$"
        branch_a="exp/decay-${phase}-a-t${trial}-$$"
        branch_b="exp/decay-${phase}-b-t${trial}-$$"

        cd "$PROJECT_DIR"
        git worktree add -b "$branch_a" "$wt_a" "$mutation_commit" 2>/dev/null
        git worktree add -b "$branch_b" "$wt_b" "$mutation_commit" 2>/dev/null

        # Build prompts (ORIGINAL stale attractors)
        prompt_a=$(build_placement_prompt "a")
        prompt_b=$(build_placement_prompt "b")

        echo "$prompt_a" > "$trial_dir/agent-a/prompt.md"
        echo "$prompt_b" > "$trial_dir/agent-b/prompt.md"

        # Run both agents in parallel
        (run_agent "$wt_a" "$prompt_a" "$trial_dir/agent-a") &
        pid_a=$!
        (run_agent "$wt_b" "$prompt_b" "$trial_dir/agent-b") &
        pid_b=$!

        wait "$pid_a" || true
        wait "$pid_b" || true

        dur_a=$(cat "$trial_dir/agent-a/duration_seconds" 2>/dev/null || echo "?")
        dur_b=$(cat "$trial_dir/agent-b/duration_seconds" 2>/dev/null || echo "?")
        echo "  Agent A: ${dur_a}s | Agent B: ${dur_b}s"

        # Check merge
        check_merge "$trial_dir" "$branch_a" "$branch_b"

        # Cleanup
        cd "$PROJECT_DIR"
        git worktree remove "$wt_a" --force 2>/dev/null || true
        git worktree remove "$wt_b" --force 2>/dev/null || true
        git branch -D "$branch_a" 2>/dev/null || true
        git branch -D "$branch_b" 2>/dev/null || true
    done

    # Cleanup mutation branch
    cd "$PROJECT_DIR"
    git branch -D "$mutation_branch" 2>/dev/null || true
done

# --- Scoring ---

echo ""
echo "=== Scoring ==="

SCORE_FILE="$RESULTS_DIR/scores.csv"
echo "phase,trial,agent,completion,build,tests_pass,spec_match,duration_s,total" > "$SCORE_FILE"

for phase in "${PHASES[@]}"; do
    for trial_dir in "$RESULTS_DIR/$phase/simple/trial-"*/; do
        [ ! -d "$trial_dir" ] && continue
        trial=$(basename "$trial_dir" | sed 's/trial-//')

        for agent in a b; do
            agent_dir="$trial_dir/agent-$agent"
            [ ! -d "$agent_dir" ] && continue

            f_completion=0
            f_build=0
            f_tests=0
            f_spec=0
            duration=0

            # Completion
            if [ -f "$agent_dir/full_diff.txt" ] && [ -s "$agent_dir/full_diff.txt" ]; then
                f_completion=1
            fi

            # Build
            if [ -f "$agent_dir/build_output.txt" ] && [ ! -s "$agent_dir/build_output.txt" ]; then
                f_build=1
            fi

            # Tests
            if [ -f "$agent_dir/all_tests.txt" ]; then
                if grep -q "^ok" "$agent_dir/all_tests.txt" 2>/dev/null; then
                    f_tests=1
                fi
            fi

            # Spec match
            if [ -f "$agent_dir/display.go" ]; then
                if [ "$agent" = "a" ]; then
                    grep -q 'func FormatBytes' "$agent_dir/display.go" 2>/dev/null && f_spec=1
                else
                    grep -q 'func FormatRate' "$agent_dir/display.go" 2>/dev/null && f_spec=1
                fi
            fi

            # Duration
            if [ -f "$agent_dir/duration_seconds" ]; then
                duration=$(cat "$agent_dir/duration_seconds")
            fi

            total=$((f_completion + f_build + f_tests + f_spec))
            echo "$phase,$trial,$agent,$f_completion,$f_build,$f_tests,$f_spec,$duration,$total" >> "$SCORE_FILE"
            printf "  %-14s t%-2s agent-%s: comp=%d build=%d test=%d spec=%d  %d/4  (%ds)\n" \
                "$phase" "$trial" "$agent" \
                "$f_completion" "$f_build" "$f_tests" "$f_spec" "$total" "$duration"
        done
    done
done

# --- Merge summary ---

echo ""
echo "=== Merge Results by Phase ==="
echo ""

MERGE_FILE="$RESULTS_DIR/merge_summary.csv"
echo "phase,trial,result,conflict_files,detail" > "$MERGE_FILE"

printf "%-14s %6s %9s %7s %12s %9s\n" "Phase" "Trials" "Conflicts" "Success" "Build Fail" "Semantic"

for phase in "${PHASES[@]}"; do
    conflicts=0
    success=0
    build_fail=0
    semantic=0
    total=0

    for trial_dir in "$RESULTS_DIR/$phase/simple/trial-"*/; do
        [ ! -d "$trial_dir" ] && continue
        total=$((total + 1))
        trial=$(basename "$trial_dir" | sed 's/trial-//')

        if [ -f "$trial_dir/merge_result.csv" ]; then
            result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
            cf=$(cut -d',' -f2 "$trial_dir/merge_result.csv")
            detail=$(cut -d',' -f3 "$trial_dir/merge_result.csv")
            echo "$phase,$trial,$result,$cf,$detail" >> "$MERGE_FILE"

            case "$result" in
                conflict) conflicts=$((conflicts + 1)) ;;
                success) success=$((success + 1)) ;;
                build_fail) build_fail=$((build_fail + 1)) ;;
                semantic_conflict) semantic=$((semantic + 1)) ;;
            esac
        fi
    done

    printf "%-14s %6d %9d %7d %12d %9d\n" "$phase" "$total" "$conflicts" "$success" "$build_fail" "$semantic"
done

# --- Degradation curve ---

echo ""
echo "=== Degradation Curve ==="
echo ""
echo "Phase          Success Rate   (vs 100% baseline)"
echo "baseline       100%           (20/20 from prior experiment)"

for phase in "${PHASES[@]}"; do
    total=0
    success=0
    for trial_dir in "$RESULTS_DIR/$phase/simple/trial-"*/; do
        [ ! -d "$trial_dir" ] && continue
        total=$((total + 1))
        if [ -f "$trial_dir/merge_result.csv" ]; then
            result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
            [ "$result" = "success" ] && success=$((success + 1))
        fi
    done
    if [ "$total" -gt 0 ]; then
        rate=$((success * 100 / total))
        printf "%-14s %3d%%           (%d/%d)\n" "$phase" "$rate" "$success" "$total"
    fi
done

echo ""
echo "=== Complete ==="
echo "Results: $RESULTS_DIR"
