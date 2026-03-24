#!/usr/bin/env bash
# Analyze results for decomposition quality experiment
#
# Produces:
#   1. Merge results table by condition
#   2. Individual agent success rates
#   3. Duration summary
#   4. Diff hunk analysis with anchoring variance
#   5. Decomposition quality gradient summary
#
# Usage: ./analyze-decomposition.sh RESULTS_DIR

set -euo pipefail

RESULTS_DIR="${1:?Usage: analyze-decomposition.sh RESULTS_DIR}"
ANALYSIS_FILE="$RESULTS_DIR/analysis.md"

echo "# Decomposition Quality Experiment Results" > "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "Generated: $(date)" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "## Hypothesis" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "Decomposition quality (task description + file structure) predicts conflict rate. Best decomposition (anchored-sectioned) should approach 0% conflict without any coordination primitives." >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"

# Collect merge results
echo "## Merge Results by Condition" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "| Condition | Trials | Conflicts | Clean Merge | Build Fail | Semantic Fail | No Change | Conflict % |" >> "$ANALYSIS_FILE"
echo "|-----------|--------|-----------|-------------|------------|---------------|-----------|------------|" >> "$ANALYSIS_FILE"

echo ""
echo "=== Merge Results by Condition ==="
printf "%-22s %6s %9s %11s %10s %13s %9s %10s\n" \
    "Condition" "Trials" "Conflicts" "Clean" "BuildFail" "SemanticFail" "NoChange" "Conflict%"

# Process in desired order
for condition in bare-flat rich-flat anchored-flat bare-sectioned anchored-sectioned; do
    condition_dir="$RESULTS_DIR/$condition"
    [ ! -d "$condition_dir" ] && continue

    conflicts=0
    success=0
    build_fail=0
    semantic=0
    no_change=0
    total=0

    for trial_dir in "$condition_dir"/trial-*/; do
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

    if [ "$total" -gt 0 ]; then
        conflict_pct=$((conflicts * 100 / total))
    else
        conflict_pct=0
    fi

    printf "%-22s %6d %9d %11d %10d %13d %9d %9d%%\n" \
        "$condition" "$total" "$conflicts" "$success" "$build_fail" "$semantic" "$no_change" "$conflict_pct"
    echo "| $condition | $total | $conflicts | $success | $build_fail | $semantic | $no_change | ${conflict_pct}% |" >> "$ANALYSIS_FILE"
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

    for condition in bare-flat rich-flat anchored-flat bare-sectioned anchored-sectioned; do
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
                printf "  %-22s agent-%s: avg=%d.%02d/6 perfect=%d/%d\n" \
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

for condition in bare-flat rich-flat anchored-flat bare-sectioned anchored-sectioned; do
    condition_dir="$RESULTS_DIR/$condition"
    [ ! -d "$condition_dir" ] && continue

    durations_a=""
    durations_b=""
    for trial_dir in "$condition_dir"/trial-*/; do
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
            printf "  %-22s: agent-a avg=%ds, agent-b avg=%ds\n" \
                "$condition" "$avg_a" "$avg_b"
            echo "- **$condition**: Agent A avg=${avg_a}s, Agent B avg=${avg_b}s" >> "$ANALYSIS_FILE"
        fi
    fi
done

# Diff hunk analysis with anchoring variance
echo "" >> "$ANALYSIS_FILE"
echo "## Anchoring Analysis" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "Where did each agent's changes land? Lower variance = stronger anchoring." >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"

echo ""
echo "=== Anchoring Analysis ==="

for condition in bare-flat rich-flat anchored-flat bare-sectioned anchored-sectioned; do
    condition_dir="$RESULTS_DIR/$condition"
    [ ! -d "$condition_dir" ] && continue

    echo "### $condition" >> "$ANALYSIS_FILE"
    echo "" >> "$ANALYSIS_FILE"

    # Collect hunk start lines for each agent across trials
    hunk_starts_a=""
    hunk_starts_b=""

    for trial_dir in "$condition_dir"/trial-*/; do
        [ ! -d "$trial_dir" ] && continue
        trial=$(basename "$trial_dir" | sed 's/trial-//')

        # Extract first hunk start line from display.go diff
        start_a=""
        start_b=""
        if [ -f "$trial_dir/agent-a/full_diff.txt" ]; then
            start_a=$(grep '^@@ ' "$trial_dir/agent-a/full_diff.txt" | grep 'display.go' -A0 | head -1 | sed 's/^@@ -\([0-9]*\).*/\1/' 2>/dev/null || echo "")
            if [ -z "$start_a" ]; then
                start_a=$(grep '^@@ ' "$trial_dir/agent-a/full_diff.txt" | head -1 | sed 's/^@@ -\([0-9]*\).*/\1/' 2>/dev/null || echo "")
            fi
        fi
        if [ -f "$trial_dir/agent-b/full_diff.txt" ]; then
            start_b=$(grep '^@@ ' "$trial_dir/agent-b/full_diff.txt" | grep 'display.go' -A0 | head -1 | sed 's/^@@ -\([0-9]*\).*/\1/' 2>/dev/null || echo "")
            if [ -z "$start_b" ]; then
                start_b=$(grep '^@@ ' "$trial_dir/agent-b/full_diff.txt" | head -1 | sed 's/^@@ -\([0-9]*\).*/\1/' 2>/dev/null || echo "")
            fi
        fi

        [ -n "$start_a" ] && hunk_starts_a="$hunk_starts_a $start_a"
        [ -n "$start_b" ] && hunk_starts_b="$hunk_starts_b $start_b"

        # Per-trial hunk detail
        hunks_a=$(grep '^@@' "$trial_dir/agent-a/full_diff.txt" 2>/dev/null | head -3 || echo "(none)")
        hunks_b=$(grep '^@@' "$trial_dir/agent-b/full_diff.txt" 2>/dev/null | head -3 || echo "(none)")
        echo "- Trial $trial: A=[$start_a] B=[$start_b]" >> "$ANALYSIS_FILE"
    done

    # Compute anchoring statistics
    if [ -n "$hunk_starts_a" ]; then
        # Mean and variance for agent A
        sum=0; count=0
        for v in $hunk_starts_a; do sum=$((sum + v)); count=$((count + 1)); done
        if [ "$count" -gt 0 ]; then
            mean_a=$((sum / count))
            var_sum=0
            for v in $hunk_starts_a; do
                diff=$((v - mean_a))
                var_sum=$((var_sum + diff * diff))
            done
            var_a=$((var_sum / count))
            echo "  Agent A: mean_line=$mean_a variance=$var_a (N=$count)"
            echo "" >> "$ANALYSIS_FILE"
            echo "Agent A anchoring: mean=$mean_a, variance=$var_a (N=$count)" >> "$ANALYSIS_FILE"
        fi
    fi

    if [ -n "$hunk_starts_b" ]; then
        sum=0; count=0
        for v in $hunk_starts_b; do sum=$((sum + v)); count=$((count + 1)); done
        if [ "$count" -gt 0 ]; then
            mean_b=$((sum / count))
            var_sum=0
            for v in $hunk_starts_b; do
                diff=$((v - mean_b))
                var_sum=$((var_sum + diff * diff))
            done
            var_b=$((var_sum / count))
            echo "  Agent B: mean_line=$mean_b variance=$var_b (N=$count)"
            echo "Agent B anchoring: mean=$mean_b, variance=$var_b (N=$count)" >> "$ANALYSIS_FILE"
        fi
    fi

    echo "" >> "$ANALYSIS_FILE"
done

# Summary comparison with prior experiments
echo "" >> "$ANALYSIS_FILE"
echo "## Comparison with Prior Data" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "| Experiment | Condition | Conflict Rate | N |" >> "$ANALYSIS_FILE"
echo "|------------|-----------|---------------|---|" >> "$ANALYSIS_FILE"
echo "| Prior (Mar 10) | no-coord additive | 100% | 20 |" >> "$ANALYSIS_FILE"
echo "| Prior (Mar 10) | placement additive | 0% | 20 |" >> "$ANALYSIS_FILE"
echo "| Prior (Mar 23) | no-coord modification | 0% | 40 |" >> "$ANALYSIS_FILE"

for condition in bare-flat rich-flat anchored-flat bare-sectioned anchored-sectioned; do
    condition_dir="$RESULTS_DIR/$condition"
    [ ! -d "$condition_dir" ] && continue
    conflicts=0; total=0
    for trial_dir in "$condition_dir"/trial-*/; do
        [ ! -d "$trial_dir" ] && continue
        total=$((total + 1))
        if [ -f "$trial_dir/merge_result.csv" ]; then
            result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
            [ "$result" = "conflict" ] && conflicts=$((conflicts + 1))
        fi
    done
    if [ "$total" -gt 0 ]; then
        pct=$((conflicts * 100 / total))
        echo "| This (decomp) | $condition | ${pct}% | $total |" >> "$ANALYSIS_FILE"
    fi
done

echo "" >> "$ANALYSIS_FILE"
echo "## Interpretation Guide" >> "$ANALYSIS_FILE"
echo "" >> "$ANALYSIS_FILE"
echo "- If bare-flat ~100%: Confirms baseline (replicates prior no-coord)" >> "$ANALYSIS_FILE"
echo "- If anchored-sectioned ~0%: Decomposition quality CAN eliminate coordination need" >> "$ANALYSIS_FILE"
echo "- If gradient is monotonic (C1>C2>C3>C4>C5): Relationship is continuous" >> "$ANALYSIS_FILE"
echo "- If anchored-flat alone ~0%: File structure doesn't matter, task descriptions suffice" >> "$ANALYSIS_FILE"
echo "- If anchored-sectioned still >50%: Decomposition hypothesis is wrong" >> "$ANALYSIS_FILE"

echo ""
echo "Analysis written to: $ANALYSIS_FILE"
