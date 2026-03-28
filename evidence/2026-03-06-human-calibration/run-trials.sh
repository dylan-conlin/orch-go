#!/bin/bash
# Human calibration experiment: 24 trials
# 4 scenarios × 3 variants × 2 runs

set -euo pipefail

# Prevent orch Stop hook from contaminating --print output when run
# from within a spawned worker session. See: orch-go-ssbwv
export ORCH_PRINT_MODE=1

BASE="evidence/2026-03-06-human-calibration"
TRANSCRIPTS="$BASE/transcripts"
PROMPTS="$BASE/prompts"
VARIANTS="$BASE/variants"

# Common claude flags
CLAUDE_FLAGS="--print --no-session-persistence --output-format text --dangerously-skip-permissions --disable-slash-commands --model sonnet --tools \"\""

run_trial() {
    local scenario=$1
    local variant=$2
    local run_num=$3
    local prompt_file="$PROMPTS/${scenario}.txt"
    local outdir="$TRANSCRIPTS/${scenario}-${variant}"
    local outfile="$outdir/run${run_num}.txt"

    mkdir -p "$outdir"

    echo "  Running ${scenario}-${variant} run ${run_num}..."

    if [ "$variant" = "bare" ]; then
        claude --print --no-session-persistence --output-format text \
            --dangerously-skip-permissions --disable-slash-commands \
            --model sonnet --tools "" \
            -- "$(cat "$prompt_file")" > "$outfile" 2>/dev/null
    else
        local variant_file="$VARIANTS/${scenario}-${variant}.md"
        claude --print --no-session-persistence --output-format text \
            --dangerously-skip-permissions --disable-slash-commands \
            --model sonnet --system-prompt "$(cat "$variant_file")" \
            --tools "" \
            -- "$(cat "$prompt_file")" > "$outfile" 2>/dev/null
    fi

    echo "  Done: $outfile ($(wc -c < "$outfile") bytes)"
}

echo "=== Human Calibration Experiment ==="
echo "24 trials: 4 scenarios × 3 variants × 2 runs"
echo ""

# S09: contradiction-detection (bare, with-stance, with-stance-and-action)
echo "[S09] contradiction-detection"
for run in 1 2; do
    run_trial s09 bare $run
    run_trial s09 with-stance $run
    run_trial s09 with-stance-and-action $run
done

# S11: absence-as-evidence (bare, without-stance, with-stance)
echo "[S11] absence-as-evidence"
for run in 1 2; do
    run_trial s11 bare $run
    run_trial s11 without-stance $run
    run_trial s11 with-stance $run
done

# S12: downstream-consumer-contract (bare, without-stance, with-stance)
echo "[S12] downstream-consumer-contract"
for run in 1 2; do
    run_trial s12 bare $run
    run_trial s12 without-stance $run
    run_trial s12 with-stance $run
done

# S13: stale-deprecation-claim (bare, without-stance, with-stance)
echo "[S13] stale-deprecation-claim"
for run in 1 2; do
    run_trial s13 bare $run
    run_trial s13 without-stance $run
    run_trial s13 with-stance $run
done

echo ""
echo "=== Complete: 24 transcripts saved ==="
echo "Transcripts at: $TRANSCRIPTS/"
