#!/bin/bash
# Investigation Stance Contrastive Experiment
# 3 scenarios × 3 variants × 6 runs = 54 trials
# Tests whether investigation skill stance ("test before concluding") transfers

set -euo pipefail

BASE="evidence/2026-03-06-investigation-stance-contrastive"
SCENARIOS_DIR="$BASE/scenarios"
RESULTS_DIR="$BASE/results"
RUNS=6

echo "=== Investigation Stance Contrastive Experiment ==="
echo "54 trials: 3 scenarios × 3 variants × 6 runs"
echo ""

# Run bare variant
echo "[1/3] Running bare variant (N=$RUNS)..."
skillc test --scenarios "$SCENARIOS_DIR" --bare --runs "$RUNS" --json > "$RESULTS_DIR/bare.json" 2>/dev/null
echo "  Saved: $RESULTS_DIR/bare.json"

# Run without-stance variant
echo "[2/3] Running without-stance variant (N=$RUNS)..."
skillc test --scenarios "$SCENARIOS_DIR" --bare --runs "$RUNS" --json --variant-key without-stance > "$RESULTS_DIR/without-stance.json" 2>/dev/null
echo "  Saved: $RESULTS_DIR/without-stance.json"

# Run with-stance variant
echo "[3/3] Running with-stance variant (N=$RUNS)..."
skillc test --scenarios "$SCENARIOS_DIR" --bare --runs "$RUNS" --json --variant-key with-stance > "$RESULTS_DIR/with-stance.json" 2>/dev/null
echo "  Saved: $RESULTS_DIR/with-stance.json"

echo ""
echo "=== Complete: 54 trials ==="
echo "Results: $RESULTS_DIR/"
