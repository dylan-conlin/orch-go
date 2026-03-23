#!/usr/bin/env bash
# Git-Merge-Educated Messaging Experiment
#
# Tests whether correcting agents' false model of git merge mechanics
# changes coordination outcomes.
#
# This is the SAME as the messaging condition from the 4-condition experiment,
# but with explicit git merge education added to the prompt.
#
# Baseline: messaging condition → 20/20 CONFLICT (agents believed
#   same-point additions merge cleanly if function names differ)
#
# Hypothesis: Correcting the false merge model will reduce conflicts.
#
# Usage:
#   ./run-merge-educated.sh              # Run N=10 per task type (20 total)
#   ./run-merge-educated.sh --trials 3   # Fewer trials for testing

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/prompts"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_BASE="$SCRIPT_DIR/results"

# Defaults
TRIALS=10
TASK_TYPES=("simple" "complex")
MODEL="haiku"
MODEL_FULL="claude-haiku-4-5-20251001"
TIMEOUT_MINUTES=10

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --trials) TRIALS="$2"; shift 2 ;;
        --task) TASK_TYPES=("$2"); shift 2 ;;
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

RESULTS_DIR="$RESULTS_BASE/merge-educated-$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

# Get baseline commit
BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

echo "=== Git-Merge-Educated Messaging Experiment ==="
echo "Project:    $PROJECT_DIR"
echo "Baseline:   $BASELINE_COMMIT"
echo "Model:      $MODEL ($MODEL_FULL)"
echo "Trials:     $TRIALS per task type"
echo "Tasks:      ${TASK_TYPES[*]}"
echo "Condition:  merge-educated (messaging + git merge education)"
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
  "task_types": $(printf '%s\n' "${TASK_TYPES[@]}" | jq -R . | jq -s .),
  "conditions": ["merge-educated"],
  "timeout_minutes": $TIMEOUT_MINUTES,
  "experiment": "git-merge-educated-messaging",
  "description": "Same as messaging condition but with explicit git merge education added to prompt",
  "baseline_comparison": "messaging condition from 20260310-174045 (20/20 CONFLICT)"
}
EOF

# --- Git merge education text ---

GIT_MERGE_EDUCATION="
### CRITICAL: How Git Merge Actually Works

**You likely have a FALSE mental model of git merge.** Most AI agents believe that if two developers add different functions at the same location in a file, git will merge them cleanly. THIS IS WRONG.

**How git merge ACTUALLY works:**
- Git merges at the TEXT level, not the semantic level
- Git does NOT understand function boundaries, names, or code structure
- When two branches both INSERT text at the SAME line position, git produces a CONFLICT — regardless of what the text contains
- Example: If you add 30 lines after line 94 and the other agent adds 40 lines after line 94, git sees two competing insertions at line 94 and marks it as CONFLICT

**Concrete example of what WILL conflict:**
- Branch A: adds FormatBytes (30 lines) after FormatDurationShort
- Branch B: adds FormatRate (30 lines) after FormatDurationShort
- Result: CONFLICT — git cannot merge these because both modified the same region

**What you MUST do to avoid conflicts:**
- Choose a DIFFERENT insertion point than the other agent — at least 3 lines apart
- For display.go: one agent should insert near the TOP of the file (after StripANSI, before FormatDuration), the other after FormatDurationShort
- For display_test.go: same principle — use DIFFERENT locations
- Do NOT both insert 'after FormatDurationShort' — this WILL conflict even though your function names are different

**After reading the other agent's plan, if you see they are inserting at the same location as you, you MUST move your code to a different location.**"

# --- Prompt builder ---

build_merge_educated_prompt() {
    local task_type="$1"  # simple or complex
    local role="$2"       # a or b
    local msg_dir="$3"    # messaging directory

    # Base prompt
    local base_prompt
    base_prompt=$(cat "$PROMPTS_DIR/${task_type}-${role}.md")

    # Other agent's prompt
    local other_role
    if [ "$role" = "a" ]; then other_role="b"; else other_role="a"; fi
    local other_prompt
    other_prompt=$(cat "$PROMPTS_DIR/${task_type}-${other_role}.md")

    local prompt="${base_prompt}

## IMPORTANT: Agent Coordination Protocol

Another agent is SIMULTANEOUSLY working on this same codebase implementing a different feature.

Their task: ${other_prompt}

${GIT_MERGE_EDUCATION}

### Coordination Mechanism

You have access to a shared coordination directory: ${msg_dir}

1. BEFORE writing any code, create your implementation plan:
   - Write to: ${msg_dir}/plan-${role}.txt
   - Include: which files you'll modify, where you'll insert code (after which function), what function names you'll add
   - IMPORTANT: State the EXACT line region you plan to use (e.g., 'inserting at lines 95-130')

2. AFTER writing your plan, check for the other agent's plan:
   - Read: ${msg_dir}/plan-${other_role}.txt
   - If it exists, review it and check for OVERLAPPING LINE REGIONS
   - If your insertion regions overlap or are adjacent (within 3 lines), you MUST change YOUR insertion point
   - Remember: different function names at the same location STILL CONFLICT in git

3. After implementing, write a summary:
   - Write to: ${msg_dir}/done-${role}.txt
   - Include: what you implemented, the EXACT line numbers of your insertion

### Goal: Your changes must merge cleanly with the other agent's changes. Two insertions at the same line position WILL NOT merge cleanly regardless of content."

    echo "$prompt"
}

# --- Agent runner (same as main experiment) ---

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

    # Run tests (target pkg/display/ specifically to avoid stale .go files in results/)
    go test ./pkg/display/ -v > "$result_dir/all_tests.txt" 2>&1 || true
    go build ./pkg/display/ > "$result_dir/build_output.txt" 2>&1 || true

    # Copy modified files
    cp "$worktree/pkg/display/display.go" "$result_dir/display.go" 2>/dev/null || true
    cp "$worktree/pkg/display/display_test.go" "$result_dir/display_test.go" 2>/dev/null || true
    cp "$worktree/pkg/display/table.go" "$result_dir/table.go" 2>/dev/null || true
    cp "$worktree/pkg/display/table_test.go" "$result_dir/table_test.go" 2>/dev/null || true
}

# --- Merge checker (same as main experiment) ---

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

total_trials=0
for task_type in "${TASK_TYPES[@]}"; do
    total_trials=$((total_trials + TRIALS))
done

current=0
for task_type in "${TASK_TYPES[@]}"; do
    echo ""
    echo "===== Condition: merge-educated | Task: $task_type ====="

    cond_dir="$RESULTS_DIR/merge-educated/$task_type"
    mkdir -p "$cond_dir"

    for trial in $(seq 1 "$TRIALS"); do
        current=$((current + 1))
        echo ""
        echo "--- [$current/$total_trials] merge-educated/$task_type trial $trial ---"

        trial_dir="$cond_dir/trial-$trial"
        mkdir -p "$trial_dir/agent-a" "$trial_dir/agent-b"

        # Create worktrees
        wt_a="/tmp/coord-mergedu-a-t${trial}-$$"
        wt_b="/tmp/coord-mergedu-b-t${trial}-$$"
        branch_a="exp/coord-mergedu-${task_type}-a-t${trial}-$$"
        branch_b="exp/coord-mergedu-${task_type}-b-t${trial}-$$"

        cd "$PROJECT_DIR"
        git worktree add -b "$branch_a" "$wt_a" "$BASELINE_COMMIT" 2>/dev/null
        git worktree add -b "$branch_b" "$wt_b" "$BASELINE_COMMIT" 2>/dev/null

        # Messaging directory
        msg_dir="/tmp/coord-msg-mergedu-${trial}-$$"
        mkdir -p "$msg_dir"

        # Build prompts
        prompt_a=$(build_merge_educated_prompt "$task_type" "a" "$msg_dir")
        prompt_b=$(build_merge_educated_prompt "$task_type" "b" "$msg_dir")

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

        # Save messaging artifacts, then cleanup
        cp -r "$msg_dir" "$trial_dir/messages" 2>/dev/null || true
        rm -rf "$msg_dir"
    done
done

echo ""
echo "=== Scoring ==="
bash "$SCRIPT_DIR/score.sh" "$RESULTS_DIR"

echo ""
echo "=== Analysis ==="
bash "$SCRIPT_DIR/analyze.sh" "$RESULTS_DIR"

echo ""
echo "=== Complete ==="
echo "Results: $RESULTS_DIR"
echo ""
echo "Compare against baseline:"
echo "  Messaging baseline (20260310-174045): 20/20 CONFLICT"
echo "  This experiment: see merge results above"
