#!/usr/bin/env bash
# Generate placement constraints from collision analysis.
#
# Reads collision_analysis.json and produces placement instruction text
# that can be appended to agent prompts.
#
# Strategy:
#   - Agent A keeps the gravitational insertion point (after the collision function)
#   - Agent B gets assigned a non-overlapping alternative (as far as possible from collision)
#
# Usage: ./generate_constraints.sh ANALYSIS_JSON SOURCE_FILE TEST_FILE
#   Outputs constraint files: constraint-a.md, constraint-b.md

set -euo pipefail

ANALYSIS_JSON="${1:?Usage: generate_constraints.sh ANALYSIS_JSON SOURCE_FILE TEST_FILE}"
SOURCE_FILE="${2:?Usage: generate_constraints.sh ANALYSIS_JSON SOURCE_FILE TEST_FILE}"
TEST_FILE="${3:?Usage: generate_constraints.sh ANALYSIS_JSON SOURCE_FILE TEST_FILE}"
OUTPUT_DIR=$(dirname "$ANALYSIS_JSON")

# Parse the gravitational function from analysis
GRAVITATIONAL=$(python3 -c "
import json, sys
with open('$ANALYSIS_JSON') as f:
    data = json.load(f)
print(data['gravitational_function'])
")

if [ -z "$GRAVITATIONAL" ]; then
    echo "ERROR: No gravitational function found in analysis"
    exit 1
fi

echo "=== Constraint Generation ==="
echo "Gravitational function: $GRAVITATIONAL"
echo "Source: $SOURCE_FILE"
echo "Test: $TEST_FILE"

# Find all function names in order
readarray -t SRC_FUNCTIONS < <(grep -n '^func ' "$SOURCE_FILE" | sed -E 's/^([0-9]+):func ([A-Za-z_][A-Za-z0-9_]*).*/\2/')
readarray -t TEST_FUNCTIONS < <(grep -n '^func ' "$TEST_FILE" | sed -E 's/^([0-9]+):func ([A-Za-z_][A-Za-z0-9_]*).*/\2/')

echo "Source functions: ${SRC_FUNCTIONS[*]}"
echo "Test functions: ${TEST_FUNCTIONS[*]}"

# Find the gravitational function's position and pick an alternative
grav_idx=-1
for i in "${!SRC_FUNCTIONS[@]}"; do
    if [ "${SRC_FUNCTIONS[$i]}" = "$GRAVITATIONAL" ]; then
        grav_idx=$i
        break
    fi
done

if [ "$grav_idx" -eq -1 ]; then
    echo "ERROR: Gravitational function '$GRAVITATIONAL' not found in source"
    exit 1
fi

# Pick the alternative insertion point: farthest function from gravitational
# Strategy: if gravitational is the last function, use the function just before FormatDuration
# (i.e., StripANSI). If it's somewhere else, use the opposite end.
total_funcs=${#SRC_FUNCTIONS[@]}
if [ "$grav_idx" -eq $((total_funcs - 1)) ]; then
    # Gravitational is last — pick the middle function as alternative
    alt_idx=$((total_funcs / 2))
else
    # Gravitational is not last — use last function
    alt_idx=$((total_funcs - 1))
fi

ALT_FUNC="${SRC_FUNCTIONS[$alt_idx]}"

# Find the function that comes AFTER the alternative (for "BEFORE X" instruction)
next_after_alt=""
if [ "$alt_idx" -lt $((total_funcs - 1)) ]; then
    next_after_alt="${SRC_FUNCTIONS[$((alt_idx + 1))]}"
fi

# Map to test functions
# Convention: TestFuncName maps to FuncName
GRAV_TEST="Test${GRAVITATIONAL}"
ALT_TEST="Test${ALT_FUNC}"

# Find the test function after the alternative test
next_after_alt_test=""
if [ -n "$next_after_alt" ]; then
    next_after_alt_test="Test${next_after_alt}"
fi

echo ""
echo "Agent A: insert after $GRAVITATIONAL (gravitational — keep original)"
echo "Agent B: insert after $ALT_FUNC (alternative)"
echo ""

# Generate constraint text for Agent A (keeps gravitational point)
CONSTRAINT_A="
## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`${GRAVITATIONAL}\` function in $(basename "$SOURCE_FILE").
Place your new test function(s) IMMEDIATELY after \`${GRAV_TEST}\` in $(basename "$TEST_FILE").

Do NOT place code anywhere else in these files."

# Generate constraint text for Agent B (redirected to alternative)
if [ -n "$next_after_alt" ]; then
    CONSTRAINT_B="
## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`${ALT_FUNC}\` function in $(basename "$SOURCE_FILE") (BEFORE \`${next_after_alt}\`).
Place your new test function(s) IMMEDIATELY after \`${ALT_TEST}\` in $(basename "$TEST_FILE") (BEFORE \`${next_after_alt_test}\`).

Do NOT place code anywhere else in these files."
else
    CONSTRAINT_B="
## IMPORTANT: Placement Instructions

Place your new function(s) IMMEDIATELY after the \`${ALT_FUNC}\` function in $(basename "$SOURCE_FILE").
Place your new test function(s) IMMEDIATELY after \`${ALT_TEST}\` in $(basename "$TEST_FILE").

Do NOT place code anywhere else in these files."
fi

# Write constraint files
echo "$CONSTRAINT_A" > "$OUTPUT_DIR/constraint-a.md"
echo "$CONSTRAINT_B" > "$OUTPUT_DIR/constraint-b.md"

# Write metadata about the constraint generation
cat > "$OUTPUT_DIR/constraint_metadata.json" << ENDJSON
{
  "gravitational_function": "$GRAVITATIONAL",
  "agent_a_placement": "after $GRAVITATIONAL",
  "agent_b_placement": "after $ALT_FUNC",
  "agent_b_before": "${next_after_alt:-end-of-file}",
  "source_file": "$SOURCE_FILE",
  "test_file": "$TEST_FILE",
  "generation_method": "automated from collision analysis",
  "human_intervention": false
}
ENDJSON

echo "Constraint A:"
echo "$CONSTRAINT_A"
echo ""
echo "Constraint B:"
echo "$CONSTRAINT_B"
echo ""
echo "Files written:"
echo "  $OUTPUT_DIR/constraint-a.md"
echo "  $OUTPUT_DIR/constraint-b.md"
echo "  $OUTPUT_DIR/constraint_metadata.json"
