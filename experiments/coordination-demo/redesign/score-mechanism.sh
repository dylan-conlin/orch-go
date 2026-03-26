#!/usr/bin/env bash
# Score and analyze mechanism discrimination experiment results
#
# Reads scores.json from each trial and produces:
#   1. Per-N compliance summary (Phase 1)
#   2. Per-constraint survival curve
#   3. Variance analysis
#   4. Set comparison (Phase 2)
#   5. Removal recovery matrix (Phase 3)
#
# Usage:
#   ./score-mechanism.sh results/mechanism-p1-TIMESTAMP
#   ./score-mechanism.sh results/mechanism-p2-TIMESTAMP
#   ./score-mechanism.sh results/mechanism-p3-TIMESTAMP

set -euo pipefail

RESULTS_DIR="${1:?Usage: $0 <results-dir>}"

if [ ! -d "$RESULTS_DIR" ]; then
    echo "ERROR: Results directory not found: $RESULTS_DIR"
    exit 1
fi

PHASE=$(jq -r '.phase' "$RESULTS_DIR/metadata.json" 2>/dev/null || echo "1")

echo "================================================"
echo "  MECHANISM EXPERIMENT ANALYSIS (Phase $PHASE)"
echo "================================================"
echo "Results: $RESULTS_DIR"
echo ""

# ============================================================
# PHASE 1 ANALYSIS: Scaling Curve
# ============================================================

analyze_phase1() {
    echo "=== DEGRADATION CURVE ==="
    echo ""
    printf "%-4s  %-8s  %-8s  %-8s  %-8s  %-10s\n" "N" "Trials" "Mean" "StdDev" "Min" "Max"
    printf "%-4s  %-8s  %-8s  %-8s  %-8s  %-10s\n" "---" "------" "------" "------" "------" "--------"

    local all_n_data=""

    for n_dir in "$RESULTS_DIR"/phase1/n*/; do
        [ -d "$n_dir" ] || continue
        local n
        n=$(basename "$n_dir" | sed 's/n//')

        local rates=()
        local trial_count=0

        for trial_dir in "$n_dir"/trial-*/; do
            [ -f "$trial_dir/scores.json" ] || continue
            local dry_run
            dry_run=$(jq -r '.dry_run // false' "$trial_dir/scores.json" 2>/dev/null)
            [ "$dry_run" = "true" ] && continue

            local rate
            rate=$(jq -r '.compliance_rate' "$trial_dir/scores.json" 2>/dev/null)
            [ "$rate" != "null" ] && rates+=("$rate") && trial_count=$((trial_count + 1))
        done

        if [ ${#rates[@]} -eq 0 ]; then
            printf "%-4s  %-8s  %-8s\n" "$n" "0" "no data"
            continue
        fi

        # Calculate mean, stddev, min, max
        local stats
        stats=$(printf '%s\n' "${rates[@]}" | awk '
        {
            sum += $1; sumsq += $1*$1; count++
            vals[count] = $1
        }
        END {
            mean = sum/count
            variance = (sumsq/count) - (mean*mean)
            if (variance < 0) variance = 0
            stddev = sqrt(variance)
            min = vals[1]; max = vals[1]
            for (i = 1; i <= count; i++) {
                if (vals[i] < min) min = vals[i]
                if (vals[i] > max) max = vals[i]
            }
            printf "%.3f %.3f %.3f %.3f", mean, stddev, min, max
        }')

        read -r mean stddev min max <<< "$stats"
        printf "%-4s  %-8s  %-8s  %-8s  %-8s  %-10s\n" "$n" "$trial_count" "$mean" "$stddev" "$min" "$max"

        all_n_data+="$n $mean $stddev
"
    done

    # Curve shape analysis
    echo ""
    echo "=== CURVE SHAPE ANALYSIS ==="
    echo ""
    echo "$all_n_data" | awk '
    NF >= 2 {
        n[NR] = $1; rate[NR] = $2; sd[NR] = $3; count = NR
    }
    END {
        if (count < 3) { print "Insufficient data points for shape analysis"; exit }

        # Check for gradual decline (resource competition signature)
        monotonic_decline = 1
        max_drop = 0
        for (i = 2; i <= count; i++) {
            drop = rate[i-1] - rate[i]
            if (drop < -0.01) monotonic_decline = 0  # increase
            if (drop > max_drop) { max_drop = drop; max_drop_n = n[i] }
        }

        total_decline = rate[1] - rate[count]
        if (total_decline < 0.05) {
            print "SHAPE: FLAT — minimal degradation detected"
            print "  Total decline: " total_decline
            print "  Interpretation: constraints may not degrade at tested scale"
        } else if (max_drop > total_decline * 0.6) {
            print "SHAPE: STEP FUNCTION — most degradation at single point"
            printf "  Largest drop: %.3f at N=%s\n", max_drop, max_drop_n
            printf "  Total decline: %.3f\n", total_decline
            print "  Interpretation: CONSISTENT WITH threshold collapse"
        } else if (monotonic_decline) {
            print "SHAPE: GRADUAL DECLINE — smooth monotonic degradation"
            printf "  Total decline: %.3f\n", total_decline
            print "  Interpretation: CONSISTENT WITH resource competition"
        } else {
            print "SHAPE: IRREGULAR — non-monotonic with local maxima"
            printf "  Total decline: %.3f\n", total_decline
            print "  Interpretation: CONSISTENT WITH interference (pair-dependent)"
        }
    }'

    # Per-constraint survival analysis
    echo ""
    echo "=== PER-CONSTRAINT SURVIVAL ==="
    echo ""
    echo "Shows which constraints survive as N increases."
    echo "(1 = compliant in all trials at that N, 0 = never compliant)"
    echo ""

    # Collect all constraint IDs that appear in any trial
    local all_cids=()
    for trial_dir in "$RESULTS_DIR"/phase1/n*/trial-*/; do
        [ -f "$trial_dir/scores.json" ] || continue
        local cids
        cids=$(jq -r '.per_constraint // {} | keys[]' "$trial_dir/scores.json" 2>/dev/null)
        for cid in $cids; do
            local found=false
            for existing in "${all_cids[@]+"${all_cids[@]}"}"; do
                [ "$existing" = "$cid" ] && found=true && break
            done
            [ "$found" = false ] && all_cids+=("$cid")
        done
    done

    # Sort
    IFS=$'\n' all_cids=($(sort <<< "${all_cids[*]}")); unset IFS

    printf "%-5s" ""
    for n_dir in "$RESULTS_DIR"/phase1/n*/; do
        [ -d "$n_dir" ] || continue
        local n
        n=$(basename "$n_dir" | sed 's/n//')
        printf "  N=%-3s" "$n"
    done
    echo ""

    for cid in "${all_cids[@]}"; do
        printf "%-5s" "$cid"
        for n_dir in "$RESULTS_DIR"/phase1/n*/; do
            [ -d "$n_dir" ] || continue

            local present=0
            local compliant=0
            for trial_dir in "$n_dir"/trial-*/; do
                [ -f "$trial_dir/scores.json" ] || continue
                local val
                val=$(jq -r ".per_constraint.\"$cid\" // \"absent\"" "$trial_dir/scores.json" 2>/dev/null)
                if [ "$val" != "absent" ]; then
                    present=$((present + 1))
                    [ "$val" -eq 1 ] && compliant=$((compliant + 1))
                fi
            done

            if [ "$present" -eq 0 ]; then
                printf "  %-5s" "-"
            else
                local frac
                frac=$(echo "scale=1; $compliant * 100 / $present" | bc)
                printf "  %-5s" "${frac}%"
            fi
        done
        echo ""
    done

    # Determinism check
    echo ""
    echo "=== DETERMINISM CHECK ==="
    echo "Do constraints fail deterministically (0/5 or 5/5) or probabilistically?"
    echo ""
    local deterministic=0
    local probabilistic=0
    local total_checks=0

    for n_dir in "$RESULTS_DIR"/phase1/n*/; do
        [ -d "$n_dir" ] || continue
        for cid in "${all_cids[@]}"; do
            local present=0
            local compliant=0
            for trial_dir in "$n_dir"/trial-*/; do
                [ -f "$trial_dir/scores.json" ] || continue
                local val
                val=$(jq -r ".per_constraint.\"$cid\" // \"absent\"" "$trial_dir/scores.json" 2>/dev/null)
                if [ "$val" != "absent" ]; then
                    present=$((present + 1))
                    [ "$val" -eq 1 ] && compliant=$((compliant + 1))
                fi
            done

            if [ "$present" -ge 3 ]; then
                total_checks=$((total_checks + 1))
                if [ "$compliant" -eq 0 ] || [ "$compliant" -eq "$present" ]; then
                    deterministic=$((deterministic + 1))
                else
                    probabilistic=$((probabilistic + 1))
                fi
            fi
        done
    done

    if [ "$total_checks" -gt 0 ]; then
        local det_pct
        det_pct=$(echo "scale=1; $deterministic * 100 / $total_checks" | bc)
        echo "Deterministic (0/N or N/N): $deterministic/$total_checks ($det_pct%)"
        echo "Probabilistic (mixed):      $probabilistic/$total_checks"
        echo ""
        if [ "$(echo "$det_pct > 80" | bc)" -eq 1 ]; then
            echo "HIGH DETERMINISM: Consistent with threshold collapse or interference"
        elif [ "$(echo "$det_pct < 40" | bc)" -eq 1 ]; then
            echo "LOW DETERMINISM: Consistent with resource competition"
        else
            echo "MIXED: May indicate multiple mechanisms at different scales"
        fi
    fi
}

# ============================================================
# PHASE 2 ANALYSIS: Set Comparison
# ============================================================

analyze_phase2() {
    echo "=== SET COMPARISON ==="
    echo ""
    printf "%-14s  %-8s  %-8s  %-8s\n" "Set" "Mean" "StdDev" "Trials"
    printf "%-14s  %-8s  %-8s  %-8s\n" "----------" "------" "------" "------"

    local set_means=""

    for set_dir in "$RESULTS_DIR"/phase2/*/; do
        [ -d "$set_dir" ] || continue
        local set_name
        set_name=$(basename "$set_dir")

        local rates=()
        for trial_dir in "$set_dir"/trial-*/; do
            [ -f "$trial_dir/scores.json" ] || continue
            local dry_run
            dry_run=$(jq -r '.dry_run // false' "$trial_dir/scores.json" 2>/dev/null)
            [ "$dry_run" = "true" ] && continue
            local rate
            rate=$(jq -r '.compliance_rate' "$trial_dir/scores.json" 2>/dev/null)
            [ "$rate" != "null" ] && rates+=("$rate")
        done

        if [ ${#rates[@]} -eq 0 ]; then
            printf "%-14s  %-8s\n" "$set_name" "no data"
            continue
        fi

        local stats
        stats=$(printf '%s\n' "${rates[@]}" | awk '
        { sum += $1; sumsq += $1*$1; count++ }
        END {
            mean = sum/count
            variance = (sumsq/count) - (mean*mean)
            if (variance < 0) variance = 0
            stddev = sqrt(variance)
            printf "%.3f %.3f %d", mean, stddev, count
        }')

        read -r mean stddev count <<< "$stats"
        printf "%-14s  %-8s  %-8s  %-8s\n" "$set_name" "$mean" "$stddev" "$count"
        set_means+="$set_name $mean
"
    done

    echo ""
    echo "=== SET COMPARISON INTERPRETATION ==="
    echo ""

    local distributed_mean clustered_mean testing_mean
    distributed_mean=$(echo "$set_means" | awk '/distributed/ {print $2}')
    clustered_mean=$(echo "$set_means" | awk '/clustered/ {print $2}')
    testing_mean=$(echo "$set_means" | awk '/testing/ {print $2}')

    if [ -n "$distributed_mean" ] && [ -n "$clustered_mean" ]; then
        local diff
        diff=$(echo "scale=3; $distributed_mean - $clustered_mean" | bc)
        echo "Distributed - Clustered = $diff"

        if [ "$(echo "$diff > 0.10" | bc)" -eq 1 ]; then
            echo "SIGNIFICANT DIFFERENCE: Clustered constraints comply less"
            echo "Interpretation: CONSISTENT WITH interference (semantic proximity matters)"
        elif [ "$(echo "$diff < -0.10" | bc)" -eq 1 ]; then
            echo "REVERSE EFFECT: Clustered constraints comply MORE"
            echo "Interpretation: Semantic coherence may help, not hinder"
        else
            echo "NO SIGNIFICANT DIFFERENCE: Set composition doesn't matter at this N"
            echo "Interpretation: CONSISTENT WITH resource competition or threshold (only N matters)"
        fi
    fi
}

# ============================================================
# PHASE 3 ANALYSIS: Removal Recovery
# ============================================================

analyze_phase3() {
    echo "=== REMOVAL RECOVERY MATRIX ==="
    echo ""
    echo "Shows compliance rate change when each constraint is removed."
    echo ""

    # Get baseline
    local baseline_rates=()
    for trial_dir in "$RESULTS_DIR"/phase3/baseline/trial-*/; do
        [ -f "$trial_dir/scores.json" ] || continue
        local rate
        rate=$(jq -r '.compliance_rate' "$trial_dir/scores.json" 2>/dev/null)
        [ "$rate" != "null" ] && baseline_rates+=("$rate")
    done

    local baseline_mean="0"
    if [ ${#baseline_rates[@]} -gt 0 ]; then
        baseline_mean=$(printf '%s\n' "${baseline_rates[@]}" | awk '{ sum += $1; count++ } END { printf "%.3f", sum/count }')
    fi

    printf "%-10s  %-10s  %-10s  %-10s\n" "Removed" "Mean" "Delta" "Recovery"
    printf "%-10s  %-10s  %-10s  %-10s\n" "--------" "--------" "--------" "--------"
    printf "%-10s  %-10s  %-10s  %-10s\n" "baseline" "$baseline_mean" "-" "-"

    local recovery_pattern=""

    for remove_dir in "$RESULTS_DIR"/phase3/remove-*/; do
        [ -d "$remove_dir" ] || continue
        local removed
        removed=$(basename "$remove_dir" | sed 's/remove-//')

        local rates=()
        for trial_dir in "$remove_dir"/trial-*/; do
            [ -f "$trial_dir/scores.json" ] || continue
            local rate
            rate=$(jq -r '.compliance_rate' "$trial_dir/scores.json" 2>/dev/null)
            [ "$rate" != "null" ] && rates+=("$rate")
        done

        if [ ${#rates[@]} -eq 0 ]; then
            printf "%-10s  %-10s\n" "$removed" "no data"
            continue
        fi

        local mean
        mean=$(printf '%s\n' "${rates[@]}" | awk '{ sum += $1; count++ } END { printf "%.3f", sum/count }')

        local delta
        delta=$(echo "scale=3; $mean - $baseline_mean" | bc)

        local recovery_label
        if [ "$(echo "$delta > 0.10" | bc)" -eq 1 ]; then
            recovery_label="STRONG"
            recovery_pattern+="strong "
        elif [ "$(echo "$delta > 0.03" | bc)" -eq 1 ]; then
            recovery_label="weak"
            recovery_pattern+="weak "
        else
            recovery_label="none"
            recovery_pattern+="none "
        fi

        printf "%-10s  %-10s  %-10s  %-10s\n" "$removed" "$mean" "$delta" "$recovery_label"
    done

    echo ""
    echo "=== REMOVAL INTERPRETATION ==="
    echo ""

    local strong_count weak_count none_count
    strong_count=$(echo "$recovery_pattern" | grep -o 'strong' | wc -l | tr -d ' ')
    weak_count=$(echo "$recovery_pattern" | grep -o 'weak' | wc -l | tr -d ' ')
    none_count=$(echo "$recovery_pattern" | grep -o 'none' | wc -l | tr -d ' ')
    local total=$((strong_count + weak_count + none_count))

    if [ "$total" -eq 0 ]; then
        echo "No removal data to analyze"
        return
    fi

    echo "Strong recovery: $strong_count/$total"
    echo "Weak recovery:   $weak_count/$total"
    echo "No recovery:     $none_count/$total"
    echo ""

    if [ "$strong_count" -gt "$((total / 2))" ]; then
        echo "MOST REMOVALS CAUSE STRONG RECOVERY"
        echo "Interpretation: CONSISTENT WITH threshold collapse"
        echo "  (removing ANY constraint from overloaded set helps all others)"
    elif [ "$strong_count" -le 2 ] && [ "$none_count" -gt "$((total / 2))" ]; then
        echo "MOST REMOVALS CAUSE NO RECOVERY — FEW CAUSE STRONG"
        echo "Interpretation: CONSISTENT WITH interference"
        echo "  (only specific bottleneck constraints matter)"
    else
        echo "UNIFORM WEAK RECOVERY"
        echo "Interpretation: CONSISTENT WITH resource competition"
        echo "  (each removal frees a small amount of attention for all others)"
    fi
}

# ============================================================
# VERDICT
# ============================================================

print_verdict() {
    echo ""
    echo "================================================"
    echo "  MECHANISM VERDICT (Phase $PHASE)"
    echo "================================================"
    echo ""
    echo "Decision criteria:"
    echo "  Resource competition: gradual curve + probabilistic + uniform recovery"
    echo "  Interference:         irregular curve + deterministic + selective recovery"
    echo "  Threshold collapse:   step function + deterministic + universal recovery"
    echo ""
    echo "To reach a final verdict, run all 3 phases and compare:"
    echo "  Phase 1 → curve shape (gradual vs step vs irregular)"
    echo "  Phase 2 → set dependence (yes = interference, no = resource/threshold)"
    echo "  Phase 3 → removal pattern (uniform = resource, selective = interference, universal = threshold)"
}

# ============================================================
# MAIN
# ============================================================

case "$PHASE" in
    1) analyze_phase1 ;;
    2) analyze_phase2 ;;
    3) analyze_phase3 ;;
    *) echo "Analyzing all phases found..."
       [ -d "$RESULTS_DIR/phase1" ] && analyze_phase1
       [ -d "$RESULTS_DIR/phase2" ] && analyze_phase2
       [ -d "$RESULTS_DIR/phase3" ] && analyze_phase3
       ;;
esac

print_verdict

echo ""
echo "Raw data: $RESULTS_DIR"
