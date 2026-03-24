#!/usr/bin/env bash
# Decomposition Quality Experiment — 5 Conditions x N=10
#
# Tests whether improving task decomposition quality eliminates the need for
# coordination primitives. Varies two independent factors:
#   - Task description quality: bare, rich, domain-anchored
#   - File structure: flat (existing) vs sectioned (with section comments)
#
# Conditions:
#   C1: bare-flat       — Minimal task, no location hints, flat file
#   C2: rich-flat       — Full context + requirements, no location hints, flat file
#   C3: anchored-flat   — Domain-anchored tasks ("size" vs "rate"), flat file
#   C4: bare-sectioned  — Minimal task, no location hints, sectioned file
#   C5: anchored-sectioned — Domain-anchored tasks, sectioned file with sections
#
# Reference points (from prior experiments, not rerun):
#   Prior no-coord  = 100% conflict (N=20)
#   Prior placement = 0%   conflict (N=20)
#
# Usage:
#   ./run-decomposition.sh                           # Run all conditions, N=10
#   ./run-decomposition.sh --condition bare-flat      # Run single condition
#   ./run-decomposition.sh --trials 3                # Run fewer trials

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/prompts/decomp"
FIXTURES_DIR="$SCRIPT_DIR/fixtures"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_BASE="$SCRIPT_DIR/results"

# Defaults
TRIALS=10
CONDITIONS=("bare-flat" "rich-flat" "anchored-flat" "bare-sectioned" "anchored-sectioned")
MODEL="haiku"
MODEL_FULL="claude-haiku-4-5-20251001"
TIMEOUT_MINUTES=10

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --condition) CONDITIONS=("$2"); shift 2 ;;
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

RESULTS_DIR="$RESULTS_BASE/decomp-$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

# Get baseline commit
BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

echo "=== Decomposition Quality Experiment ==="
echo "Project:    $PROJECT_DIR"
echo "Baseline:   $BASELINE_COMMIT"
echo "Model:      $MODEL ($MODEL_FULL)"
echo "Trials:     $TRIALS"
echo "Conditions: ${CONDITIONS[*]}"
echo "Results:    $RESULTS_DIR"
echo "Timeout:    ${TIMEOUT_MINUTES}m per agent"
echo ""
echo "Hypothesis: Improving decomposition quality reduces conflict rate"
echo "            without coordination primitives."
echo ""

# Save experiment metadata
cat > "$RESULTS_DIR/metadata.json" << EOF
{
  "timestamp": "$TIMESTAMP",
  "baseline_commit": "$BASELINE_COMMIT",
  "model": "$MODEL",
  "model_full": "$MODEL_FULL",
  "trials": $TRIALS,
  "task_type": "additive",
  "experiment": "decomposition-quality",
  "conditions": $(printf '%s\n' "${CONDITIONS[@]}" | jq -R . | jq -s .),
  "timeout_minutes": $TIMEOUT_MINUTES,
  "hypothesis": "Decomposition quality (task description + file structure) predicts conflict rate. Best decomposition (anchored-sectioned) should approach 0% conflict without coordination.",
  "design": {
    "independent_variables": ["task_description_quality", "file_structure"],
    "task_description_levels": ["bare", "rich", "anchored"],
    "file_structure_levels": ["flat", "sectioned"],
    "dependent_variable": "merge_conflict_rate",
    "coordination": "none (zero coordination across all conditions)"
  }
}
EOF

# --- Prompt selection ---

get_prompt_file() {
    local condition="$1"
    local role="$2"

    case "$condition" in
        bare-flat|bare-sectioned)
            echo "$PROMPTS_DIR/bare-${role}.md"
            ;;
        rich-flat)
            echo "$PROMPTS_DIR/rich-${role}.md"
            ;;
        anchored-flat|anchored-sectioned)
            echo "$PROMPTS_DIR/anchored-${role}.md"
            ;;
    esac
}

uses_sectioned_file() {
    local condition="$1"
    case "$condition" in
        *-sectioned) return 0 ;;
        *) return 1 ;;
    esac
}

# --- Agent runner ---

run_agent() {
    local worktree="$1"
    local prompt="$2"
    local result_dir="$3"

    mkdir -p "$result_dir"

    local start_time=$(date +%s)
    echo "$start_time" > "$result_dir/start_time"

    # Run claude
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

    # Stage and commit any unstaged changes
    cd "$worktree"
    git add pkg/display/ 2>/dev/null || true
    git diff --cached --quiet 2>/dev/null || \
        git commit -m "agent work" --no-verify 2>/dev/null || true

    # Capture results (exclude .beads/ side-effects)
    cd "$worktree"
    git diff --stat "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/diff_stat.txt" 2>/dev/null || true
    git diff --numstat "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/numstat.txt" 2>/dev/null || true
    git diff "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/full_diff.txt" 2>/dev/null || true
    git log --oneline "$BASELINE_COMMIT"..HEAD > "$result_dir/commits.txt" 2>/dev/null || true
    git status --short > "$result_dir/git_status.txt" 2>/dev/null || true

    # Run tests
    go test ./pkg/display/ -v > "$result_dir/all_tests.txt" 2>&1 || true
    go build ./pkg/display/ > "$result_dir/build_output.txt" 2>&1 || true

    # Copy modified files
    cp "$worktree/pkg/display/display.go" "$result_dir/display.go" 2>/dev/null || true
    cp "$worktree/pkg/display/display_test.go" "$result_dir/display_test.go" 2>/dev/null || true
}

# --- Merge checker ---

check_merge() {
    local trial_dir="$1"
    local branch_a="$2"
    local branch_b="$3"

    local merge_wt="/tmp/decomp-merge-$$"
    local merge_branch="decomp-merge-test-$$"

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
        # Test post-merge
        local build_ok=true
        go build ./pkg/display/ > "$trial_dir/merge_build.txt" 2>&1 || build_ok=false

        if [ "$build_ok" = false ]; then
            echo "build_fail,0,merged but build fails" > "$trial_dir/merge_result.csv"
            echo "  Merge: BUILD FAIL (clean merge but compilation error)"
        else
            local test_out
            test_out=$(go test ./pkg/display/ -v 2>&1) || true
            if echo "$test_out" | grep -q "^ok"; then
                echo "success,0,clean merge + tests pass" > "$trial_dir/merge_result.csv"
                echo "  Merge: SUCCESS (clean merge, tests pass)"
            else
                echo "semantic_conflict,0,merged but tests fail" > "$trial_dir/merge_result.csv"
                echo "  Merge: SEMANTIC CONFLICT (merged cleanly but tests fail)"
            fi
            echo "$test_out" > "$trial_dir/merge_tests.txt"
        fi
    fi

    # Cleanup
    cd "$PROJECT_DIR"
    git worktree remove "$merge_wt" --force 2>/dev/null || true
    git branch -D "$merge_branch" 2>/dev/null || true
}

# --- Main loop ---

total_trials=$((${#CONDITIONS[@]} * TRIALS))
current=0

for condition in "${CONDITIONS[@]}"; do
    echo ""
    echo "===== Condition: $condition ====="

    cond_dir="$RESULTS_DIR/$condition"
    mkdir -p "$cond_dir"

    for trial in $(seq 1 "$TRIALS"); do
        current=$((current + 1))
        echo ""
        echo "--- [$current/$total_trials] $condition trial $trial ---"

        trial_dir="$cond_dir/trial-$trial"
        mkdir -p "$trial_dir/agent-a" "$trial_dir/agent-b"

        # Create worktrees
        wt_a="/tmp/decomp-${condition}-a-t${trial}-$$"
        wt_b="/tmp/decomp-${condition}-b-t${trial}-$$"
        branch_a="exp/decomp-${condition}-a-t${trial}-$$"
        branch_b="exp/decomp-${condition}-b-t${trial}-$$"

        cd "$PROJECT_DIR"
        git worktree add -b "$branch_a" "$wt_a" "$BASELINE_COMMIT" 2>/dev/null
        git worktree add -b "$branch_b" "$wt_b" "$BASELINE_COMMIT" 2>/dev/null

        # For sectioned conditions, replace display files with sectioned variants
        if uses_sectioned_file "$condition"; then
            cp "$FIXTURES_DIR/display_sectioned.go" "$wt_a/pkg/display/display.go"
            cp "$FIXTURES_DIR/display_test_sectioned.go" "$wt_a/pkg/display/display_test.go"
            cp "$FIXTURES_DIR/display_sectioned.go" "$wt_b/pkg/display/display.go"
            cp "$FIXTURES_DIR/display_test_sectioned.go" "$wt_b/pkg/display/display_test.go"

            # Commit the sectioned files as the new baseline for this trial
            cd "$wt_a" && git add pkg/display/ && git commit -m "setup: sectioned display files" --no-verify 2>/dev/null || true
            cd "$wt_b" && git add pkg/display/ && git commit -m "setup: sectioned display files" --no-verify 2>/dev/null || true
        fi

        # Build prompts
        prompt_file_a=$(get_prompt_file "$condition" "a")
        prompt_file_b=$(get_prompt_file "$condition" "b")
        prompt_a=$(cat "$prompt_file_a")
        prompt_b=$(cat "$prompt_file_b")

        # Save prompts for reproducibility
        echo "$prompt_a" > "$trial_dir/agent-a/prompt.md"
        echo "$prompt_b" > "$trial_dir/agent-b/prompt.md"

        # Run both agents in parallel
        (run_agent "$wt_a" "$prompt_a" "$trial_dir/agent-a") &
        pid_a=$!
        (run_agent "$wt_b" "$prompt_b" "$trial_dir/agent-b") &
        pid_b=$!

        wait "$pid_a" || true
        wait "$pid_b" || true

        # Read durations
        dur_a=$(cat "$trial_dir/agent-a/duration_seconds" 2>/dev/null || echo "?")
        dur_b=$(cat "$trial_dir/agent-b/duration_seconds" 2>/dev/null || echo "?")
        echo "  Agent A: ${dur_a}s | Agent B: ${dur_b}s"

        # Check merge
        check_merge "$trial_dir" "$branch_a" "$branch_b"

        # Cleanup worktrees
        cd "$PROJECT_DIR"
        git worktree remove "$wt_a" --force 2>/dev/null || true
        git worktree remove "$wt_b" --force 2>/dev/null || true
        git branch -D "$branch_a" 2>/dev/null || true
        git branch -D "$branch_b" 2>/dev/null || true
    done
done

echo ""
echo "=== Scoring ==="
bash "$SCRIPT_DIR/score-decomposition.sh" "$RESULTS_DIR"

echo ""
echo "=== Analysis ==="
bash "$SCRIPT_DIR/analyze-decomposition.sh" "$RESULTS_DIR"

echo ""
echo "=== Complete ==="
echo "Results: $RESULTS_DIR"
