#!/usr/bin/env bash
# Agent Scaling Experiment — N>2 with Limited Insertion Points
#
# Tests whether structural placement degrades when agents exceed
# the number of natural insertion points (3 functions in pkg/scaling).
#
# Conditions:
#   1. no-placement     — No coordination instructions
#   2. even-placement   — Agents spread across 3 regions (shared where necessary)
#   3. sub-placement    — Fine-grained sub-region instructions for shared regions
#
# Usage:
#   ./run-scaling.sh                              # Default: N=4, 5 trials, all conditions
#   ./run-scaling.sh --agents 6                   # N=6 agents
#   ./run-scaling.sh --agents 4 --trials 5        # Specify both
#   ./run-scaling.sh --condition no-placement      # Single condition
#   ./run-scaling.sh --agents 4 --condition even-placement --trials 3

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/prompts/scaling"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_BASE="$SCRIPT_DIR/results"

# Defaults
NUM_AGENTS=4
TRIALS=5
CONDITIONS=("no-placement" "even-placement" "sub-placement")
MODEL="haiku"
MODEL_FULL="claude-haiku-4-5-20251001"
TIMEOUT_MINUTES=10
TARGET_PKG="pkg/scaling"

# Agent labels
ALL_AGENTS=("a" "b" "c" "d" "e" "f")

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --agents) NUM_AGENTS="$2"; shift 2 ;;
        --trials) TRIALS="$2"; shift 2 ;;
        --condition) CONDITIONS=("$2"); shift 2 ;;
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

# Select agent labels
AGENTS=("${ALL_AGENTS[@]:0:$NUM_AGENTS}")

RESULTS_DIR="$RESULTS_BASE/scaling-n${NUM_AGENTS}-$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

echo "================================================"
echo "  AGENT SCALING EXPERIMENT (N=$NUM_AGENTS)"
echo "================================================"
echo "Project:        $PROJECT_DIR"
echo "Baseline:       $BASELINE_COMMIT"
echo "Model:          $MODEL ($MODEL_FULL)"
echo "Agents:         ${AGENTS[*]} ($NUM_AGENTS total)"
echo "Insertion pts:  3 (Normalize, Clamp, Wrap)"
echo "Ratio:          3:$NUM_AGENTS = $(echo "scale=2; 3/$NUM_AGENTS" | bc)"
echo "Trials:         $TRIALS"
echo "Conditions:     ${CONDITIONS[*]}"
echo "Results:        $RESULTS_DIR"
echo ""

# Save experiment metadata
cat > "$RESULTS_DIR/metadata.json" << EOF
{
  "experiment": "agent-scaling",
  "timestamp": "$TIMESTAMP",
  "baseline_commit": "$BASELINE_COMMIT",
  "model": "$MODEL",
  "model_full": "$MODEL_FULL",
  "num_agents": $NUM_AGENTS,
  "agents": $(printf '%s\n' "${AGENTS[@]}" | jq -R . | jq -s .),
  "insertion_points": 3,
  "ip_to_agent_ratio": $(echo "scale=2; 3/$NUM_AGENTS" | bc),
  "trials": $TRIALS,
  "conditions": $(printf '%s\n' "${CONDITIONS[@]}" | jq -R . | jq -s .),
  "timeout_minutes": $TIMEOUT_MINUTES,
  "target_package": "$TARGET_PKG"
}
EOF

# --- Placement constraint builders ---

# Expected function names per agent (for scoring)
declare -A AGENT_FUNCTIONS
AGENT_FUNCTIONS[a]="Capitalize"
AGENT_FUNCTIONS[b]="PadLeft"
AGENT_FUNCTIONS[c]="Abs"
AGENT_FUNCTIONS[d]="RoundTo"
AGENT_FUNCTIONS[e]="Repeat"
AGENT_FUNCTIONS[f]="Lerp"

# even-placement: spread agents across 3 regions, share where necessary
# N=4: Region1(Normalize): a  Region2(Clamp): c,d  Region3(Wrap): b
# N=6: Region1(Normalize): a,e  Region2(Clamp): c,d  Region3(Wrap): b,f
build_even_placement() {
    local agent="$1"
    local num="$2"

    case "$num" in
        4)
            case "$agent" in
                a) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Normalize\` function in scaling.go (BEFORE \`Clamp\`).
Place your new test function(s) IMMEDIATELY after \`TestNormalize\` in scaling_test.go (BEFORE \`TestClamp\`).

Do NOT place code anywhere else in these files." ;;
                b) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Wrap\` function in scaling.go (at the end of the file).
Place your new test function(s) IMMEDIATELY after \`TestWrap\` in scaling_test.go (at the end of the file).

Do NOT place code anywhere else in these files." ;;
                c) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Clamp\` function in scaling.go (BEFORE \`Wrap\`).
Place your new test function(s) IMMEDIATELY after \`TestClamp\` in scaling_test.go (BEFORE \`TestWrap\`).

Do NOT place code anywhere else in these files." ;;
                d) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Clamp\` function in scaling.go (BEFORE \`Wrap\`).
Place your new test function(s) IMMEDIATELY after \`TestClamp\` in scaling_test.go (BEFORE \`TestWrap\`).

Do NOT place code anywhere else in these files." ;;
            esac
            ;;
        6)
            case "$agent" in
                a) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Normalize\` function in scaling.go (BEFORE \`Clamp\`).
Place your new test function(s) IMMEDIATELY after \`TestNormalize\` in scaling_test.go (BEFORE \`TestClamp\`).

Do NOT place code anywhere else in these files." ;;
                b) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Wrap\` function in scaling.go (at the end of the file).
Place your new test function(s) IMMEDIATELY after \`TestWrap\` in scaling_test.go (at the end of the file).

Do NOT place code anywhere else in these files." ;;
                c) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Clamp\` function in scaling.go (BEFORE \`Wrap\`).
Place your new test function(s) IMMEDIATELY after \`TestClamp\` in scaling_test.go (BEFORE \`TestWrap\`).

Do NOT place code anywhere else in these files." ;;
                d) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Clamp\` function in scaling.go (BEFORE \`Wrap\`).
Place your new test function(s) IMMEDIATELY after \`TestClamp\` in scaling_test.go (BEFORE \`TestWrap\`).

Do NOT place code anywhere else in these files." ;;
                e) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Normalize\` function in scaling.go (BEFORE \`Clamp\`).
Place your new test function(s) IMMEDIATELY after \`TestNormalize\` in scaling_test.go (BEFORE \`TestClamp\`).

Do NOT place code anywhere else in these files." ;;
                f) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Wrap\` function in scaling.go (at the end of the file).
Place your new test function(s) IMMEDIATELY after \`TestWrap\` in scaling_test.go (at the end of the file).

Do NOT place code anywhere else in these files." ;;
            esac
            ;;
    esac
}

# sub-placement: fine-grained sub-region instructions for shared regions
# Agents sharing a region get "place FIRST in region" vs "place LAST in region"
build_sub_placement() {
    local agent="$1"
    local num="$2"

    case "$num" in
        4)
            case "$agent" in
                a) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Normalize\` function in scaling.go (BEFORE \`Clamp\`).
Place your new test function(s) IMMEDIATELY after \`TestNormalize\` in scaling_test.go (BEFORE \`TestClamp\`).

Do NOT place code anywhere else in these files." ;;
                b) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Wrap\` function in scaling.go (at the end of the file).
Place your new test function(s) IMMEDIATELY after \`TestWrap\` in scaling_test.go (at the end of the file).

Do NOT place code anywhere else in these files." ;;
                c) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Clamp\` function in scaling.go.
Your function should be the FIRST new function after \`Clamp\`, placed DIRECTLY after its closing brace.
Place your new test function(s) IMMEDIATELY after \`TestClamp\` in scaling_test.go, DIRECTLY after its closing brace.

Do NOT place code anywhere else in these files. Another agent is also adding code in this region — you MUST be first (closest to Clamp)." ;;
                d) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) in scaling.go in the region between \`Clamp\` and \`Wrap\`, but IMMEDIATELY BEFORE the \`Wrap\` function.
Your function should be the LAST new function before \`Wrap\`.
Place your new test function(s) in scaling_test.go IMMEDIATELY BEFORE \`TestWrap\`.

Do NOT place code anywhere else in these files. Another agent is also adding code in this region — you MUST be last (closest to Wrap)." ;;
            esac
            ;;
        6)
            case "$agent" in
                a) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Normalize\` function in scaling.go.
Your function should be the FIRST new function after \`Normalize\`, placed DIRECTLY after its closing brace.
Place your new test function(s) IMMEDIATELY after \`TestNormalize\` in scaling_test.go, DIRECTLY after its closing brace.

Do NOT place code anywhere else in these files. Another agent is also adding code in this region — you MUST be first (closest to Normalize)." ;;
                b) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) at the VERY END of scaling.go, after ALL other functions including \`Wrap\`.
Your function should be the LAST function in the file.
Place your new test function(s) at the VERY END of scaling_test.go, after ALL other test functions.

Do NOT place code anywhere else in these files. Another agent is also adding code after Wrap — you MUST be last (at the very end of the file)." ;;
                c) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Clamp\` function in scaling.go.
Your function should be the FIRST new function after \`Clamp\`, placed DIRECTLY after its closing brace.
Place your new test function(s) IMMEDIATELY after \`TestClamp\` in scaling_test.go, DIRECTLY after its closing brace.

Do NOT place code anywhere else in these files. Another agent is also adding code in this region — you MUST be first (closest to Clamp)." ;;
                d) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) in scaling.go in the region between \`Clamp\` and \`Wrap\`, but IMMEDIATELY BEFORE the \`Wrap\` function.
Your function should be the LAST new function before \`Wrap\`.
Place your new test function(s) in scaling_test.go IMMEDIATELY BEFORE \`TestWrap\`.

Do NOT place code anywhere else in these files. Another agent is also adding code in this region — you MUST be last (closest to Wrap)." ;;
                e) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) in scaling.go in the region between \`Normalize\` and \`Clamp\`, but IMMEDIATELY BEFORE the \`Clamp\` function.
Your function should be the LAST new function before \`Clamp\`.
Place your new test function(s) in scaling_test.go IMMEDIATELY BEFORE \`TestClamp\`.

Do NOT place code anywhere else in these files. Another agent is also adding code in this region — you MUST be last (closest to Clamp)." ;;
                f) echo "

## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`Wrap\` function in scaling.go.
Your function should be the FIRST new function after \`Wrap\`, placed DIRECTLY after its closing brace.
Place your new test function(s) IMMEDIATELY after \`TestWrap\` in scaling_test.go, DIRECTLY after its closing brace.

Do NOT place code anywhere else in these files. Another agent is also adding code after Wrap — you MUST be first (closest to Wrap)." ;;
            esac
            ;;
    esac
}

# --- Prompt assembly ---

build_prompt() {
    local agent="$1"
    local condition="$2"

    local base_prompt
    base_prompt=$(cat "$PROMPTS_DIR/agent-${agent}.md")

    case "$condition" in
        no-placement)
            echo "$base_prompt"
            ;;
        even-placement)
            local placement
            placement=$(build_even_placement "$agent" "$NUM_AGENTS")
            echo "${base_prompt}${placement}"
            ;;
        sub-placement)
            local placement
            placement=$(build_sub_placement "$agent" "$NUM_AGENTS")
            echo "${base_prompt}${placement}"
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

    cd "$worktree"
    timeout "${TIMEOUT_MINUTES}m" env -u CLAUDECODE BEADS_NO_DAEMON=1 claude \
        --model "$MODEL_FULL" \
        --dangerously-skip-permissions \
        -p "$prompt" \
        > "$result_dir/stdout.log" 2>"$result_dir/stderr.log" || true

    local end_time=$(date +%s)
    echo "$end_time" > "$result_dir/end_time"
    echo "$((end_time - start_time))" > "$result_dir/duration_seconds"

    cd "$worktree"
    git add "$TARGET_PKG/" 2>/dev/null || true
    git diff --cached --quiet 2>/dev/null || \
        git commit -m "agent work" --no-verify 2>/dev/null || true

    cd "$worktree"
    git diff --stat "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/diff_stat.txt" 2>/dev/null || true
    git diff --numstat "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/numstat.txt" 2>/dev/null || true
    git diff "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/full_diff.txt" 2>/dev/null || true
    git log --oneline "$BASELINE_COMMIT"..HEAD > "$result_dir/commits.txt" 2>/dev/null || true

    go test ./"$TARGET_PKG"/ -v > "$result_dir/all_tests.txt" 2>&1 || true
    go build ./"$TARGET_PKG"/ > "$result_dir/build_output.txt" 2>&1 || true

    cp "$worktree/$TARGET_PKG/scaling.go" "$result_dir/scaling.go" 2>/dev/null || true
    cp "$worktree/$TARGET_PKG/scaling_test.go" "$result_dir/scaling_test.go" 2>/dev/null || true
}

# --- Pairwise merge checker ---

check_pairwise_merge() {
    local trial_dir="$1"
    local agent_i="$2"
    local agent_j="$3"
    local branch_i="$4"
    local branch_j="$5"

    local merge_wt="/tmp/scaling-merge-${agent_i}${agent_j}-$$-$(date +%s)"
    local merge_branch="scaling-merge-${agent_i}${agent_j}-$$-$(date +%s)"

    cd "$PROJECT_DIR"
    git worktree add -b "$merge_branch" "$merge_wt" "$branch_i" 2>/dev/null || {
        echo "no-merge" > "$trial_dir/merge_${agent_i}_${agent_j}.txt"
        return
    }

    cd "$merge_wt"
    local merge_out
    merge_out=$(git merge "$branch_j" --no-edit 2>&1) || true

    if echo "$merge_out" | grep -q "CONFLICT"; then
        echo "conflict" > "$trial_dir/merge_${agent_i}_${agent_j}.txt"
        echo "$merge_out" > "$trial_dir/merge_${agent_i}_${agent_j}_output.txt"
        git diff > "$trial_dir/merge_${agent_i}_${agent_j}_conflict.txt" 2>/dev/null || true
        git merge --abort 2>/dev/null || true
    elif echo "$merge_out" | grep -q "Already up to date"; then
        echo "no_change" > "$trial_dir/merge_${agent_i}_${agent_j}.txt"
    else
        local build_ok=true
        go build ./"$TARGET_PKG"/ > /dev/null 2>&1 || build_ok=false

        if [ "$build_ok" = false ]; then
            echo "build_fail" > "$trial_dir/merge_${agent_i}_${agent_j}.txt"
        else
            local test_out
            test_out=$(go test ./"$TARGET_PKG"/ -v 2>&1) || true
            if echo "$test_out" | grep -q "^ok"; then
                echo "success" > "$trial_dir/merge_${agent_i}_${agent_j}.txt"
            else
                echo "semantic_conflict" > "$trial_dir/merge_${agent_i}_${agent_j}.txt"
            fi
        fi
    fi

    cd "$PROJECT_DIR"
    git worktree remove "$merge_wt" --force 2>/dev/null || true
    git branch -D "$merge_branch" 2>/dev/null || true
}

# --- Sequential N-way merge ---

check_nway_merge() {
    local trial_dir="$1"
    shift
    # Remaining args are branch names in order
    local branches=("$@")

    local merge_wt="/tmp/scaling-nway-$$-$(date +%s)"
    local merge_branch="scaling-nway-$$-$(date +%s)"

    # Start from first branch
    cd "$PROJECT_DIR"
    git worktree add -b "$merge_branch" "$merge_wt" "${branches[0]}" 2>/dev/null || {
        echo "no-merge,0" > "$trial_dir/nway_merge_result.csv"
        return
    }

    cd "$merge_wt"
    local total_merged=0
    local first_conflict=""

    for ((i=1; i<${#branches[@]}; i++)); do
        local merge_out
        merge_out=$(git merge "${branches[$i]}" --no-edit 2>&1) || true

        if echo "$merge_out" | grep -q "CONFLICT"; then
            first_conflict="agent-${AGENTS[$i]}"
            echo "conflict,$total_merged,$first_conflict" > "$trial_dir/nway_merge_result.csv"
            echo "$merge_out" > "$trial_dir/nway_merge_output.txt"
            git merge --abort 2>/dev/null || true
            break
        elif echo "$merge_out" | grep -q "Already up to date"; then
            # Count as merged (no new changes)
            total_merged=$((total_merged + 1))
        else
            total_merged=$((total_merged + 1))
        fi
    done

    if [ -z "$first_conflict" ]; then
        # All merged — check build and tests
        local build_ok=true
        go build ./"$TARGET_PKG"/ > "$trial_dir/nway_build.txt" 2>&1 || build_ok=false

        if [ "$build_ok" = false ]; then
            echo "build_fail,$total_merged,build" > "$trial_dir/nway_merge_result.csv"
        else
            local test_out
            test_out=$(go test ./"$TARGET_PKG"/ -v 2>&1) || true
            if echo "$test_out" | grep -q "^ok"; then
                echo "success,$total_merged,all" > "$trial_dir/nway_merge_result.csv"
            else
                echo "semantic_conflict,$total_merged,tests" > "$trial_dir/nway_merge_result.csv"
            fi
            echo "$test_out" > "$trial_dir/nway_merge_tests.txt"
        fi
    fi

    cd "$PROJECT_DIR"
    git worktree remove "$merge_wt" --force 2>/dev/null || true
    git branch -D "$merge_branch" 2>/dev/null || true
}

# --- Scoring ---

score_agent() {
    local agent_dir="$1"
    local agent_label="$2"

    local f_completion=0
    local f_build=0
    local f_tests=0
    local f_spec=0

    # Completion
    if [ -f "$agent_dir/full_diff.txt" ] && [ -s "$agent_dir/full_diff.txt" ]; then
        f_completion=1
    fi

    # Build
    if [ -f "$agent_dir/build_output.txt" ] && [ ! -s "$agent_dir/build_output.txt" ]; then
        f_build=1
    fi

    # Tests
    if [ -f "$agent_dir/all_tests.txt" ] && grep -q "^ok" "$agent_dir/all_tests.txt" 2>/dev/null; then
        f_tests=1
    fi

    # Spec match
    local expected_func="${AGENT_FUNCTIONS[$agent_label]}"
    if [ -f "$agent_dir/scaling.go" ] && grep -q "func $expected_func" "$agent_dir/scaling.go" 2>/dev/null; then
        f_spec=1
    fi

    local total=$((f_completion + f_build + f_tests + f_spec))
    local duration
    duration=$(cat "$agent_dir/duration_seconds" 2>/dev/null || echo "?")
    echo "$f_completion,$f_build,$f_tests,$f_spec,$total,$duration"
}

# =========================================
#  MAIN LOOP
# =========================================

for condition in "${CONDITIONS[@]}"; do
    echo ""
    echo "================================================"
    echo "  Condition: $condition (N=$NUM_AGENTS)"
    echo "================================================"

    cond_dir="$RESULTS_DIR/$condition"
    mkdir -p "$cond_dir"

    for trial in $(seq 1 "$TRIALS"); do
        echo ""
        echo "--- $condition trial $trial/$TRIALS ---"

        trial_dir="$cond_dir/trial-$trial"
        mkdir -p "$trial_dir"

        # Create worktrees and branches for each agent
        declare -A wt_paths
        declare -A branch_names
        for agent in "${AGENTS[@]}"; do
            wt_paths[$agent]="/tmp/scaling-${condition}-${agent}-t${trial}-$$"
            branch_names[$agent]="exp/scaling-${condition}-${agent}-t${trial}-$$"

            cd "$PROJECT_DIR"
            git worktree add -b "${branch_names[$agent]}" "${wt_paths[$agent]}" "$BASELINE_COMMIT" 2>/dev/null
            mkdir -p "$trial_dir/agent-$agent"
        done

        # Build prompts
        for agent in "${AGENTS[@]}"; do
            local_prompt=$(build_prompt "$agent" "$condition")
            echo "$local_prompt" > "$trial_dir/agent-$agent/prompt.md"
        done

        # Run all agents in parallel
        declare -A agent_pids
        for agent in "${AGENTS[@]}"; do
            local_prompt=$(cat "$trial_dir/agent-$agent/prompt.md")
            (run_agent "${wt_paths[$agent]}" "$local_prompt" "$trial_dir/agent-$agent") &
            agent_pids[$agent]=$!
        done

        # Wait for all agents
        for agent in "${AGENTS[@]}"; do
            wait "${agent_pids[$agent]}" || true
        done

        # Report agent results
        for agent in "${AGENTS[@]}"; do
            dur=$(cat "$trial_dir/agent-$agent/duration_seconds" 2>/dev/null || echo "?")
            score_line=$(score_agent "$trial_dir/agent-$agent" "$agent")
            echo "  Agent $agent: ${dur}s | score: $score_line"
        done

        # Check all pairwise merges
        echo "  Pairwise merges:"
        for ((i=0; i<${#AGENTS[@]}; i++)); do
            for ((j=i+1; j<${#AGENTS[@]}; j++)); do
                ai="${AGENTS[$i]}"
                aj="${AGENTS[$j]}"
                check_pairwise_merge "$trial_dir" "$ai" "$aj" "${branch_names[$ai]}" "${branch_names[$aj]}"
                result=$(cat "$trial_dir/merge_${ai}_${aj}.txt" 2>/dev/null || echo "unknown")
                echo "    $ai+$aj: $result"
            done
        done

        # Check sequential N-way merge
        ordered_branches=()
        for agent in "${AGENTS[@]}"; do
            ordered_branches+=("${branch_names[$agent]}")
        done
        check_nway_merge "$trial_dir" "${ordered_branches[@]}"
        nway_result=$(cat "$trial_dir/nway_merge_result.csv" 2>/dev/null || echo "unknown")
        echo "  N-way merge: $nway_result"

        # Cleanup worktrees and branches
        cd "$PROJECT_DIR"
        for agent in "${AGENTS[@]}"; do
            git worktree remove "${wt_paths[$agent]}" --force 2>/dev/null || true
            git branch -D "${branch_names[$agent]}" 2>/dev/null || true
        done

        unset wt_paths branch_names agent_pids
    done
done

# =========================================
#  RESULTS SUMMARY
# =========================================

echo ""
echo "================================================"
echo "  RESULTS SUMMARY (N=$NUM_AGENTS)"
echo "================================================"

for condition in "${CONDITIONS[@]}"; do
    echo ""
    echo "=== $condition ==="

    cond_dir="$RESULTS_DIR/$condition"
    total_pairs=0
    pair_conflicts=0
    pair_successes=0
    nway_successes=0
    nway_total=0

    for trial_dir in "$cond_dir"/trial-*/; do
        [ ! -d "$trial_dir" ] && continue

        # Count pairwise results
        for merge_file in "$trial_dir"/merge_*_*.txt; do
            [ ! -f "$merge_file" ] && continue
            # Skip output/conflict detail files
            [[ "$merge_file" == *_output.txt ]] && continue
            [[ "$merge_file" == *_conflict.txt ]] && continue
            total_pairs=$((total_pairs + 1))
            result=$(cat "$merge_file")
            case "$result" in
                conflict) pair_conflicts=$((pair_conflicts + 1)) ;;
                success) pair_successes=$((pair_successes + 1)) ;;
            esac
        done

        # Count N-way results
        if [ -f "$trial_dir/nway_merge_result.csv" ]; then
            nway_total=$((nway_total + 1))
            nway_result=$(cut -d',' -f1 "$trial_dir/nway_merge_result.csv")
            if [ "$nway_result" = "success" ]; then
                nway_successes=$((nway_successes + 1))
            fi
        fi
    done

    echo "  Pairwise: $pair_successes/$total_pairs SUCCESS, $pair_conflicts/$total_pairs CONFLICT"
    echo "  N-way:    $nway_successes/$nway_total SUCCESS"

    # Per-pair breakdown
    echo "  Per-pair breakdown:"
    for ((i=0; i<${#AGENTS[@]}; i++)); do
        for ((j=i+1; j<${#AGENTS[@]}; j++)); do
            ai="${AGENTS[$i]}"
            aj="${AGENTS[$j]}"
            pair_s=0
            pair_c=0
            pair_t=0
            for trial_dir in "$cond_dir"/trial-*/; do
                [ ! -f "$trial_dir/merge_${ai}_${aj}.txt" ] && continue
                pair_t=$((pair_t + 1))
                r=$(cat "$trial_dir/merge_${ai}_${aj}.txt")
                case "$r" in
                    success) pair_s=$((pair_s + 1)) ;;
                    conflict) pair_c=$((pair_c + 1)) ;;
                esac
            done
            echo "    $ai+$aj: $pair_s/$pair_t success, $pair_c/$pair_t conflict"
        done
    done
done

echo ""
echo "================================================"
echo "  EXPERIMENT COMPLETE"
echo "================================================"
echo "Results: $RESULTS_DIR"
