#!/usr/bin/env bash
# Experiment C: Anticipatory Placement
#
# Tests whether an LLM can predict non-overlapping insertion points
# via static analysis of the codebase + task descriptions, without
# needing any failure data.
#
# Design:
#   1. Give claude-haiku the codebase + both task descriptions
#   2. Ask it to generate non-overlapping insertion points
#   3. Use those placements in agent prompts
#   4. Run agents, check merges
#   5. Compare against human-designed placement baseline (20/20 success)
#
# Usage:
#   ./run-anticipatory.sh                    # Run all (10 simple + 10 complex)
#   ./run-anticipatory.sh --trials 3         # Fewer trials
#   ./run-anticipatory.sh --task simple      # Single task type

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/prompts"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_BASE="$SCRIPT_DIR/results"

# Defaults
TRIALS=10
TASK_TYPES=("simple" "complex")
AGENT_MODEL="haiku"
AGENT_MODEL_FULL="claude-haiku-4-5-20251001"
PLACEMENT_MODEL="haiku"
PLACEMENT_MODEL_FULL="claude-haiku-4-5-20251001"
TIMEOUT_MINUTES=10

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --trials) TRIALS="$2"; shift 2 ;;
        --task) TASK_TYPES=("$2"); shift 2 ;;
        --agent-model) AGENT_MODEL="$2"; shift 2 ;;
        --placement-model) PLACEMENT_MODEL="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

# Resolve models
resolve_model() {
    case "$1" in
        haiku) echo "claude-haiku-4-5-20251001" ;;
        opus)  echo "claude-opus-4-5-20251101" ;;
        sonnet) echo "claude-sonnet-4-5-20250514" ;;
        *) echo "$1" ;;
    esac
}
AGENT_MODEL_FULL=$(resolve_model "$AGENT_MODEL")
PLACEMENT_MODEL_FULL=$(resolve_model "$PLACEMENT_MODEL")

RESULTS_DIR="$RESULTS_BASE/$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

echo "=== Experiment C: Anticipatory Placement ==="
echo "Project:          $PROJECT_DIR"
echo "Baseline:         $BASELINE_COMMIT"
echo "Agent model:      $AGENT_MODEL ($AGENT_MODEL_FULL)"
echo "Placement model:  $PLACEMENT_MODEL ($PLACEMENT_MODEL_FULL)"
echo "Trials:           $TRIALS per task type"
echo "Tasks:            ${TASK_TYPES[*]}"
echo "Results:          $RESULTS_DIR"
echo ""

# Save metadata
cat > "$RESULTS_DIR/metadata.json" << EOF
{
  "experiment": "anticipatory-placement",
  "timestamp": "$TIMESTAMP",
  "baseline_commit": "$BASELINE_COMMIT",
  "agent_model": "$AGENT_MODEL",
  "agent_model_full": "$AGENT_MODEL_FULL",
  "placement_model": "$PLACEMENT_MODEL",
  "placement_model_full": "$PLACEMENT_MODEL_FULL",
  "trials": $TRIALS,
  "task_types": $(printf '%s\n' "${TASK_TYPES[@]}" | jq -R . | jq -s .),
  "condition": "anticipatory",
  "timeout_minutes": $TIMEOUT_MINUTES
}
EOF

# --- Read codebase files ---
DISPLAY_GO=$(cat "$PROJECT_DIR/pkg/display/display.go")
DISPLAY_TEST_GO=$(cat "$PROJECT_DIR/pkg/display/display_test.go")

# --- Generate placement via LLM ---
generate_placement() {
    local task_type="$1"
    local trial_num="$2"
    local output_dir="$3"

    local task_a
    local task_b
    task_a=$(cat "$PROMPTS_DIR/${task_type}-a.md")
    task_b=$(cat "$PROMPTS_DIR/${task_type}-b.md")

    local placement_prompt="You are a code placement coordinator. Two parallel agents will each implement one task, modifying the same files simultaneously. Their changes must merge cleanly via git.

## Target Codebase

### pkg/display/display.go
\`\`\`go
${DISPLAY_GO}
\`\`\`

### pkg/display/display_test.go
\`\`\`go
${DISPLAY_TEST_GO}
\`\`\`

## Task for Agent A
${task_a}

## Task for Agent B
${task_b}

## Your Job

Analyze the codebase structure and both tasks. Assign specific, non-overlapping insertion points for each agent so their code changes will merge cleanly.

For each agent, specify:
1. Where in display.go to place their new function(s) — reference an existing function name
2. Where in display_test.go to place their new test(s) — reference an existing test function name

CRITICAL RULES:
- Agent A and Agent B MUST use DIFFERENT insertion points
- Reference only functions that exist in the current code
- Each placement must be \"immediately after\" a specific existing function

Output EXACTLY in this format (no other text):
AGENT_A_CODE_AFTER: <function name>
AGENT_A_TEST_AFTER: <test function name>
AGENT_B_CODE_AFTER: <function name>
AGENT_B_TEST_AFTER: <test function name>"

    echo "$placement_prompt" > "$output_dir/placement_prompt.md"

    # Call LLM for placement
    local llm_output
    llm_output=$(cd "$PROJECT_DIR" && timeout 120 env -u CLAUDECODE BEADS_NO_DAEMON=1 claude \
        --model "$PLACEMENT_MODEL_FULL" \
        --dangerously-skip-permissions \
        -p "$placement_prompt" 2>"$output_dir/placement_stderr.log") || {
        echo "PLACEMENT_FAILED"
        return 1
    }

    echo "$llm_output" > "$output_dir/placement_output.txt"

    # Parse output
    local a_code a_test b_code b_test
    a_code=$(echo "$llm_output" | grep -i "AGENT_A_CODE_AFTER:" | head -1 | sed 's/.*AGENT_A_CODE_AFTER:\s*//' | xargs)
    a_test=$(echo "$llm_output" | grep -i "AGENT_A_TEST_AFTER:" | head -1 | sed 's/.*AGENT_A_TEST_AFTER:\s*//' | xargs)
    b_code=$(echo "$llm_output" | grep -i "AGENT_B_CODE_AFTER:" | head -1 | sed 's/.*AGENT_B_CODE_AFTER:\s*//' | xargs)
    b_test=$(echo "$llm_output" | grep -i "AGENT_B_TEST_AFTER:" | head -1 | sed 's/.*AGENT_B_TEST_AFTER:\s*//' | xargs)

    if [ -z "$a_code" ] || [ -z "$a_test" ] || [ -z "$b_code" ] || [ -z "$b_test" ]; then
        echo "PARSE_FAILED"
        echo "Parse failed: a_code='$a_code' a_test='$a_test' b_code='$b_code' b_test='$b_test'" > "$output_dir/placement_parse_error.txt"
        return 1
    fi

    # Check if placements are actually different
    local placements_different="true"
    if [ "$a_code" = "$b_code" ]; then
        placements_different="false"
    fi

    # Save parsed placements
    cat > "$output_dir/placement_parsed.json" << PEOF
{
  "agent_a_code_after": "$a_code",
  "agent_a_test_after": "$a_test",
  "agent_b_code_after": "$b_code",
  "agent_b_test_after": "$b_test",
  "placements_different": $placements_different
}
PEOF

    echo "$a_code|$a_test|$b_code|$b_test|$placements_different"
}

# --- Build prompt with LLM-generated placement ---
build_anticipatory_prompt() {
    local task_type="$1"
    local role="$2"
    local code_after="$3"
    local test_after="$4"

    local base_prompt
    base_prompt=$(cat "$PROMPTS_DIR/${task_type}-${role}.md")

    local placement_note="

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`${code_after}\` function in display.go.
Place your new test function(s) IMMEDIATELY after \`${test_after}\` in display_test.go.

Do NOT place code anywhere else in these files."

    echo "${base_prompt}${placement_note}"
}

# --- Agent runner (same as run.sh) ---
run_agent() {
    local worktree="$1"
    local prompt="$2"
    local result_dir="$3"

    mkdir -p "$result_dir"

    local start_time=$(date +%s)
    echo "$start_time" > "$result_dir/start_time"

    cd "$worktree"
    timeout "${TIMEOUT_MINUTES}m" env -u CLAUDECODE BEADS_NO_DAEMON=1 claude \
        --model "$AGENT_MODEL_FULL" \
        --dangerously-skip-permissions \
        -p "$prompt" \
        > "$result_dir/stdout.log" 2>"$result_dir/stderr.log" || true

    local end_time=$(date +%s)
    echo "$end_time" > "$result_dir/end_time"
    local duration=$((end_time - start_time))
    echo "$duration" > "$result_dir/duration_seconds"

    # Stage and commit
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

    go test ./pkg/display/ -v > "$result_dir/all_tests.txt" 2>&1 || true
    go build ./pkg/display/ > "$result_dir/build_output.txt" 2>&1 || true

    cp "$worktree/pkg/display/display.go" "$result_dir/display.go" 2>/dev/null || true
    cp "$worktree/pkg/display/display_test.go" "$result_dir/display_test.go" 2>/dev/null || true
    cp "$worktree/pkg/display/table.go" "$result_dir/table.go" 2>/dev/null || true
    cp "$worktree/pkg/display/table_test.go" "$result_dir/table_test.go" 2>/dev/null || true
}

# --- Merge checker (same as run.sh) ---
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
        local build_ok=true
        go build ./pkg/display/ > "$trial_dir/merge_build.txt" 2>&1 || build_ok=false

        if [ "$build_ok" = false ]; then
            echo "build_fail,0,merged but build fails" > "$trial_dir/merge_result.csv"
            echo "  Merge: BUILD FAIL"
        else
            local test_out
            test_out=$(go test ./pkg/display/ -v 2>&1) || true
            if echo "$test_out" | grep -q "^ok"; then
                echo "success,0,clean merge + tests pass" > "$trial_dir/merge_result.csv"
                echo "  Merge: SUCCESS (clean merge, tests pass)"
            else
                echo "semantic_conflict,0,merged but tests fail" > "$trial_dir/merge_result.csv"
                echo "  Merge: SEMANTIC CONFLICT (tests fail)"
            fi
            echo "$test_out" > "$trial_dir/merge_tests.txt"
        fi
    fi

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
placement_successes=0
placement_failures=0

for task_type in "${TASK_TYPES[@]}"; do
    echo ""
    echo "===== Task: $task_type ====="

    cond_dir="$RESULTS_DIR/anticipatory/$task_type"
    mkdir -p "$cond_dir"

    for trial in $(seq 1 "$TRIALS"); do
        current=$((current + 1))
        echo ""
        echo "--- [$current/$total_trials] anticipatory/$task_type trial $trial ---"

        trial_dir="$cond_dir/trial-$trial"
        mkdir -p "$trial_dir/agent-a" "$trial_dir/agent-b" "$trial_dir/placement"

        # Step 1: Generate placement via LLM
        echo "  Generating placements via $PLACEMENT_MODEL..."
        placement_result=$(generate_placement "$task_type" "$trial" "$trial_dir/placement") || {
            echo "  PLACEMENT GENERATION FAILED — skipping trial"
            placement_failures=$((placement_failures + 1))
            echo "placement_failed,0,LLM placement generation failed" > "$trial_dir/merge_result.csv"
            continue
        }

        # Parse placement result
        IFS='|' read -r a_code a_test b_code b_test placements_diff <<< "$placement_result"
        echo "  Agent A: code after '$a_code', tests after '$a_test'"
        echo "  Agent B: code after '$b_code', tests after '$b_test'"
        echo "  Placements different: $placements_diff"

        if [ "$placements_diff" = "false" ]; then
            echo "  WARNING: LLM generated SAME insertion point for both agents!"
        fi

        # Step 2: Build prompts with LLM-generated placements
        prompt_a=$(build_anticipatory_prompt "$task_type" "a" "$a_code" "$a_test")
        prompt_b=$(build_anticipatory_prompt "$task_type" "b" "$b_code" "$b_test")

        echo "$prompt_a" > "$trial_dir/agent-a/prompt.md"
        echo "$prompt_b" > "$trial_dir/agent-b/prompt.md"

        # Step 3: Create worktrees
        wt_a="/tmp/coord-antic-a-t${trial}-$$"
        wt_b="/tmp/coord-antic-b-t${trial}-$$"
        branch_a="exp/antic-${task_type}-a-t${trial}-$$"
        branch_b="exp/antic-${task_type}-b-t${trial}-$$"

        cd "$PROJECT_DIR"
        git worktree add -b "$branch_a" "$wt_a" "$BASELINE_COMMIT" 2>/dev/null
        git worktree add -b "$branch_b" "$wt_b" "$BASELINE_COMMIT" 2>/dev/null

        # Step 4: Run both agents in parallel
        (run_agent "$wt_a" "$prompt_a" "$trial_dir/agent-a") &
        pid_a=$!
        (run_agent "$wt_b" "$prompt_b" "$trial_dir/agent-b") &
        pid_b=$!

        wait "$pid_a" || true
        wait "$pid_b" || true

        dur_a=$(cat "$trial_dir/agent-a/duration_seconds" 2>/dev/null || echo "?")
        dur_b=$(cat "$trial_dir/agent-b/duration_seconds" 2>/dev/null || echo "?")
        echo "  Agent A: ${dur_a}s | Agent B: ${dur_b}s"

        # Step 5: Check merge
        check_merge "$trial_dir" "$branch_a" "$branch_b"

        # Track placement success
        merge_outcome=$(cut -d',' -f1 "$trial_dir/merge_result.csv" 2>/dev/null || echo "unknown")
        if [ "$merge_outcome" = "success" ]; then
            placement_successes=$((placement_successes + 1))
        fi

        # Cleanup
        cd "$PROJECT_DIR"
        git worktree remove "$wt_a" --force 2>/dev/null || true
        git worktree remove "$wt_b" --force 2>/dev/null || true
        git branch -D "$branch_a" 2>/dev/null || true
        git branch -D "$branch_b" 2>/dev/null || true
    done
done

echo ""
echo "=== Scoring ==="
bash "$SCRIPT_DIR/score.sh" "$RESULTS_DIR"

echo ""
echo "=== Analysis ==="
bash "$SCRIPT_DIR/analyze.sh" "$RESULTS_DIR"

# Anticipatory-specific summary
echo ""
echo "=== Anticipatory Placement Summary ==="
echo "Total trials attempted: $current"
echo "Placement generation failures: $placement_failures"
echo "Merge successes: $placement_successes"
echo "Success rate: ${placement_successes}/${current}"

# Write summary to results
cat > "$RESULTS_DIR/anticipatory_summary.md" << EOF
# Experiment C: Anticipatory Placement Results

## Summary
- **Total trials:** $current
- **Placement generation failures:** $placement_failures
- **Merge successes:** $placement_successes / $current
- **Agent model:** $AGENT_MODEL ($AGENT_MODEL_FULL)
- **Placement model:** $PLACEMENT_MODEL ($PLACEMENT_MODEL_FULL)

## Comparison to Human-Designed Placement
- **Human baseline:** 20/20 (100%) success (2026-03-10 experiment)
- **LLM-generated:** ${placement_successes}/${current}

## Placement Details
See individual trial directories for:
- \`placement/placement_output.txt\` — raw LLM output
- \`placement/placement_parsed.json\` — parsed insertion points
- \`placement/placement_prompt.md\` — the prompt used
EOF

echo ""
echo "=== Complete ==="
echo "Results: $RESULTS_DIR"
echo "Summary: $RESULTS_DIR/anticipatory_summary.md"
