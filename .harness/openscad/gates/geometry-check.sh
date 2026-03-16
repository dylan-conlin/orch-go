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

# Step 1: Render with CGAL (manifold detection)
echo "=== Layer 2: CGAL Manifold Check ==="
CGAL_OUTPUT=$("$OPENSCAD" -o "$STL_OUTPUT" --backend cgal --quiet \
    --summary-file "$SUMMARY_FILE" "$SCAD_FILE" 2>&1) || {
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
    FACETS=$(python3 -c "import json; print(json.load(open('$SUMMARY_FILE')).get('facets', 0))" 2>/dev/null || echo "0")

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

echo "=== Layer 2 PASS ==="
exit 0
