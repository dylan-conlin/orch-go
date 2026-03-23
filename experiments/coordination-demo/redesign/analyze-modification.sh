#!/usr/bin/env bash
# Analyze results for modification experiment
#
# Usage: ./analyze-modification.sh RESULTS_DIR

set -euo pipefail

RESULTS_DIR="${1:?Usage: analyze-modification.sh RESULTS_DIR}"
ANALYSIS_FILE="$RESULTS_DIR/analysis.md"

echo "# Modification Task Coordination Experiment Results" > "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "Generated: $(date)" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "## Hypothesis" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "Modification tasks should produce 0% conflict rate across all conditions because agents are anchored to different functions (no gravitational insertion point)." >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"

# Collect merge results
echo "## Merge Results by Condition" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "| Condition | Trials | Conflicts | Clean Merge | Build Fail | Semantic Fail | No Change |" >> "$ANALYSIS_FILE"
echo "|-----------|--------|-----------|-------------|------------|---------------|-----------|" >> "$ANALYSIS_FILE"

echo ""
echo "=== Merge Results by Condition ==="
printf "%-14s %6s %9s %11s %10s %13s %9s\n" \
    "Condition" "Trials" "Conflicts" "Clean" "BuildFail" "SemanticFail" "NoChange"

for condition_dir in "$RESULTS_DIR"/*/; do
    condition=$(basename "$condition_dir")
    [ ! -d "$condition_dir" ] && continue
    [[ "$condition" =~ \. ]] && continue

    conflicts=0
    success=0
    build_fail=0
    semantic=0
    no_change=0
    total=0

    for trial_dir in "$condition_dir"trial-*/; do
        [ ! -d "$trial_dir" ] && continue
        total=$((total + 1))

        if [ -f "$trial_dir/merge_result.csv" ]; then
            result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
            case "$result" in
                conflict) conflicts=$((conflicts + 1)) ;;
                success) success=$((success + 1)) ;;
                build_fail) build_fail=$((build_fail + 1)) ;;
                semantic_conflict) semantic=$((semantic + 1)) ;;
                no_change) no_change=$((no_change + 1)) ;;
            esac
        fi
    done

    printf "%-14s %6d %9d %11d %10d %13d %9d\n" \
        "$condition" "$total" "$conflicts" "$success" "$build_fail" "$semantic" "$no_change"
    echo "| $condition | $total | $conflicts | $success | $build_fail | $semantic | $no_change |" >> "$ANALYSIS_FILE"
done

# Individual success rates
echo "" >> "$ANALYSIS_FILE"
echo "## Individual Agent Success Rates" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"

if [ -f "$RESULTS_DIR/scores.csv" ]; then
    echo "| Condition | Agent | Avg Score | Perfect (6/6) | Trials |" >> "$ANALYSIS_FILE"
    echo "|-----------|-------|-----------|---------------|--------|" >> "$ANALYSIS_FILE"

    echo ""
    echo "=== Individual Success Rates ==="

    for condition in no-coord placement context-share messaging; do
        for agent in a b; do
            total_score=0
            perfect=0
            count=0
            while IFS=',' read -r c t ag comp build tests regr files spec dur tot; do
                if [ "$c" = "$condition" ] && [ "$ag" = "$agent" ]; then
                    total_score=$((total_score + tot))
                    count=$((count + 1))
                    if [ "$tot" = "6" ]; then
                        perfect=$((perfect + 1))
                    fi
                fi
            done < <(tail -n +2 "$RESULTS_DIR/scores.csv")

            if [ "$count" -gt 0 ]; then
                avg=$((total_score * 100 / count))
                printf "  %-14s agent-%s: avg=%d.%02d/6 perfect=%d/%d\n" \
                    "$condition" "$agent" \
                    "$((avg / 100))" "$((avg % 100))" "$perfect" "$count"
                echo "| $condition | $agent | $((avg / 100)).$((avg % 100))/6 | $perfect/$count | $count |" >> "$ANALYSIS_FILE"
            fi
        done
    done
fi

# Duration summary
echo "" >> "$ANALYSIS_FILE"
echo "## Duration Summary" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"

echo ""
echo "=== Duration Summary ==="

for condition_dir in "$RESULTS_DIR"/*/; do
    condition=$(basename "$condition_dir")
    [ ! -d "$condition_dir" ] && continue
    [[ "$condition" =~ \. ]] && continue

    durations_a=""
    durations_b=""
    for trial_dir in "$condition_dir"trial-*/; do
        [ ! -d "$trial_dir" ] && continue
        da=$(cat "$trial_dir/agent-a/duration_seconds" 2>/dev/null || echo "0")
        db=$(cat "$trial_dir/agent-b/duration_seconds" 2>/dev/null || echo "0")
        durations_a="$durations_a $da"
        durations_b="$durations_b $db"
    done

    if [ -n "$durations_a" ]; then
        sum_a=0; count_a=0
        for d in $durations_a; do sum_a=$((sum_a + d)); count_a=$((count_a + 1)); done
        sum_b=0; count_b=0
        for d in $durations_b; do sum_b=$((sum_b + d)); count_b=$((count_b + 1)); done

        if [ "$count_a" -gt 0 ] && [ "$count_b" -gt 0 ]; then
            avg_a=$((sum_a / count_a))
            avg_b=$((sum_b / count_b))
            printf "  %-14s: agent-a avg=%ds, agent-b avg=%ds\n" \
                "$condition" "$avg_a" "$avg_b"
            echo "- **$condition**: Agent A avg=${avg_a}s, Agent B avg=${avg_b}s" >> "$ANALYSIS_FILE"
        fi
    fi
done

# Messaging artifacts summary
echo "" >> "$ANALYSIS_FILE"
echo "## Messaging Condition Artifacts" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"

msg_dir_found=false
for trial_dir in "$RESULTS_DIR"/messaging/trial-*/; do
    [ ! -d "$trial_dir" ] && continue
    trial=$(basename "$trial_dir" | sed 's/trial-//')

    if [ -d "$trial_dir/messages" ]; then
        msg_dir_found=true
        plan_a_exists="no"
        plan_b_exists="no"
        [ -f "$trial_dir/messages/plan-a.txt" ] && plan_a_exists="yes"
        [ -f "$trial_dir/messages/plan-b.txt" ] && plan_b_exists="yes"
        echo "- Trial $trial: plan-a=$plan_a_exists, plan-b=$plan_b_exists" >> "$ANALYSIS_FILE"
    fi
done

if [ "$msg_dir_found" = false ]; then
    echo "No messaging condition data found." >> "$ANALYSIS_FILE"
fi

# Diff hunk analysis
echo "" >> "$ANALYSIS_FILE"
echo "## Diff Hunk Analysis" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "Where did each agent's changes land in the file?" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"

echo ""
echo "=== Diff Hunk Analysis ==="

for condition_dir in "$RESULTS_DIR"/*/; do
    condition=$(basename "$condition_dir")
    [ ! -d "$condition_dir" ] && continue
    [[ "$condition" =~ \. ]] && continue

    echo "### $condition" >> "$ANALYSIS_FILE"
    echo "" >> "$ANALYSIS_FILE"

    for trial_dir in "$condition_dir"trial-*/; do
        [ ! -d "$trial_dir" ] && continue
        trial=$(basename "$trial_dir" | sed 's/trial-//')

        hunks_a=""
        hunks_b=""
        if [ -f "$trial_dir/agent-a/full_diff.txt" ]; then
            hunks_a=$(grep '^@@' "$trial_dir/agent-a/full_diff.txt" 2>/dev/null | head -5 || echo "(none)")
        fi
        if [ -f "$trial_dir/agent-b/full_diff.txt" ]; then
            hunks_b=$(grep '^@@' "$trial_dir/agent-b/full_diff.txt" 2>/dev/null | head -5 || echo "(none)")
        fi

        echo "- Trial $trial:" >> "$ANALYSIS_FILE"
        echo "  - Agent A hunks: $hunks_a" >> "$ANALYSIS_FILE"
        echo "  - Agent B hunks: $hunks_b" >> "$ANALYSIS_FILE"
    done
    echo "" >> "$ANALYSIS_FILE"
done

echo ""
echo "Analysis written to: $ANALYSIS_FILE"
