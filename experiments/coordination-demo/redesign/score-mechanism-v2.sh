#!/usr/bin/env bash
# Score and analyze mechanism discrimination v2 (tension-based) results
#
# Reads scores.json from each trial and produces:
#   Phase 1: Per-N resolution curve, tier-stratified degradation, sacrifice patterns
#   Phase 2: Tier comparison (HARD vs MEDIUM vs EASY)
#   Phase 3: Pair removal recovery analysis
#
# Key metrics:
#   - Resolution rate: % of pairs where agent picked a side (A-wins + B-wins + both) / total
#   - Sacrifice pattern: which side of each pair the agent consistently chooses
#   - Tier degradation: do HARD/MEDIUM/EASY pairs degrade differently?
#   - Both-satisfied rate: how often does agent find creative workaround?
#   - Neither rate: how often does agent drop both constraints?
#
# Usage:
#   ./score-mechanism-v2.sh results/mechanism-v2-p1-TIMESTAMP
#   ./score-mechanism-v2.sh results/mechanism-v2-p2-TIMESTAMP

set -euo pipefail

RESULTS_DIR="${1:?Usage: $0 <results-dir>}"

if [ ! -d "$RESULTS_DIR" ]; then
    echo "ERROR: Results directory not found: $RESULTS_DIR"
    exit 1
fi

PHASE=$(jq -r '.phase' "$RESULTS_DIR/metadata.json" 2>/dev/null || echo "1")

echo "================================================"
echo "  MECHANISM v2 ANALYSIS — TENSION-BASED (Phase $PHASE)"
echo "================================================"
echo "Results: $RESULTS_DIR"
echo ""

# ============================================================
# PHASE 1 ANALYSIS: Tension Scaling Curve
# ============================================================

analyze_phase1() {
    echo "=== RESOLUTION CURVE (by N pairs) ==="
    echo ""
    echo "For each N (pair count), how does the agent resolve tension?"
    echo ""
    printf "%-6s  %-6s  %-8s  %-8s  %-8s  %-8s  %-8s  %-8s\n" \
        "N" "Trials" "Resolve" "A-wins" "B-wins" "Both" "Neither" "TaskOK"
    printf "%-6s  %-6s  %-8s  %-8s  %-8s  %-8s  %-8s  %-8s\n" \
        "-----" "------" "-------" "------" "------" "------" "-------" "------"

    local all_n_data=""

    for n_dir in "$RESULTS_DIR"/phase1/n*/; do
        [ -d "$n_dir" ] || continue
        local n
        n=$(basename "$n_dir" | sed 's/n//')

        local trial_count=0
        local sum_resolve=0 sum_a=0 sum_b=0 sum_both=0 sum_neither=0
        local sum_task=0
        local resolve_rates=""

        for trial_dir in "$n_dir"/trial-*/; do
            [ -f "$trial_dir/scores.json" ] || continue
            local dry_run
            dry_run=$(jq -r '.dry_run // false' "$trial_dir/scores.json" 2>/dev/null)
            [ "$dry_run" = "true" ] && continue

            local version
            version=$(jq -r '.version // 1' "$trial_dir/scores.json" 2>/dev/null)
            [ "$version" != "2" ] && continue

            trial_count=$((trial_count + 1))

            local total_pairs a_wins b_wins both neither task_ok resolve_rate
            total_pairs=$(jq -r '.total_pairs' "$trial_dir/scores.json")
            a_wins=$(jq -r '.total_a_wins' "$trial_dir/scores.json")
            b_wins=$(jq -r '.total_b_wins' "$trial_dir/scores.json")
            both=$(jq -r '.total_both' "$trial_dir/scores.json")
            neither=$(jq -r '.total_neither' "$trial_dir/scores.json")
            task_ok=$(jq -r '.task_complete' "$trial_dir/scores.json")
            resolve_rate=$(jq -r '.resolve_rate' "$trial_dir/scores.json")

            sum_a=$((sum_a + a_wins))
            sum_b=$((sum_b + b_wins))
            sum_both=$((sum_both + both))
            sum_neither=$((sum_neither + neither))
            sum_task=$((sum_task + task_ok))
            resolve_rates+="$resolve_rate "
        done

        if [ "$trial_count" -eq 0 ]; then
            printf "%-6s  %-6s  %s\n" "$n" "0" "no data"
            continue
        fi

        # Calculate mean resolve rate
        local mean_resolve
        mean_resolve=$(echo "$resolve_rates" | tr ' ' '\n' | awk '
            NF { sum += $1; count++ }
            END { if (count > 0) printf "%.3f", sum/count; else print "0.000" }
        ')

        local total_pairs_all=$((sum_a + sum_b + sum_both + sum_neither))
        local mean_a mean_b mean_both mean_neither
        if [ "$total_pairs_all" -gt 0 ]; then
            mean_a=$(echo "scale=3; $sum_a / $total_pairs_all" | bc)
            mean_b=$(echo "scale=3; $sum_b / $total_pairs_all" | bc)
            mean_both=$(echo "scale=3; $sum_both / $total_pairs_all" | bc)
            mean_neither=$(echo "scale=3; $sum_neither / $total_pairs_all" | bc)
        else
            mean_a="0.000"; mean_b="0.000"; mean_both="0.000"; mean_neither="0.000"
        fi

        printf "%-6s  %-6s  %-8s  %-8s  %-8s  %-8s  %-8s  %-8s\n" \
            "$n" "$trial_count" "$mean_resolve" "$mean_a" "$mean_b" "$mean_both" "$mean_neither" "$sum_task/$trial_count"

        all_n_data+="$n $mean_resolve $mean_both $mean_neither
"
    done

    # Curve shape analysis
    echo ""
    echo "=== CURVE SHAPE ANALYSIS ==="
    echo ""
    echo "$all_n_data" | awk '
    NF >= 4 {
        n[NR] = $1; resolve[NR] = $2; both[NR] = $3; neither[NR] = $4
        count = NR
    }
    END {
        if (count < 3) { print "Insufficient data points for curve analysis"; exit }

        # Check for degradation
        first_resolve = resolve[1]; last_resolve = resolve[count]
        delta = first_resolve - last_resolve

        if (delta < 0.05) {
            print "Curve shape: FLAT (no degradation)"
            print "  Resolution rate stable from N=" n[1] " to N=" n[count]
            print "  Delta: " delta
            print ""
            print "INTERPRETATION: If flat, tension constraints are also being trivially resolved."
            print "  Check per-tier analysis below for differential effects."
        } else if (delta > 0.3) {
            # Check if step or gradual
            max_step = 0; step_n = ""
            for (i = 2; i <= count; i++) {
                step = resolve[i-1] - resolve[i]
                if (step > max_step) { max_step = step; step_n = n[i] }
            }
            if (max_step > delta * 0.6) {
                print "Curve shape: STEP FUNCTION (sharp cutoff at N=" step_n ")"
                print "  -> Consistent with THRESHOLD COLLAPSE mechanism"
            } else {
                print "Curve shape: GRADUAL DECLINE"
                print "  -> Consistent with RESOURCE COMPETITION mechanism"
            }
        } else {
            print "Curve shape: MILD DECLINE (delta=" delta ")"
            print "  -> Weak signal; may need more trials or stronger constraints"
        }

        # Check neither-rate scaling
        first_neither = neither[1]; last_neither = neither[count]
        neither_delta = last_neither - first_neither
        if (neither_delta > 0.15) {
            print ""
            print "Neither-rate increases from " first_neither " to " last_neither
            print "  -> Agents dropping more constraints under load"
        }
    }'

    # Per-pair survival analysis
    echo ""
    echo ""
    echo "=== PER-PAIR SURVIVAL (across all N values) ==="
    echo ""
    printf "%-5s  %-24s  %-6s  %-8s  %-8s  %-8s  %-8s  %-12s\n" \
        "Pair" "Name" "Tier" "A-wins" "B-wins" "Both" "Neither" "A-preference"
    printf "%-5s  %-24s  %-6s  %-8s  %-8s  %-8s  %-8s  %-12s\n" \
        "----" "----" "----" "------" "------" "------" "-------" "-----------"

    # Aggregate per-pair data across all trials
    for pair in P01 P02 P03 P04 P05 P06 P07 P08 P09 P10 P11 P12; do
        local p_a=0 p_b=0 p_both=0 p_neither=0 p_total=0

        for trial_dir in "$RESULTS_DIR"/phase1/n*/trial-*/; do
            [ -f "$trial_dir/scores.json" ] || continue
            local dry_run
            dry_run=$(jq -r '.dry_run // false' "$trial_dir/scores.json" 2>/dev/null)
            [ "$dry_run" = "true" ] && continue

            # Check if this pair was in this trial
            local resolution
            resolution=$(jq -r ".per_pair.${pair}.resolution // \"absent\"" "$trial_dir/scores.json" 2>/dev/null)
            [ "$resolution" = "absent" ] && continue

            p_total=$((p_total + 1))
            case "$resolution" in
                a_wins)  p_a=$((p_a + 1)) ;;
                b_wins)  p_b=$((p_b + 1)) ;;
                both)    p_both=$((p_both + 1)) ;;
                neither) p_neither=$((p_neither + 1)) ;;
            esac
        done

        if [ "$p_total" -eq 0 ]; then
            continue
        fi

        # Calculate A-preference (how consistently agent picks A over B)
        local a_pref
        if [ $((p_a + p_b)) -gt 0 ]; then
            a_pref=$(echo "scale=2; $p_a / ($p_a + $p_b)" | bc)
        else
            a_pref="N/A"
        fi

        local tier name
        tier=$(jq -r ".tiers | to_entries[] | select(.value[] == \"$pair\") | .key" "$RESULTS_DIR/metadata.json" 2>/dev/null | head -1)
        # Fallback tier lookup
        case "$pair" in
            P0[1-4]) tier="HARD" ;;
            P0[5-8]) tier="MED" ;;
            P09|P10|P11|P12) tier="EASY" ;;
        esac

        local rate_a=$(echo "scale=2; $p_a * 100 / $p_total" | bc)
        local rate_b=$(echo "scale=2; $p_b * 100 / $p_total" | bc)
        local rate_both=$(echo "scale=2; $p_both * 100 / $p_total" | bc)
        local rate_neither=$(echo "scale=2; $p_neither * 100 / $p_total" | bc)

        printf "%-5s  %-24s  %-6s  %5s%%  %5s%%  %5s%%  %5s%%    %-12s\n" \
            "$pair" "" "$tier" "$rate_a" "$rate_b" "$rate_both" "$rate_neither" "$a_pref"
    done

    # Tier-stratified analysis
    echo ""
    echo ""
    echo "=== TIER-STRATIFIED DEGRADATION ==="
    echo ""
    echo "Key discrimination: if EASY degrades like HARD -> resource competition"
    echo "                    if EASY holds while HARD degrades -> interference"
    echo ""

    for tier in HARD MED EASY; do
        echo "--- $tier tier ---"
        printf "%-6s  %-8s  %-8s  %-8s\n" "N" "Resolve" "Both%" "Neither%"

        for n_dir in "$RESULTS_DIR"/phase1/n*/; do
            [ -d "$n_dir" ] || continue
            local n
            n=$(basename "$n_dir" | sed 's/n//')

            local t_resolved=0 t_both=0 t_neither=0 t_total=0

            for trial_dir in "$n_dir"/trial-*/; do
                [ -f "$trial_dir/scores.json" ] || continue
                local dry_run
                dry_run=$(jq -r '.dry_run // false' "$trial_dir/scores.json" 2>/dev/null)
                [ "$dry_run" = "true" ] && continue

                # Get all pairs of this tier
                local pairs_in_tier
                pairs_in_tier=$(jq -r ".per_pair | to_entries[] | select(.value.tier == \"${tier}\") | .key" "$trial_dir/scores.json" 2>/dev/null)
                [ -z "$pairs_in_tier" ] && continue

                while IFS= read -r pid; do
                    [ -z "$pid" ] && continue
                    t_total=$((t_total + 1))
                    local res
                    res=$(jq -r ".per_pair.${pid}.resolution" "$trial_dir/scores.json" 2>/dev/null)
                    case "$res" in
                        a_wins|b_wins) t_resolved=$((t_resolved + 1)) ;;
                        both) t_resolved=$((t_resolved + 1)); t_both=$((t_both + 1)) ;;
                        neither) t_neither=$((t_neither + 1)) ;;
                    esac
                done <<< "$pairs_in_tier"
            done

            if [ "$t_total" -gt 0 ]; then
                local r_rate=$(echo "scale=3; $t_resolved / $t_total" | bc)
                local b_rate=$(echo "scale=3; $t_both / $t_total" | bc)
                local n_rate=$(echo "scale=3; $t_neither / $t_total" | bc)
                printf "%-6s  %-8s  %-8s  %-8s\n" "$n" "$r_rate" "$b_rate" "$n_rate"
            fi
        done
        echo ""
    done

    # Mechanism discrimination summary
    echo ""
    echo "=== MECHANISM DISCRIMINATION ==="
    echo ""
    echo "Compare tier curves above to distinguish mechanisms:"
    echo ""
    echo "  RESOURCE COMPETITION:"
    echo "    - All tiers degrade uniformly with N"
    echo "    - EASY pairs (normally trivially satisfiable) also degrade"
    echo "    - Neither-rate increases across all tiers"
    echo ""
    echo "  INTERFERENCE:"
    echo "    - HARD pairs degrade early, MEDIUM later, EASY holds"
    echo "    - Some specific pairs show consistent sacrifice patterns"
    echo "    - EASY 'both-satisfied' rate stays high"
    echo ""
    echo "  THRESHOLD COLLAPSE:"
    echo "    - All tiers show sharp cutoff at same N"
    echo "    - Binary: near-100% then near-0%"
    echo "    - No gradual degradation in any tier"
}

# ============================================================
# PHASE 2 ANALYSIS: Tier Comparison
# ============================================================

analyze_phase2() {
    echo "=== TIER COMPARISON (at critical N) ==="
    echo ""
    printf "%-8s  %-6s  %-8s  %-8s  %-8s  %-8s  %-8s\n" \
        "Set" "Trials" "Resolve" "A-wins%" "B-wins%" "Both%" "Neither%"
    printf "%-8s  %-6s  %-8s  %-8s  %-8s  %-8s  %-8s\n" \
        "------" "------" "-------" "-------" "-------" "------" "--------"

    for set_dir in "$RESULTS_DIR"/phase2/*/; do
        [ -d "$set_dir" ] || continue
        local set_name
        set_name=$(basename "$set_dir")

        local trial_count=0
        local sum_a=0 sum_b=0 sum_both=0 sum_neither=0

        for trial_dir in "$set_dir"/trial-*/; do
            [ -f "$trial_dir/scores.json" ] || continue
            local dry_run
            dry_run=$(jq -r '.dry_run // false' "$trial_dir/scores.json" 2>/dev/null)
            [ "$dry_run" = "true" ] && continue

            trial_count=$((trial_count + 1))
            sum_a=$((sum_a + $(jq -r '.total_a_wins' "$trial_dir/scores.json")))
            sum_b=$((sum_b + $(jq -r '.total_b_wins' "$trial_dir/scores.json")))
            sum_both=$((sum_both + $(jq -r '.total_both' "$trial_dir/scores.json")))
            sum_neither=$((sum_neither + $(jq -r '.total_neither' "$trial_dir/scores.json")))
        done

        if [ "$trial_count" -eq 0 ]; then continue; fi

        local total=$((sum_a + sum_b + sum_both + sum_neither))
        local resolved=$((sum_a + sum_b + sum_both))
        if [ "$total" -gt 0 ]; then
            printf "%-8s  %-6s  %-8s  %-8s  %-8s  %-8s  %-8s\n" \
                "$set_name" "$trial_count" \
                "$(echo "scale=3; $resolved / $total" | bc)" \
                "$(echo "scale=1; $sum_a * 100 / $total" | bc)%" \
                "$(echo "scale=1; $sum_b * 100 / $total" | bc)%" \
                "$(echo "scale=1; $sum_both * 100 / $total" | bc)%" \
                "$(echo "scale=1; $sum_neither * 100 / $total" | bc)%"
        fi
    done

    echo ""
    echo "Interpretation:"
    echo "  - If HARD set has much lower resolve rate than EASY -> interference mechanism"
    echo "  - If all sets degrade equally -> resource competition mechanism"
    echo "  - If MIXED set degrades more than pure-tier sets -> cross-tier interference"
}

# ============================================================
# PHASE 3 ANALYSIS: Pair Removal Recovery
# ============================================================

analyze_phase3() {
    echo "=== PAIR REMOVAL RECOVERY ==="
    echo ""
    printf "%-12s  %-6s  %-8s  %-8s  %-8s  %-8s\n" \
        "Removed" "Tier" "Resolve" "Both%" "Neither%" "Delta"
    printf "%-12s  %-6s  %-8s  %-8s  %-8s  %-8s\n" \
        "----------" "----" "-------" "------" "--------" "-----"

    # Get baseline resolve rate
    local baseline_resolve="0"
    local baseline_dir="$RESULTS_DIR/phase3/baseline"
    if [ -d "$baseline_dir" ]; then
        local bl_resolved=0 bl_total=0
        for trial_dir in "$baseline_dir"/trial-*/; do
            [ -f "$trial_dir/scores.json" ] || continue
            local dry_run
            dry_run=$(jq -r '.dry_run // false' "$trial_dir/scores.json" 2>/dev/null)
            [ "$dry_run" = "true" ] && continue

            bl_resolved=$((bl_resolved + $(jq -r '.resolved' "$trial_dir/scores.json")))
            bl_total=$((bl_total + $(jq -r '.total_pairs' "$trial_dir/scores.json")))
        done
        if [ "$bl_total" -gt 0 ]; then
            baseline_resolve=$(echo "scale=3; $bl_resolved / $bl_total" | bc)
        fi
        printf "%-12s  %-6s  %-8s  %-8s  %-8s  %-8s\n" \
            "baseline" "---" "$baseline_resolve" "---" "---" "---"
    fi

    for remove_dir in "$RESULTS_DIR"/phase3/remove-*/; do
        [ -d "$remove_dir" ] || continue
        local removed
        removed=$(basename "$remove_dir" | sed 's/remove-//')

        local trial_count=0
        local sum_resolved=0 sum_total=0 sum_both=0 sum_neither=0

        for trial_dir in "$remove_dir"/trial-*/; do
            [ -f "$trial_dir/scores.json" ] || continue
            local dry_run
            dry_run=$(jq -r '.dry_run // false' "$trial_dir/scores.json" 2>/dev/null)
            [ "$dry_run" = "true" ] && continue

            trial_count=$((trial_count + 1))
            sum_resolved=$((sum_resolved + $(jq -r '.resolved' "$trial_dir/scores.json")))
            sum_total=$((sum_total + $(jq -r '.total_pairs' "$trial_dir/scores.json")))
            sum_both=$((sum_both + $(jq -r '.total_both' "$trial_dir/scores.json")))
            sum_neither=$((sum_neither + $(jq -r '.total_neither' "$trial_dir/scores.json")))
        done

        if [ "$trial_count" -eq 0 ] || [ "$sum_total" -eq 0 ]; then continue; fi

        local resolve_rate=$(echo "scale=3; $sum_resolved / $sum_total" | bc)
        local both_pct=$(echo "scale=1; $sum_both * 100 / $sum_total" | bc)
        local neither_pct=$(echo "scale=1; $sum_neither * 100 / $sum_total" | bc)
        local delta=$(echo "scale=3; $resolve_rate - $baseline_resolve" | bc)

        local tier
        case "$removed" in
            P0[1-4]) tier="HARD" ;;
            P0[5-8]) tier="MED" ;;
            P09|P10|P11|P12) tier="EASY" ;;
            *) tier="?" ;;
        esac

        printf "%-12s  %-6s  %-8s  %5s%%  %6s%%  %-8s\n" \
            "-$removed" "$tier" "$resolve_rate" "$both_pct" "$neither_pct" "$delta"
    done

    echo ""
    echo "Interpretation:"
    echo "  - If removing a HARD pair helps ALL remaining pairs -> that pair was an interference source"
    echo "  - If removing ANY pair helps equally -> resource competition (shared budget)"
    echo "  - If no removal helps -> pairs don't interact (independent degradation)"
}

# ============================================================
# MAIN
# ============================================================

case "$PHASE" in
    1) analyze_phase1 ;;
    2) analyze_phase2 ;;
    3) analyze_phase3 ;;
    *) echo "Unknown phase: $PHASE" ;;
esac

echo ""
echo "================================================"
echo "  ANALYSIS COMPLETE"
echo "================================================"
