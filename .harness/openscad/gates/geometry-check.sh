#!/bin/bash
# gates/geometry-check.sh — Layer 2 geometry validation
# Renders with CGAL backend, checks manifold status, validates polygon budget
set -euo pipefail

# Resolve openscad binary (may be alias or non-standard location)
OPENSCAD="${OPENSCAD:-$(command -v openscad 2>/dev/null || echo "/Users/dylanconlin/Documents/personal/openscad/build/OpenSCAD.app/Contents/MacOS/OpenSCAD")}"

SCAD_FILE="$1"
NAME=$(basename "$SCAD_FILE" .scad)
SUMMARY_FILE="${2:-exports/${NAME}-summary.json}"
STL_OUTPUT="${3:-exports/${NAME}.stl}"
MAX_FACETS="${MAX_FACETS:-200000}"
WARN_FACETS="${WARN_FACETS:-50000}"

# Pass-through -D flags from OPENSCAD_DFLAGS env var (space-separated)
DFLAGS=()
if [ -n "${OPENSCAD_DFLAGS:-}" ]; then
    read -ra DFLAGS <<< "$OPENSCAD_DFLAGS"
fi

# Step 1: Render with CGAL (manifold detection)
echo "=== Layer 2: CGAL Manifold Check ==="
echo "File: $SCAD_FILE"
CGAL_OUTPUT=$("$OPENSCAD" -o "$STL_OUTPUT" --backend cgal --quiet \
    --summary all --summary-file "$SUMMARY_FILE" ${DFLAGS[@]+"${DFLAGS[@]}"} "$SCAD_FILE" 2>&1) || {
    echo "GATE FAIL: CGAL render failed (exit $?)"
    echo "$CGAL_OUTPUT" | grep -i "error\|warning\|assert\|GATE FAIL" || true
    exit 1
}

# Step 2: Check for non-manifold warnings in CGAL output
if echo "$CGAL_OUTPUT" | grep -qi "not.*manifold\|Simple: no"; then
    echo "GATE FAIL: Non-manifold geometry detected"
    echo "$CGAL_OUTPUT"
    exit 1
fi

# Step 3: Check summary metrics
if [ -f "$SUMMARY_FILE" ]; then
    # Summary JSON nests under geometry.facets (OpenSCAD 2025+)
    FACETS=$(python3 -c "
import json
d = json.load(open('$SUMMARY_FILE'))
g = d.get('geometry', d)
print(g.get('facets', 0))
" 2>/dev/null || echo "0")

    if [ "$FACETS" -gt "$MAX_FACETS" ]; then
        echo "GATE FAIL: facet count $FACETS > $MAX_FACETS limit"
        exit 1
    fi
    if [ "$FACETS" -gt "$WARN_FACETS" ]; then
        echo "GATE WARN: facet count $FACETS > $WARN_FACETS warning threshold"
    fi

    echo "Geometry OK: $FACETS facets"
else
    echo "GATE WARN: No summary file produced — metrics unavailable"
fi

# Step 4: Zero-volume / degenerate STL check
if [ -f "$STL_OUTPUT" ]; then
    STL_SIZE=$(wc -c < "$STL_OUTPUT" | tr -d ' ')
    if [ "$STL_SIZE" -lt 200 ]; then
        echo "GATE FAIL: STL output is $STL_SIZE bytes (degenerate/empty geometry)"
        exit 1
    fi
fi

echo "=== Layer 2 PASS ==="
exit 0
