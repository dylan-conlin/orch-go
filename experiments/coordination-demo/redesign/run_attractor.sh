#!/usr/bin/env bash
# Automated Attractor Discovery Experiment
#
# Two-phase experiment testing whether the system can learn placement
# constraints from observed merge conflicts:
#
#   Phase 1 (trials 1-5):  No attractors. Agents collide. Record patterns.
#   Phase 2 (trials 6-20): Auto-generated placement from Phase 1 collisions.
#
# Usage:
#   ./run_attractor.sh                        # Full experiment (Phase 1 + Phase 2)
#   ./run_attractor.sh --phase1-trials 3      # Fewer Phase 1 trials
#   ./run_attractor.sh --phase2-trials 10     # Fewer Phase 2 trials
#   ./run_attractor.sh --task simple          # Simple tasks only

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/prompts"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_BASE="$SCRIPT_DIR/results"

# Defaults
PHASE1_TRIALS=5
PHASE2_TRIALS=15
TASK_TYPES=("simple")   # Start with simple to minimize cost
MODEL="haiku"
MODEL_FULL="claude-haiku-4-5-20251001"
TIMEOUT_MINUTES=10

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --phase1-trials) PHASE1_TRIALS="$2"; shift 2 ;;
        --phase2-trials) PHASE2_TRIALS="$2"; shift 2 ;;
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

RESULTS_DIR="$RESULTS_BASE/attractor-$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

echo "=========================================="
echo "  AUTOMATED ATTRACTOR DISCOVERY EXPERIMENT"
echo "=========================================="
echo "Project:        $PROJECT_DIR"
echo "Baseline:       $BASELINE_COMMIT"
echo "Model:          $MODEL ($MODEL_FULL)"
echo "Phase 1 trials: $PHASE1_TRIALS (no attractors — collect collisions)"
echo "Phase 2 trials: $PHASE2_TRIALS (auto-generated attractors)"
echo "Tasks:          ${TASK_TYPES[*]}"
echo "Results:        $RESULTS_DIR"
echo ""

# Save experiment metadata
cat > "$RESULTS_DIR/metadata.json" << EOF
{
  "experiment": "automated-attractor-discovery",
  "timestamp": "$TIMESTAMP",
  "baseline_commit": "$BASELINE_COMMIT",
  "model": "$MODEL",
  "model_full": "$MODEL_FULL",
  "phase1_trials": $PHASE1_TRIALS,
  "phase2_trials": $PHASE2_TRIALS,
  "task_types": $(printf '%s\n' "${TASK_TYPES[@]}" | jq -R . | jq -s .),
  "timeout_minutes": $TIMEOUT_MINUTES
}
EOF

# --- Agent runner (reused from run.sh) ---

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
    git add pkg/display/ 2>/dev/null || true
    git diff --cached --quiet 2>/dev/null || \
        git commit -m "agent work" --no-verify 2>/dev/null || true

    cd "$worktree"
    git diff --stat "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/diff_stat.txt" 2>/dev/null || true
    git diff --numstat "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/numstat.txt" 2>/dev/null || true
    git diff "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/full_diff.txt" 2>/dev/null || true
    git log --oneline "$BASELINE_COMMIT"..HEAD > "$result_dir/commits.txt" 2>/dev/null || true

    go test ./pkg/display/ -v > "$result_dir/all_tests.txt" 2>&1 || true
    go build ./... > "$result_dir/build_output.txt" 2>&1 || true

    cp "$worktree/pkg/display/display.go" "$result_dir/display.go" 2>/dev/null || true
    cp "$worktree/pkg/display/display_test.go" "$result_dir/display_test.go" 2>/dev/null || true
}

# --- Merge checker (reused from run.sh) ---

check_merge() {
    local trial_dir="$1"
    local branch_a="$2"
    local branch_b="$3"

    local merge_wt="/tmp/coord-merge-$$-$(date +%s)"
    local merge_branch="coord-merge-test-$$-$(date +%s)"

    cd "$PROJECT_DIR"
    git worktree add -b "$merge_branch" "$merge_wt" "$branch_a" 2>/dev/null || {
        echo "no-merge,0,failed to create merge worktree" > "$trial_dir/merge_result.csv"
        return
    }

    cd "$merge_wt"
    local merge_out
    merge_out=$(git merge "$branch_b" --no-edit 2>&1) || true

    # Save raw merge output for collision analysis
    echo "$merge_out" > "$trial_dir/merge_output.txt"

    if echo "$merge_out" | grep -q "CONFLICT"; then
        local cf=$(echo "$merge_out" | grep -c "CONFLICT")
        echo "conflict,$cf,merge conflict" > "$trial_dir/merge_result.csv"
        echo "  Merge: CONFLICT ($cf files)"

        # Capture the conflict diff for analysis
        git diff > "$trial_dir/conflict_diff.txt" 2>/dev/null || true
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
                echo "  Merge: SUCCESS (clean merge, tests pass)"
            else
                echo "semantic_conflict,0,merged but tests fail" > "$trial_dir/merge_result.csv"
                echo "  Merge: SEMANTIC CONFLICT (merged cleanly but tests fail)"
            fi
            echo "$test_out" > "$trial_dir/merge_tests.txt"
        fi
    fi

    cd "$PROJECT_DIR"
    git worktree remove "$merge_wt" --force 2>/dev/null || true
    git branch -D "$merge_branch" 2>/dev/null || true
}

# --- Trial runner ---

run_trial() {
    local phase="$1"
    local task_type="$2"
    local trial_num="$3"
    local trial_dir="$4"
    local constraint_a="$5"   # empty for Phase 1
    local constraint_b="$6"   # empty for Phase 1

    mkdir -p "$trial_dir/agent-a" "$trial_dir/agent-b"

    # Create worktrees
    local wt_a="/tmp/attractor-${phase}-a-t${trial_num}-$$"
    local wt_b="/tmp/attractor-${phase}-b-t${trial_num}-$$"
    local branch_a="exp/attractor-${phase}-${task_type}-a-t${trial_num}-$$"
    local branch_b="exp/attractor-${phase}-${task_type}-b-t${trial_num}-$$"

    cd "$PROJECT_DIR"
    git worktree add -b "$branch_a" "$wt_a" "$BASELINE_COMMIT" 2>/dev/null
    git worktree add -b "$branch_b" "$wt_b" "$BASELINE_COMMIT" 2>/dev/null

    # Build prompts
    local base_a
    base_a=$(cat "$PROMPTS_DIR/${task_type}-a.md")
    local base_b
    base_b=$(cat "$PROMPTS_DIR/${task_type}-b.md")

    local prompt_a="$base_a"
    local prompt_b="$base_b"

    # Append constraints if provided (Phase 2)
    if [ -n "$constraint_a" ]; then
        prompt_a="${base_a}${constraint_a}"
    fi
    if [ -n "$constraint_b" ]; then
        prompt_b="${base_b}${constraint_b}"
    fi

    # Save prompts
    echo "$prompt_a" > "$trial_dir/agent-a/prompt.md"
    echo "$prompt_b" > "$trial_dir/agent-b/prompt.md"

    # Run both agents in parallel
    (run_agent "$wt_a" "$prompt_a" "$trial_dir/agent-a") &
    local pid_a=$!
    (run_agent "$wt_b" "$prompt_b" "$trial_dir/agent-b") &
    local pid_b=$!

    wait "$pid_a" || true
    wait "$pid_b" || true

    # Read durations
    local dur_a dur_b
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
}

# =========================================
#  PHASE 1: No attractors — collect collisions
# =========================================

echo ""
echo "=========================================="
echo "  PHASE 1: Collecting collision patterns"
echo "=========================================="
echo ""

for task_type in "${TASK_TYPES[@]}"; do
    phase1_dir="$RESULTS_DIR/phase1/$task_type"
    mkdir -p "$phase1_dir"

    for trial in $(seq 1 "$PHASE1_TRIALS"); do
        echo "--- Phase 1 | $task_type | trial $trial/$PHASE1_TRIALS ---"
        run_trial "p1" "$task_type" "$trial" "$phase1_dir/trial-$trial" "" ""
    done

    # Analyze collisions
    echo ""
    echo "=== Analyzing Phase 1 collisions for $task_type ==="
    bash "$SCRIPT_DIR/analyze_collisions.sh" "$phase1_dir" "$PROJECT_DIR/pkg/display/display.go"

    # Generate constraints
    echo ""
    echo "=== Generating constraints for $task_type ==="
    bash "$SCRIPT_DIR/generate_constraints.sh" \
        "$phase1_dir/collision_analysis.json" \
        "$PROJECT_DIR/pkg/display/display.go" \
        "$PROJECT_DIR/pkg/display/display_test.go"
done

# =========================================
#  PHASE 2: Auto-generated attractors
# =========================================

echo ""
echo "=========================================="
echo "  PHASE 2: Applying auto-generated attractors"
echo "=========================================="
echo ""

for task_type in "${TASK_TYPES[@]}"; do
    phase1_dir="$RESULTS_DIR/phase1/$task_type"
    phase2_dir="$RESULTS_DIR/phase2/$task_type"
    mkdir -p "$phase2_dir"

    # Load generated constraints
    constraint_a=$(cat "$phase1_dir/constraint-a.md" 2>/dev/null || echo "")
    constraint_b=$(cat "$phase1_dir/constraint-b.md" 2>/dev/null || echo "")

    if [ -z "$constraint_a" ] || [ -z "$constraint_b" ]; then
        echo "ERROR: No constraints generated for $task_type — skipping Phase 2"
        continue
    fi

    echo "Constraints loaded:"
    echo "  Agent A: $(head -3 "$phase1_dir/constraint-a.md" | tail -1)"
    echo "  Agent B: $(head -3 "$phase1_dir/constraint-b.md" | tail -1)"
    echo ""

    # Copy constraint metadata
    cp "$phase1_dir/constraint_metadata.json" "$phase2_dir/" 2>/dev/null || true
    cp "$phase1_dir/constraint-a.md" "$phase2_dir/" 2>/dev/null || true
    cp "$phase1_dir/constraint-b.md" "$phase2_dir/" 2>/dev/null || true

    for trial in $(seq 1 "$PHASE2_TRIALS"); do
        echo "--- Phase 2 | $task_type | trial $trial/$PHASE2_TRIALS ---"
        run_trial "p2" "$task_type" "$trial" "$phase2_dir/trial-$trial" "$constraint_a" "$constraint_b"
    done
done

# =========================================
#  SCORING & ANALYSIS
# =========================================

echo ""
echo "=========================================="
echo "  RESULTS SUMMARY"
echo "=========================================="
echo ""

# Score all trials
echo "=== Phase 1 (No Attractors) ==="
for task_type in "${TASK_TYPES[@]}"; do
    phase1_dir="$RESULTS_DIR/phase1/$task_type"
    p1_conflicts=0
    p1_success=0
    p1_total=0

    for trial_dir in "$phase1_dir"/trial-*/; do
        [ ! -d "$trial_dir" ] && continue
        p1_total=$((p1_total + 1))
        if [ -f "$trial_dir/merge_result.csv" ]; then
            result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
            case "$result" in
                conflict) p1_conflicts=$((p1_conflicts + 1)) ;;
                success) p1_success=$((p1_success + 1)) ;;
            esac
        fi
    done
    echo "  $task_type: $p1_conflicts/$p1_total CONFLICT, $p1_success/$p1_total SUCCESS"
done

echo ""
echo "=== Phase 2 (Auto-Generated Attractors) ==="
for task_type in "${TASK_TYPES[@]}"; do
    phase2_dir="$RESULTS_DIR/phase2/$task_type"
    p2_conflicts=0
    p2_success=0
    p2_build_fail=0
    p2_semantic=0
    p2_total=0

    for trial_dir in "$phase2_dir"/trial-*/; do
        [ ! -d "$trial_dir" ] && continue
        p2_total=$((p2_total + 1))
        if [ -f "$trial_dir/merge_result.csv" ]; then
            result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
            case "$result" in
                conflict) p2_conflicts=$((p2_conflicts + 1)) ;;
                success) p2_success=$((p2_success + 1)) ;;
                build_fail) p2_build_fail=$((p2_build_fail + 1)) ;;
                semantic_conflict) p2_semantic=$((p2_semantic + 1)) ;;
            esac
        fi
    done
    echo "  $task_type: $p2_success/$p2_total SUCCESS, $p2_conflicts/$p2_total CONFLICT, $p2_build_fail/$p2_total BUILD_FAIL, $p2_semantic/$p2_total SEMANTIC"
done

# Score individual agents
echo ""
echo "=== Individual Agent Scores ==="
bash "$SCRIPT_DIR/score.sh" "$RESULTS_DIR/phase1" 2>/dev/null || echo "(Phase 1 scoring skipped)"
bash "$SCRIPT_DIR/score.sh" "$RESULTS_DIR/phase2" 2>/dev/null || echo "(Phase 2 scoring skipped)"

# Compute convergence: minimum Phase 1 trials needed
echo ""
echo "=== Convergence Analysis ==="
echo "How many Phase 1 failures does the system need to learn effective attractors?"
for task_type in "${TASK_TYPES[@]}"; do
    phase1_dir="$RESULTS_DIR/phase1/$task_type"
    first_collision=0
    for trial_dir in "$phase1_dir"/trial-*/; do
        [ ! -d "$trial_dir" ] && continue
        trial_num=$(basename "$trial_dir" | sed 's/trial-//')
        if [ -f "$trial_dir/merge_result.csv" ]; then
            result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
            if [ "$result" = "conflict" ] && [ "$first_collision" -eq 0 ]; then
                first_collision=$trial_num
            fi
        fi
    done
    echo "  $task_type: First collision at trial $first_collision"
    echo "  → Theoretically, 1 collision is sufficient for constraint generation"
done

# Write final analysis
cat > "$RESULTS_DIR/analysis.md" << 'ENDANALYSIS'
# Automated Attractor Discovery Experiment

## Design

Two-phase experiment testing whether merge conflict patterns contain sufficient
information to automatically generate effective structural placement constraints.

### Phase 1: Collision Collection
- No coordination instructions
- Both agents independently choose insertion points
- Merge conflicts recorded with full diff context

### Phase 2: Auto-Generated Attractors
- Collision patterns from Phase 1 parsed automatically
- Gravitational insertion point identified (where both agents insert)
- Alternative insertion points discovered from file structure
- Non-overlapping placement constraints generated and injected

## Results

ENDANALYSIS

# Append actual results
echo "### Phase 1 (No Attractors)" >> "$RESULTS_DIR/analysis.md"
echo "" >> "$RESULTS_DIR/analysis.md"
for task_type in "${TASK_TYPES[@]}"; do
    phase1_dir="$RESULTS_DIR/phase1/$task_type"
    p1_c=0; p1_s=0; p1_t=0
    for td in "$phase1_dir"/trial-*/; do
        [ ! -d "$td" ] && continue
        p1_t=$((p1_t + 1))
        [ -f "$td/merge_result.csv" ] && {
            r=$(cut -d',' -f1 "$td/merge_result.csv")
            [ "$r" = "conflict" ] && p1_c=$((p1_c + 1))
            [ "$r" = "success" ] && p1_s=$((p1_s + 1))
        }
    done
    echo "- **$task_type**: $p1_c/$p1_t CONFLICT, $p1_s/$p1_t SUCCESS" >> "$RESULTS_DIR/analysis.md"
done

echo "" >> "$RESULTS_DIR/analysis.md"
echo "### Phase 2 (Auto-Generated Attractors)" >> "$RESULTS_DIR/analysis.md"
echo "" >> "$RESULTS_DIR/analysis.md"
for task_type in "${TASK_TYPES[@]}"; do
    phase2_dir="$RESULTS_DIR/phase2/$task_type"
    p2_c=0; p2_s=0; p2_bf=0; p2_sem=0; p2_t=0
    for td in "$phase2_dir"/trial-*/; do
        [ ! -d "$td" ] && continue
        p2_t=$((p2_t + 1))
        [ -f "$td/merge_result.csv" ] && {
            r=$(cut -d',' -f1 "$td/merge_result.csv")
            [ "$r" = "conflict" ] && p2_c=$((p2_c + 1))
            [ "$r" = "success" ] && p2_s=$((p2_s + 1))
            [ "$r" = "build_fail" ] && p2_bf=$((p2_bf + 1))
            [ "$r" = "semantic_conflict" ] && p2_sem=$((p2_sem + 1))
        }
    done
    echo "- **$task_type**: $p2_s/$p2_t SUCCESS, $p2_c/$p2_t CONFLICT, $p2_bf/$p2_t BUILD_FAIL, $p2_sem/$p2_t SEMANTIC" >> "$RESULTS_DIR/analysis.md"
done

# Append constraint metadata
echo "" >> "$RESULTS_DIR/analysis.md"
echo "### Auto-Generated Constraints" >> "$RESULTS_DIR/analysis.md"
echo "" >> "$RESULTS_DIR/analysis.md"
for task_type in "${TASK_TYPES[@]}"; do
    phase1_dir="$RESULTS_DIR/phase1/$task_type"
    if [ -f "$phase1_dir/constraint_metadata.json" ]; then
        echo "**$task_type:**" >> "$RESULTS_DIR/analysis.md"
        echo '```json' >> "$RESULTS_DIR/analysis.md"
        cat "$phase1_dir/constraint_metadata.json" >> "$RESULTS_DIR/analysis.md"
        echo '```' >> "$RESULTS_DIR/analysis.md"
    fi
done

echo ""
echo "=========================================="
echo "  EXPERIMENT COMPLETE"
echo "=========================================="
echo "Results: $RESULTS_DIR"
echo "Analysis: $RESULTS_DIR/analysis.md"
