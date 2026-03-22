#!/usr/bin/env bash
# Visualize coordination experiment results
#
# Usage:
#   ./visualize.sh                    # Show reference results
#   ./visualize.sh RESULTS_DIR        # Show results from a specific run

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# If a results dir is provided, analyze it; otherwise show reference data
if [ $# -gt 0 ] && [ -d "$1" ]; then
    RESULTS_DIR="$1"
    echo ""
    echo -e "${BOLD}Coordination Experiment Results${NC}"
    echo -e "${DIM}Source: $RESULTS_DIR${NC}"
    echo ""

    # Collect merge results from the run
    declare -A successes
    declare -A totals

    for condition_dir in "$RESULTS_DIR"/*/; do
        condition=$(basename "$condition_dir")
        [[ "$condition" =~ \. ]] && continue
        [ ! -d "$condition_dir" ] && continue

        s=0; t=0
        for task_dir in "$condition_dir"*/; do
            [ ! -d "$task_dir" ] && continue
            for trial_dir in "$task_dir"trial-*/; do
                [ ! -d "$trial_dir" ] && continue
                t=$((t + 1))
                if [ -f "$trial_dir/merge_result.csv" ]; then
                    result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
                    [ "$result" = "success" ] && s=$((s + 1))
                fi
            done
        done
        successes[$condition]=$s
        totals[$condition]=$t
    done

    # Display bar chart
    echo -e "${BOLD}Success Rate by Condition${NC}"
    echo ""

    for condition in no-coord context-share messaging gate placement; do
        t=${totals[$condition]:-0}
        [ "$t" -eq 0 ] && continue
        s=${successes[$condition]:-0}
        pct=$((s * 100 / t))

        # Build bar (50 chars wide)
        bar_len=$((pct / 2))
        bar=""
        for ((i=0; i<bar_len; i++)); do bar+="█"; done
        for ((i=bar_len; i<50; i++)); do bar+="░"; done

        if [ "$pct" -eq 100 ]; then
            color=$GREEN
        elif [ "$pct" -gt 0 ]; then
            color=$YELLOW
        else
            color=$RED
        fi

        printf "  %-15s %s${color}%s${NC} %3d%% (%d/%d)\n" \
            "$condition" "" "$bar" "$pct" "$s" "$t"
    done
else
    # Show reference results
    echo ""
    echo -e "${BOLD}═══════════════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  Coordination Experiment: Reference Results (100 trials)${NC}"
    echo -e "${BOLD}═══════════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  Model: claude-haiku-4-5  |  200 agent invocations  |  100% individual success"
    echo ""
    echo -e "  ${BOLD}Success Rate by Condition${NC}"
    echo ""
    echo -e "  ${DIM}Can agents merge their work without conflicts?${NC}"
    echo ""

    # no-coord
    bar_red=""
    for ((i=0; i<50; i++)); do bar_red+="░"; done
    printf "  %-15s ${RED}%s${NC}   ${RED}${BOLD}0%%${NC}  (0/20)\n" "no-coord" "$bar_red"
    echo -e "  ${DIM}                Agents work independently, no awareness of each other${NC}"
    echo ""

    # context-share
    printf "  %-15s ${RED}%s${NC}   ${RED}${BOLD}0%%${NC}  (0/20)\n" "context-share" "$bar_red"
    echo -e "  ${DIM}                Each agent given the other's full task description${NC}"
    echo ""

    # messaging
    printf "  %-15s ${RED}%s${NC}   ${RED}${BOLD}0%%${NC}  (0/20)\n" "messaging" "$bar_red"
    echo -e "  ${DIM}                Agents exchange plans via shared files before coding${NC}"
    echo ""

    # gate
    printf "  %-15s ${RED}%s${NC}   ${RED}${BOLD}0%%${NC}  (0/20)\n" "gate" "$bar_red"
    echo -e "  ${DIM}                Context sharing + mandatory post-implementation conflict check${NC}"
    echo ""

    # placement
    bar_green=""
    for ((i=0; i<50; i++)); do bar_green+="█"; done
    printf "  %-15s ${GREEN}%s${NC} ${GREEN}${BOLD}100%%${NC}  (20/20)\n" "placement" "$bar_green"
    echo -e "  ${DIM}                Explicit non-overlapping insertion points assigned by orchestrator${NC}"
    echo ""

    echo -e "  ${BOLD}───────────────────────────────────────────────────────────────────${NC}"
    echo ""
    echo -e "  ${CYAN}The only difference:${NC} placement tells agents ${BOLD}where${NC} to put their code."
    echo -e "  Every other condition tells agents ${BOLD}about${NC} each other — and it changes nothing."
    echo ""
    echo -e "  ${DIM}Individual agent scores: 200/200 scored 6/6 (100%). The agents aren't failing.${NC}"
    echo -e "  ${DIM}The coordination mechanism is failing.${NC}"
    echo ""

    # Messaging detail
    echo -e "  ${BOLD}Messaging Condition Detail${NC}"
    echo ""
    echo -e "  In 18/20 messaging trials, ${BOLD}both${NC} agents wrote coordination plans."
    echo -e "  Both agents reported: \"no conflicts expected.\""
    echo -e "  Both agents chose the same insertion point."
    echo -e "  ${RED}All 20 trials: CONFLICT.${NC}"
    echo ""
fi
