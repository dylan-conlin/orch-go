#!/bin/bash
# measure.sh — Run Layer 4 intent gate on the 100-pair measurement dataset
#
# Evaluates precision/recall of the LLM intent alignment gate against
# human-labeled ground truth.
#
# Usage:
#   ./test/measure.sh                    # Run full dataset
#   ./test/measure.sh --sample 10        # Run on 10 random pairs
#   ./test/measure.sh --drift-type subtle # Run on specific drift type
#   ./test/measure.sh --dry-run          # Show what would run, no LLM calls
#
# Output:
#   test/results/YYYY-MM-DD-HHMMSS/
#     results.jsonl    — per-pair results
#     summary.json     — aggregate metrics
#     confusion.txt    — confusion matrix

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
HARNESS_DIR="$(dirname "$SCRIPT_DIR")"
DATASET="$SCRIPT_DIR/dataset.jsonl"
GATE="$HARNESS_DIR/gates/intent-check.sh"

# Args
SAMPLE_SIZE=0
DRIFT_FILTER=""
DRY_RUN=false
MODEL="${INTENT_GATE_MODEL:-haiku}"

while [[ $# -gt 0 ]]; do
    case $1 in
        --sample) SAMPLE_SIZE="$2"; shift 2 ;;
        --drift-type) DRIFT_FILTER="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        --model) MODEL="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

# Create results directory
TIMESTAMP=$(date +%Y-%m-%d-%H%M%S)
RESULTS_DIR="$SCRIPT_DIR/results/$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

# Temp directory for per-pair files
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "=== Layer 4 Intent Gate Measurement ==="
echo "Dataset: $DATASET"
echo "Model: $MODEL"
echo "Results: $RESULTS_DIR"
echo ""

# Counters
TOTAL=0
TP=0  # True positive: gate says MISALIGNED, label is MISALIGNED
FP=0  # False positive: gate says MISALIGNED, label is ALIGNED
TN=0  # True negative: gate says ALIGNED, label is ALIGNED
FN=0  # False negative: gate says ALIGNED, label is MISALIGNED
ERRORS=0
NEEDS_REVIEW=0

# Per-drift-type counters
declare -A DRIFT_TP DRIFT_FP DRIFT_FN DRIFT_TN DRIFT_TOTAL

# Read dataset
PAIRS=()
while IFS= read -r line; do
    if [ -n "$DRIFT_FILTER" ]; then
        dtype=$(echo "$line" | python3 -c "import sys,json; print(json.load(sys.stdin).get('drift_type','none'))")
        if [ "$dtype" != "$DRIFT_FILTER" ] && [ "$DRIFT_FILTER" != "none" ]; then
            continue
        fi
        if [ "$DRIFT_FILTER" = "none" ] && [ "$dtype" != "none" ]; then
            continue
        fi
    fi
    PAIRS+=("$line")
done < "$DATASET"

# Sample if requested
if [ "$SAMPLE_SIZE" -gt 0 ] && [ "$SAMPLE_SIZE" -lt "${#PAIRS[@]}" ]; then
    # Shuffle and take first N
    SHUFFLED=()
    for i in $(shuf -i 0-$((${#PAIRS[@]}-1)) -n "$SAMPLE_SIZE"); do
        SHUFFLED+=("${PAIRS[$i]}")
    done
    PAIRS=("${SHUFFLED[@]}")
fi

echo "Evaluating ${#PAIRS[@]} pairs..."
echo ""

for pair_json in "${PAIRS[@]}"; do
    TOTAL=$((TOTAL + 1))

    # Extract fields
    PAIR_ID=$(echo "$pair_json" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
    LABEL=$(echo "$pair_json" | python3 -c "import sys,json; print(json.load(sys.stdin)['label'])")
    DRIFT_TYPE=$(echo "$pair_json" | python3 -c "import sys,json; print(json.load(sys.stdin).get('drift_type','none'))")

    # Write temp files for gate input
    echo "$pair_json" | python3 -c "import sys,json; print(json.load(sys.stdin)['spec'])" > "$TMPDIR/spec.md"
    echo "$pair_json" | python3 -c "import sys,json; print(json.load(sys.stdin)['scad_source'])" > "$TMPDIR/design.scad"
    echo "$pair_json" | python3 -c "import sys,json; print(json.load(sys.stdin)['geometry_metadata'])" > "$TMPDIR/summary.json"

    printf "[%3d/${#PAIRS[@]}] %-40s label=%-10s " "$TOTAL" "$PAIR_ID" "$LABEL"

    if [ "$DRY_RUN" = "true" ]; then
        echo "(dry run — skipped)"
        continue
    fi

    # Run intent gate
    GATE_OUTPUT=$(INTENT_GATE_MODEL="$MODEL" INTENT_GATE_VERBOSE=true \
        bash "$GATE" "$TMPDIR/design.scad" "$TMPDIR/spec.md" "$TMPDIR/summary.json" 2>&1) || true

    # Parse verdict from gate output
    VERDICT=$(echo "$GATE_OUTPUT" | grep "^VERDICT:" | head -1 | sed 's/VERDICT: *//' | tr -d '[:space:]')
    CONFIDENCE=$(echo "$GATE_OUTPUT" | grep "^CONFIDENCE:" | head -1 | sed 's/CONFIDENCE: *//' | tr -d '[:space:]')

    if [ -z "$VERDICT" ]; then
        VERDICT="ERROR"
        CONFIDENCE="0"
        ERRORS=$((ERRORS + 1))
    fi

    if [ "$VERDICT" = "NEEDS_REVIEW" ]; then
        NEEDS_REVIEW=$((NEEDS_REVIEW + 1))
    fi

    # Classify: gate positive = MISALIGNED, gate negative = ALIGNED
    GATE_POS=false
    if [ "$VERDICT" = "MISALIGNED" ]; then
        GATE_POS=true
    fi

    LABEL_POS=false
    if [ "$LABEL" = "MISALIGNED" ]; then
        LABEL_POS=true
    fi

    if [ "$GATE_POS" = "true" ] && [ "$LABEL_POS" = "true" ]; then
        TP=$((TP + 1)); CLASSIFICATION="TP"
        DRIFT_TP[$DRIFT_TYPE]=$(( ${DRIFT_TP[$DRIFT_TYPE]:-0} + 1 ))
    elif [ "$GATE_POS" = "true" ] && [ "$LABEL_POS" = "false" ]; then
        FP=$((FP + 1)); CLASSIFICATION="FP"
        DRIFT_FP[$DRIFT_TYPE]=$(( ${DRIFT_FP[$DRIFT_TYPE]:-0} + 1 ))
    elif [ "$GATE_POS" = "false" ] && [ "$LABEL_POS" = "false" ]; then
        TN=$((TN + 1)); CLASSIFICATION="TN"
        DRIFT_TN[$DRIFT_TYPE]=$(( ${DRIFT_TN[$DRIFT_TYPE]:-0} + 1 ))
    else
        FN=$((FN + 1)); CLASSIFICATION="FN"
        DRIFT_FN[$DRIFT_TYPE]=$(( ${DRIFT_FN[$DRIFT_TYPE]:-0} + 1 ))
    fi

    DRIFT_TOTAL[$DRIFT_TYPE]=$(( ${DRIFT_TOTAL[$DRIFT_TYPE]:-0} + 1 ))

    echo "verdict=$VERDICT conf=$CONFIDENCE → $CLASSIFICATION"

    # Write per-pair result
    echo "{\"id\":\"$PAIR_ID\",\"label\":\"$LABEL\",\"drift_type\":\"$DRIFT_TYPE\",\"verdict\":\"$VERDICT\",\"confidence\":$CONFIDENCE,\"classification\":\"$CLASSIFICATION\"}" >> "$RESULTS_DIR/results.jsonl"

    # Rate limit: avoid hammering the API
    sleep 1
done

if [ "$DRY_RUN" = "true" ]; then
    echo ""
    echo "Dry run complete. No LLM calls made."
    exit 0
fi

echo ""
echo "=== Results ==="
echo ""

# Compute metrics
PRECISION=0
RECALL=0
ACCURACY=0
F1=0

if [ $((TP + FP)) -gt 0 ]; then
    PRECISION=$(python3 -c "print(round($TP / ($TP + $FP), 4))")
fi
if [ $((TP + FN)) -gt 0 ]; then
    RECALL=$(python3 -c "print(round($TP / ($TP + $FN), 4))")
fi
if [ "$TOTAL" -gt 0 ]; then
    ACCURACY=$(python3 -c "print(round(($TP + $TN) / $TOTAL, 4))")
fi
if [ "$(python3 -c "print($PRECISION + $RECALL > 0)")" = "True" ]; then
    F1=$(python3 -c "p=$PRECISION; r=$RECALL; print(round(2*p*r/(p+r), 4) if p+r > 0 else 0)")
fi

echo "Confusion Matrix:"
echo "                  Predicted"
echo "              ALIGNED  MISALIGNED"
echo "Actual ALIGNED   TN=$TN     FP=$FP"
echo "   MISALIGNED    FN=$FN     TP=$TP"
echo ""
echo "Precision: $PRECISION (when gate says MISALIGNED, is it right?)"
echo "Recall:    $RECALL (does gate catch real misalignment?)"
echo "Accuracy:  $ACCURACY"
echo "F1 Score:  $F1"
echo ""
echo "Errors: $ERRORS"
echo "Needs Review: $NEEDS_REVIEW"
echo ""

# Threshold assessment
echo "=== Deployment Assessment ==="
if python3 -c "exit(0 if $PRECISION < 0.65 else 1)" 2>/dev/null; then
    echo "VERDICT: NOT DEPLOYABLE (precision < 65%)"
    echo "  Gate is noise — do not deploy."
elif python3 -c "exit(0 if $PRECISION < 0.80 else 1)" 2>/dev/null; then
    echo "VERDICT: ADVISORY ONLY (precision 65-80%)"
    echo "  Gate can warn but should not block."
else
    echo "VERDICT: ELIGIBLE FOR BLOCKING (precision >= 80%)"
    echo "  Gate precision sufficient for blocking mode."
fi
echo ""

# Per-drift-type breakdown
echo "=== Per-Drift-Type Recall ==="
for dtype in dimensional missing_features constraint_violation purpose_mismatch subtle none; do
    total=${DRIFT_TOTAL[$dtype]:-0}
    if [ "$total" -eq 0 ]; then continue; fi
    tp=${DRIFT_TP[$dtype]:-0}
    fn=${DRIFT_FN[$dtype]:-0}
    fp=${DRIFT_FP[$dtype]:-0}
    tn=${DRIFT_TN[$dtype]:-0}
    if [ $((tp + fn)) -gt 0 ]; then
        recall=$(python3 -c "print(round($tp / ($tp + $fn), 2))")
    else
        recall="N/A"
    fi
    printf "  %-25s n=%-3d TP=%-3d FN=%-3d recall=%s\n" "$dtype" "$total" "$tp" "$fn" "$recall"
done
echo ""

# Write confusion matrix
cat > "$RESULTS_DIR/confusion.txt" <<MATRIX
Layer 4 Intent Gate — Measurement Results
==========================================
Date: $(date +%Y-%m-%d)
Model: $MODEL
Dataset: 100 pairs (50 aligned, 50 misaligned)

Confusion Matrix:
                  Predicted
              ALIGNED  MISALIGNED
Actual ALIGNED   TN=$TN     FP=$FP
   MISALIGNED    FN=$FN     TP=$TP

Precision: $PRECISION
Recall:    $RECALL
Accuracy:  $ACCURACY
F1 Score:  $F1
Errors:    $ERRORS
Needs Review: $NEEDS_REVIEW
MATRIX

# Write summary JSON
python3 -c "
import json
summary = {
    'timestamp': '$(date -u +%Y-%m-%dT%H:%M:%SZ)',
    'model': '$MODEL',
    'total_pairs': $TOTAL,
    'confusion_matrix': {'TP': $TP, 'FP': $FP, 'TN': $TN, 'FN': $FN},
    'metrics': {
        'precision': $PRECISION,
        'recall': $RECALL,
        'accuracy': $ACCURACY,
        'f1': $F1,
    },
    'errors': $ERRORS,
    'needs_review': $NEEDS_REVIEW,
    'thresholds': {
        'blocking_eligible': $PRECISION >= 0.80,
        'advisory_eligible': $PRECISION >= 0.65,
    },
}
print(json.dumps(summary, indent=2))
" > "$RESULTS_DIR/summary.json"

echo "Results written to $RESULTS_DIR/"
echo "  results.jsonl  — per-pair results"
echo "  summary.json   — aggregate metrics"
echo "  confusion.txt  — confusion matrix"
