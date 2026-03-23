#!/usr/bin/env bash
# Modification Task Coordination Experiment — 4 Conditions x N=10
#
# Tests whether modification tasks (refactoring existing functions) produce
# merge conflicts without coordination. Unlike additive tasks, modification
# tasks anchor agents to different file regions — no gravitational insertion point.
#
# Conditions:
#   1. no-coord       — Each agent gets only their task, no mention of other agent
#   2. placement      — Explicit file-region boundaries ("only modify lines X-Y")
#   3. context-share  — Each agent given the other's full task description
#   4. messaging      — Agents can exchange plans via shared files
#
# Usage:
#   ./run-modification.sh                           # Run all conditions, N=10
#   ./run-modification.sh --condition no-coord      # Run single condition
#   ./run-modification.sh --trials 3                # Run fewer trials

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/prompts"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_BASE="$SCRIPT_DIR/results"

# Defaults
TRIALS=10
CONDITIONS=("no-coord" "placement" "context-share" "messaging")
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

RESULTS_DIR="$RESULTS_BASE/modify-$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

# Get baseline commit
BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

echo "=== Modification Task Coordination Experiment ==="
echo "Project:    $PROJECT_DIR"
echo "Baseline:   $BASELINE_COMMIT"
echo "Model:      $MODEL ($MODEL_FULL)"
echo "Trials:     $TRIALS"
echo "Task:       modification (Agent A: FormatDuration, Agent B: Truncate/TruncateWithPadding)"
echo "Conditions: ${CONDITIONS[*]}"
echo "Results:    $RESULTS_DIR"
echo "Timeout:    ${TIMEOUT_MINUTES}m per agent"
echo ""

# Save experiment metadata
cat > "$RESULTS_DIR/metadata.json" << EOF
{
  "timestamp": "$TIMESTAMP",
  "baseline_commit": "$BASELINE_COMMIT",
  "model": "$MODEL",
  "model_full": "$MODEL_FULL",
  "trials": $TRIALS,
  "task_type": "modification",
  "task_description": "Agent A refactors FormatDuration (add weeks, refactor if/else). Agent B refactors Truncate+TruncateWithPadding (Unicode-aware with runes).",
  "conditions": $(printf '%s\n' "${CONDITIONS[@]}" | jq -R . | jq -s .),
  "timeout_minutes": $TIMEOUT_MINUTES,
  "hypothesis": "Modification tasks should produce 0% conflict rate across all conditions because agents are anchored to different functions (no gravitational insertion point)."
}
EOF

# --- Prompt builders ---

build_prompt() {
    local role="$1"       # a or b
    local condition="$2"  # no-coord, placement, context-share, messaging
    local trial_num="$3"
    local msg_dir="$4"    # messaging directory (only used for messaging condition)

    # Base prompt (modification tasks)
    local base_prompt
    base_prompt=$(cat "$PROMPTS_DIR/modify-${role}.md")

    # Other agent's prompt (for context-share and messaging)
    local other_role
    if [ "$role" = "a" ]; then other_role="b"; else other_role="a"; fi
    local other_prompt
    other_prompt=$(cat "$PROMPTS_DIR/modify-${other_role}.md")

    case "$condition" in
        no-coord)
            echo "$base_prompt"
            ;;

        placement)
            local placement_note
            if [ "$role" = "a" ]; then
                placement_note="

## IMPORTANT: File Region Boundaries

You may ONLY modify code in these regions of display.go:
- The \`FormatDuration\` function (currently lines 49-82)
- The import block if needed for your changes

You may ONLY modify code in these regions of display_test.go:
- The \`TestFormatDuration\` function (currently lines 87-114)
- The import block if needed

Do NOT touch any other lines in these files."
            else
                placement_note="

## IMPORTANT: File Region Boundaries

You may ONLY modify code in these regions of display.go:
- The \`Truncate\` function (currently lines 14-19)
- The \`TruncateWithPadding\` function (currently lines 23-28)
- The import block if needed for your changes

You may ONLY modify code in these regions of display_test.go:
- The \`TestTruncate\` function (currently lines 8-26)
- The \`TestTruncateWithPadding\` function (currently lines 28-47)
- The import block if needed

Do NOT touch any other lines in these files."
            fi
            echo "${base_prompt}${placement_note}"
            ;;

        context-share)
            local context_note="

## IMPORTANT: Coordination Context

Another agent is SIMULTANEOUSLY working on this same codebase. They are refactoring different functions in display.go and display_test.go.

Their full task description:
---
${other_prompt}
---

You must coordinate to avoid merge conflicts:
- Be aware of which functions the other agent will modify
- Do NOT modify any functions the other agent is working on
- Ensure your changes can be merged cleanly with theirs
- Do NOT implement their task — only implement yours"
            echo "${base_prompt}${context_note}"
            ;;

        messaging)
            local msg_note="

## IMPORTANT: Agent Coordination Protocol

Another agent is SIMULTANEOUSLY working on this same codebase refactoring different functions.

Their task: ${other_prompt}

### Coordination Mechanism

You have access to a shared coordination directory: ${msg_dir}

1. BEFORE writing any code, create your implementation plan:
   - Write to: ${msg_dir}/plan-${role}.txt
   - Include: which functions you'll modify, what changes you'll make, which lines you expect to change

2. AFTER writing your plan, check for the other agent's plan:
   - Read: ${msg_dir}/plan-${other_role}.txt
   - If it exists, review it and confirm your changes don't overlap
   - If it doesn't exist yet, proceed with your plan

3. After implementing, write a summary:
   - Write to: ${msg_dir}/done-${role}.txt
   - Include: what you modified and where

### Goal: Your changes must merge cleanly with the other agent's changes."
            echo "${base_prompt}${msg_note}"
            ;;
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

    # Capture results
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

    local merge_wt="/tmp/coord-merge-$$"
    local merge_branch="coord-merge-test-$$"

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
        wt_a="/tmp/coord-mod-${condition}-a-t${trial}-$$"
        wt_b="/tmp/coord-mod-${condition}-b-t${trial}-$$"
        branch_a="exp/mod-${condition}-a-t${trial}-$$"
        branch_b="exp/mod-${condition}-b-t${trial}-$$"

        cd "$PROJECT_DIR"
        git worktree add -b "$branch_a" "$wt_a" "$BASELINE_COMMIT" 2>/dev/null
        git worktree add -b "$branch_b" "$wt_b" "$BASELINE_COMMIT" 2>/dev/null

        # Messaging directory for condition 4
        msg_dir="/tmp/coord-mod-msg-${trial}-$$"
        if [ "$condition" = "messaging" ]; then
            mkdir -p "$msg_dir"
        fi

        # Build prompts
        prompt_a=$(build_prompt "a" "$condition" "$trial" "$msg_dir")
        prompt_b=$(build_prompt "b" "$condition" "$trial" "$msg_dir")

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

        # Cleanup messaging dir
        if [ "$condition" = "messaging" ]; then
            cp -r "$msg_dir" "$trial_dir/messages" 2>/dev/null || true
            rm -rf "$msg_dir"
        fi
    done
done

echo ""
echo "=== Scoring ==="
bash "$SCRIPT_DIR/score-modification.sh" "$RESULTS_DIR"

echo ""
echo "=== Analysis ==="
bash "$SCRIPT_DIR/analyze-modification.sh" "$RESULTS_DIR"

echo ""
echo "=== Complete ==="
echo "Results: $RESULTS_DIR"
