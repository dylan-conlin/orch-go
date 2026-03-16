#!/bin/bash
# gates/intent-check.sh — Layer 4 LLM intent alignment gate
#
# Judges whether a 3D model aligns with its declared design intent.
# Uses Claude CLI (haiku) for low-cost structured judgment.
#
# Usage:
#   ./gates/intent-check.sh <scad-file> <spec-file> [summary-file] [png-file]
#
# Exit codes:
#   0 = ALIGNED or gate unavailable (fail-open)
#   1 = MISALIGNED (only when --blocking flag used)
#   2 = NEEDS_REVIEW
#
# Environment:
#   INTENT_GATE_MODEL  - model to use (default: haiku)
#   INTENT_GATE_BLOCKING - set to "true" to block on MISALIGNED (default: advisory)
#   INTENT_GATE_VERBOSE  - set to "true" for full output

set -euo pipefail

SCAD_FILE="${1:?Usage: intent-check.sh <scad-file> <spec-file> [summary-file] [png-file]}"
SPEC_FILE="${2:?Usage: intent-check.sh <scad-file> <spec-file> [summary-file] [png-file]}"
SUMMARY_FILE="${3:-}"
PNG_FILE="${4:-}"

MODEL="${INTENT_GATE_MODEL:-haiku}"
BLOCKING="${INTENT_GATE_BLOCKING:-false}"
VERBOSE="${INTENT_GATE_VERBOSE:-false}"

# Validate inputs
if [ ! -f "$SCAD_FILE" ]; then
    echo "GATE ERROR: .scad file not found: $SCAD_FILE"
    exit 0  # Fail-open
fi
if [ ! -f "$SPEC_FILE" ]; then
    echo "GATE ERROR: spec file not found: $SPEC_FILE"
    exit 0  # Fail-open
fi

# Read inputs
SPEC_CONTENT=$(cat "$SPEC_FILE")
SCAD_CONTENT=$(cat "$SCAD_FILE")

SUMMARY_CONTENT="{}"
if [ -n "$SUMMARY_FILE" ] && [ -f "$SUMMARY_FILE" ]; then
    SUMMARY_CONTENT=$(cat "$SUMMARY_FILE")
fi

# Build gate prompt
GATE_PROMPT="You are a design review gate for 3D-printable OpenSCAD parts. Your job is to judge whether a 3D model aligns with its declared design intent.

DESIGN INTENT (what was requested):
---
$SPEC_CONTENT
---

OPENSCAD SOURCE (what was built):
---
$SCAD_CONTENT
---

GEOMETRY METADATA (render output):
---
$SUMMARY_CONTENT
---

Judge this design on 4 criteria:

1. DIMENSIONAL COMPLIANCE: Do the key dimensions in the source match the spec? Check variable assignments against spec requirements. Tolerance: ±5% unless spec states otherwise.

2. FEATURE COMPLETENESS: Are all features mentioned in the spec present in the source? Look for modules, operations, and geometric primitives that implement each specified feature.

3. CONSTRAINT SATISFACTION: Are material/printer constraints respected? Check wall thickness (>=0.8mm), overhang angles (<60°), minimum hole diameter (>=1mm), and any spec-specific constraints.

4. PURPOSE FIT: Would this design actually serve its declared purpose? Consider structural adequacy, dimensional appropriateness for the use case, and practical usability.

Respond in EXACTLY this format (no other text):
VERDICT: ALIGNED
CONFIDENCE: 0.85
CRITERIA:
  dimensional: PASS
  features: PASS
  constraints: PASS
  purpose: PASS
REASONING: One paragraph explaining the verdict.

Valid VERDICT values: ALIGNED, MISALIGNED, NEEDS_REVIEW
Valid criteria values: PASS, FAIL, PARTIAL
CONFIDENCE must be between 0.0 and 1.0"

echo "=== Layer 4: Intent Alignment Gate ==="

# Call LLM via Claude CLI
RESULT=$(echo "$GATE_PROMPT" | claude --print --model "$MODEL" 2>/dev/null) || {
    echo "GATE WARN: Intent gate unavailable (LLM API error)"
    echo "Proceeding without intent check (fail-open)"
    exit 0
}

# Parse verdict
VERDICT=$(echo "$RESULT" | grep "^VERDICT:" | head -1 | sed 's/VERDICT: *//')
CONFIDENCE=$(echo "$RESULT" | grep "^CONFIDENCE:" | head -1 | sed 's/CONFIDENCE: *//')

if [ -z "$VERDICT" ]; then
    echo "GATE WARN: Could not parse LLM verdict"
    if [ "$VERBOSE" = "true" ]; then
        echo "Raw output:"
        echo "$RESULT"
    fi
    exit 0  # Fail-open on parse failure
fi

echo "Verdict: $VERDICT (confidence: $CONFIDENCE)"

if [ "$VERBOSE" = "true" ]; then
    echo ""
    echo "$RESULT"
fi

# Emit structured output for measurement
echo "---GATE_OUTPUT_START---"
echo "$RESULT"
echo "---GATE_OUTPUT_END---"

case "$VERDICT" in
    ALIGNED)
        echo "=== Layer 4 PASS ==="
        exit 0
        ;;
    MISALIGNED)
        if [ "$BLOCKING" = "true" ]; then
            echo "=== Layer 4 FAIL (blocking mode) ==="
            exit 1
        else
            echo "=== Layer 4 WARN (advisory: MISALIGNED) ==="
            exit 0
        fi
        ;;
    NEEDS_REVIEW)
        echo "=== Layer 4 WARN (needs human review) ==="
        exit 2
        ;;
    *)
        echo "GATE WARN: Unknown verdict: $VERDICT"
        exit 0
        ;;
esac
