#!/usr/bin/env bash
# Coordination Demo — Accretion Measurement
#
# Measures lines added, lines removed, net delta, and file count per agent/model
# from git diffs captured during coordination demo trials.
#
# Works with both original (run.sh) and redesign (redesign/run.sh) result formats.
#
# Usage:
#   ./measure-accretion.sh RESULTS_DIR
#   ./measure-accretion.sh results/20260309-172034
#   ./measure-accretion.sh redesign/results/20260319-155033

set -euo pipefail

RESULTS_DIR="${1:?Usage: measure-accretion.sh RESULTS_DIR}"
ACCRETION_FILE="$RESULTS_DIR/accretion.csv"

# Detect format: redesign has condition/task_type/trial-N/agent-{a,b}/ structure
# Original has trial-N/{model}/ structure
if [ -d "$RESULTS_DIR/no-coord" ] || [ -d "$RESULTS_DIR/placement" ] || [ -d "$RESULTS_DIR/context-share" ] || [ -d "$RESULTS_DIR/messaging" ]; then
    FORMAT="redesign"
else
    FORMAT="original"
fi

echo "Format detected: $FORMAT"

# --- Diff parser ---
# Extracts lines added/removed. Prefers numstat.txt (machine-parseable) over full_diff.txt.
measure_dir() {
    local dir="$1"
    local numstat="$dir/numstat.txt"
    local diff_file="$dir/full_diff.txt"

    # Prefer numstat (precise, machine-parseable)
    if [ -f "$numstat" ] && [ -s "$numstat" ]; then
        local added=0
        local removed=0
        local files=0
        while IFS=$'\t' read -r a r f; do
            [ -z "$f" ] && continue
            # Skip binary files (shown as -)
            if [ "$a" != "-" ]; then
                added=$((added + a))
            fi
            if [ "$r" != "-" ]; then
                removed=$((removed + r))
            fi
            files=$((files + 1))
        done < "$numstat"
        local net=$((added - removed))
        echo "$added,$removed,$net,$files"
        return
    fi

    # Fallback: parse unified diff
    if [ ! -f "$diff_file" ] || [ ! -s "$diff_file" ]; then
        echo "0,0,0,0"
        return
    fi

    local added=0
    local removed=0
    local files=0

    files=$(grep -c "^diff --git" "$diff_file" || true)
    added=$(grep -c "^+[^+]" "$diff_file" || true)
    removed=$(grep -c "^-[^-]" "$diff_file" || true)
    files=${files:-0}
    added=${added:-0}
    removed=${removed:-0}

    local net=$((added - removed))
    echo "$added,$removed,$net,$files"
}

# --- Measure based on format ---

if [ "$FORMAT" = "original" ]; then
    echo "model,trial,lines_added,lines_removed,net_delta,files_changed,duration_s" > "$ACCRETION_FILE"

    echo ""
    echo "=== Accretion by Model ==="
    printf "%-6s %-6s %8s %8s %8s %5s %6s\n" "Model" "Trial" "Added" "Removed" "Net" "Files" "Dur(s)"

    for trial_dir in "$RESULTS_DIR"/trial-*/*/; do
        [ ! -d "$trial_dir" ] && continue
        model=$(basename "$trial_dir")
        trial=$(basename "$(dirname "$trial_dir")" | sed 's/trial-//')
        duration=$(cat "$trial_dir/duration_seconds" 2>/dev/null || echo "0")

        metrics=$(measure_dir "$trial_dir")
        IFS=',' read -r added removed net files <<< "$metrics"

        echo "$model,$trial,$added,$removed,$net,$files,$duration" >> "$ACCRETION_FILE"
        printf "%-6s %-6s %8d %8d %8d %5d %6d\n" "$model" "$trial" "$added" "$removed" "$net" "$files" "$duration"
    done

    # Summary per model
    echo ""
    echo "=== Model Summary ==="
    for model in haiku opus sonnet; do
        model_lines=$(grep "^$model," "$ACCRETION_FILE" 2>/dev/null || true)
        [ -z "$model_lines" ] && continue

        count=0; sum_added=0; sum_removed=0; sum_net=0; sum_files=0
        while IFS=',' read -r m t a r n f d; do
            sum_added=$((sum_added + a))
            sum_removed=$((sum_removed + r))
            sum_net=$((sum_net + n))
            sum_files=$((sum_files + f))
            count=$((count + 1))
        done <<< "$model_lines"

        if [ "$count" -gt 0 ]; then
            avg_added=$((sum_added / count))
            avg_removed=$((sum_removed / count))
            avg_net=$((sum_net / count))
            avg_files=$((sum_files / count))
            printf "  %-6s: N=%d avg_added=%d avg_removed=%d avg_net=%d avg_files=%d\n" \
                "$model" "$count" "$avg_added" "$avg_removed" "$avg_net" "$avg_files"
        fi
    done

else
    # Redesign format: condition/task_type/trial-N/agent-{a,b}/
    echo "condition,task_type,trial,agent,lines_added,lines_removed,net_delta,files_changed,duration_s" > "$ACCRETION_FILE"

    echo ""
    echo "=== Accretion by Condition ==="
    printf "%-14s %-8s %-6s %-6s %8s %8s %8s %5s %6s\n" \
        "Condition" "Task" "Trial" "Agent" "Added" "Removed" "Net" "Files" "Dur(s)"

    for condition_dir in "$RESULTS_DIR"/*/; do
        condition=$(basename "$condition_dir")
        [ ! -d "$condition_dir" ] && continue
        # Skip non-condition entries (metadata files, csv, md)
        [[ "$condition" =~ \. ]] && continue
        [[ "$condition" == "scores" ]] && continue

        for task_dir in "$condition_dir"*/; do
            task_type=$(basename "$task_dir")
            [ ! -d "$task_dir" ] && continue

            for trial_dir in "$task_dir"trial-*/; do
                [ ! -d "$trial_dir" ] && continue
                trial=$(basename "$trial_dir" | sed 's/trial-//')

                for agent_dir in "$trial_dir"agent-*/; do
                    [ ! -d "$agent_dir" ] && continue
                    agent=$(basename "$agent_dir" | sed 's/agent-//')
                    duration=$(cat "$agent_dir/duration_seconds" 2>/dev/null || echo "0")

                    metrics=$(measure_dir "$agent_dir")
                    IFS=',' read -r added removed net files <<< "$metrics"

                    echo "$condition,$task_type,$trial,$agent,$added,$removed,$net,$files,$duration" >> "$ACCRETION_FILE"
                    printf "%-14s %-8s %-6s %-6s %8d %8d %8d %5d %6d\n" \
                        "$condition" "$task_type" "$trial" "$agent" "$added" "$removed" "$net" "$files" "$duration"
                done
            done
        done
    done

    # Summary per condition
    echo ""
    echo "=== Condition Summary ==="
    for condition in no-coord placement context-share messaging; do
        cond_lines=$(grep "^$condition," "$ACCRETION_FILE" 2>/dev/null || true)
        [ -z "$cond_lines" ] && continue

        count=0; sum_added=0; sum_removed=0; sum_net=0; sum_files=0
        while IFS=',' read -r c tt t ag a r n f d; do
            sum_added=$((sum_added + a))
            sum_removed=$((sum_removed + r))
            sum_net=$((sum_net + n))
            sum_files=$((sum_files + f))
            count=$((count + 1))
        done <<< "$cond_lines"

        if [ "$count" -gt 0 ]; then
            avg_added=$((sum_added / count))
            avg_removed=$((sum_removed / count))
            avg_net=$((sum_net / count))
            printf "  %-14s: N=%d avg_added=%d avg_removed=%d avg_net=%d\n" \
                "$condition" "$count" "$avg_added" "$avg_removed" "$avg_net"
        fi
    done

    # Read model from metadata if available
    if [ -f "$RESULTS_DIR/metadata.json" ]; then
        model=$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['model'])" "$RESULTS_DIR/metadata.json" 2>/dev/null || echo "unknown")
        echo ""
        echo "Model: $model"
    fi
fi

echo ""
echo "Accretion data written to: $ACCRETION_FILE"
