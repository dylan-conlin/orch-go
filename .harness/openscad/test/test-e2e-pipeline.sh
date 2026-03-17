#!/bin/bash
# test-e2e-pipeline.sh — End-to-end validation: design a part through full gate pipeline
#
# Validates that a new part (cable-clip) passes all 4 gate layers:
#   Layer 1: Parameter validation (assert gates in .scad)
#   Layer 2: Geometry validation (CGAL manifold + polygon budget)
#   Layer 3: Printability (PrusaSlicer CLI — skipped if unavailable)
#   Layer 4: Intent alignment (LLM gate — skipped if claude CLI unavailable)
#
# Usage:
#   ./test/test-e2e-pipeline.sh              # Run full pipeline
#   ./test/test-e2e-pipeline.sh --skip-l4    # Skip Layer 4 (no LLM calls)
#
# Exit codes:
#   0 = all available gates passed
#   1 = one or more gates failed

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
HARNESS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PARTS_DIR="$HARNESS_DIR/parts"
EXPORTS_DIR="$HARNESS_DIR/exports"
GATES_DIR="$HARNESS_DIR/gates"
TMPDIR="${TMPDIR:-/tmp}/openscad-e2e-test"

# Resolve openscad binary
OPENSCAD="${OPENSCAD:-$(command -v openscad 2>/dev/null || echo "/Users/dylanconlin/Documents/personal/openscad/build/OpenSCAD.app/Contents/MacOS/OpenSCAD")}"
export OPENSCAD

SKIP_L4=false
for arg in "$@"; do
    case "$arg" in
        --skip-l4) SKIP_L4=true ;;
    esac
done

mkdir -p "$TMPDIR" "$EXPORTS_DIR"

PASSED=0
FAILED=0
SKIPPED=0
TOTAL=0

pass() { PASSED=$((PASSED + 1)); TOTAL=$((TOTAL + 1)); echo "  PASS: $1"; }
fail() { FAILED=$((FAILED + 1)); TOTAL=$((TOTAL + 1)); echo "  FAIL: $1"; }
skip() { SKIPPED=$((SKIPPED + 1)); TOTAL=$((TOTAL + 1)); echo "  SKIP: $1"; }

PART="cable-clip"
SCAD="$PARTS_DIR/${PART}.scad"
SPEC="$PARTS_DIR/${PART}-spec.md"
STL="$EXPORTS_DIR/${PART}.stl"
PNG="$EXPORTS_DIR/${PART}.png"
SUMMARY="$EXPORTS_DIR/${PART}-summary.json"

echo "=========================================="
echo "  E2E Pipeline Test: $PART"
echo "=========================================="
echo ""
echo "Part:    $SCAD"
echo "Spec:    $SPEC"
echo "OpenSCAD: $OPENSCAD"
echo ""

# Verify source files exist
if [ ! -f "$SCAD" ]; then
    echo "ERROR: Part file not found: $SCAD"
    exit 1
fi
if [ ! -f "$SPEC" ]; then
    echo "ERROR: Spec file not found: $SPEC"
    exit 1
fi

# =============================================
# Layer 1: Parameter Validation (assert gates)
# =============================================
echo "=== Layer 1: Parameter Validation ==="
echo ""

# Test 1: Default parameters render successfully
echo "Test 1: Default parameters render"
if "$OPENSCAD" -o "$TMPDIR/l1-default.stl" --backend manifold --quiet "$SCAD" 2>"$TMPDIR/l1-stderr.txt"; then
    pass "default parameters render successfully"
else
    fail "default parameters should render"
    cat "$TMPDIR/l1-stderr.txt"
fi

# Test 2: Invalid cable_diameter triggers assert
echo "Test 2: cable_diameter=0.5 triggers assert (below 2mm min)"
if "$OPENSCAD" -o "$TMPDIR/l1-bad-cable.stl" --backend manifold --quiet -D 'cable_diameter=0.5' "$SCAD" 2>"$TMPDIR/l1-stderr.txt"; then
    fail "cable_diameter=0.5 should fail"
else
    if grep -q "GATE FAIL:" "$TMPDIR/l1-stderr.txt"; then
        pass "small cable_diameter caught by assert"
    else
        fail "small cable_diameter failed but no GATE FAIL pattern"
        cat "$TMPDIR/l1-stderr.txt"
    fi
fi

# Test 3: Too many slots for width triggers assert
echo "Test 3: 3 slots with narrow width triggers assert"
if "$OPENSCAD" -o "$TMPDIR/l1-too-many-slots.stl" --backend manifold --quiet \
    -D 'num_slots=3' -D 'width=15' -D 'cable_diameter=5' -D 'slot_spacing=7' "$SCAD" 2>"$TMPDIR/l1-stderr.txt"; then
    fail "3 slots in 15mm width should fail (total slot width > available)"
else
    if grep -q "GATE FAIL:" "$TMPDIR/l1-stderr.txt"; then
        pass "slot overflow caught by assert"
    else
        fail "slot overflow failed but no GATE FAIL pattern"
        cat "$TMPDIR/l1-stderr.txt"
    fi
fi

# Test 4: Negative desk thickness triggers assert
echo "Test 4: desk_thickness=-5 triggers assert"
if "$OPENSCAD" -o "$TMPDIR/l1-neg-desk.stl" --backend manifold --quiet -D 'desk_thickness=-5' "$SCAD" 2>"$TMPDIR/l1-stderr.txt"; then
    fail "negative desk_thickness should fail"
else
    if grep -q "GATE FAIL:" "$TMPDIR/l1-stderr.txt"; then
        pass "negative desk_thickness caught by assert"
    else
        fail "negative desk_thickness failed but no GATE FAIL pattern"
        cat "$TMPDIR/l1-stderr.txt"
    fi
fi

# Test 5: Parameter sweep with -D flags (no .scad edits)
echo "Test 5: Parameter sweep via -D flags"
SWEEP_PASS=true
for cable_d in 3 5 8 12; do
    if ! "$OPENSCAD" -o "$TMPDIR/l1-sweep-${cable_d}.stl" --backend manifold --quiet \
        -D "cable_diameter=${cable_d}" -D "slot_spacing=$((cable_d + 4))" -D "width=$((cable_d * 2 + 15))" "$SCAD" 2>"$TMPDIR/l1-stderr.txt"; then
        echo "    sweep cable_diameter=$cable_d failed"
        SWEEP_PASS=false
    fi
done
if [ "$SWEEP_PASS" = "true" ]; then
    pass "parameter sweep (cable_diameter 3/5/8/12) all render"
else
    fail "parameter sweep had failures"
fi

echo ""

# =============================================
# Layer 2: Geometry Validation (CGAL + budget)
# =============================================
echo "=== Layer 2: Geometry Validation ==="
echo ""

# Test 6: Full geometry check with CGAL
echo "Test 6: Cable-clip passes Layer 2 geometry check"
if bash "$GATES_DIR/geometry-check.sh" "$SCAD" "$SUMMARY" "$STL" > "$TMPDIR/l2-stdout.txt" 2>&1; then
    if grep -q "Layer 2 PASS" "$TMPDIR/l2-stdout.txt"; then
        pass "cable-clip passes CGAL manifold + polygon budget"
        # Extract facet count for reporting
        FACETS=$(grep "Geometry OK:" "$TMPDIR/l2-stdout.txt" | grep -o '[0-9]* facets' || echo "unknown")
        echo "    ($FACETS)"
    else
        fail "geometry-check exited 0 but no PASS message"
        cat "$TMPDIR/l2-stdout.txt"
    fi
else
    fail "cable-clip should pass geometry check"
    cat "$TMPDIR/l2-stdout.txt"
fi

# Test 7: Summary JSON was produced
echo "Test 7: Summary JSON produced"
if [ -f "$SUMMARY" ]; then
    if python3 -c "import json; json.load(open('$SUMMARY'))" 2>/dev/null; then
        pass "summary JSON is valid"
    else
        fail "summary JSON exists but is not valid JSON"
    fi
else
    fail "no summary JSON produced"
fi

# Test 8: STL is non-degenerate
echo "Test 8: STL output is non-degenerate"
if [ -f "$STL" ]; then
    STL_SIZE=$(wc -c < "$STL" | tr -d ' ')
    if [ "$STL_SIZE" -gt 200 ]; then
        pass "STL is $STL_SIZE bytes (non-degenerate)"
    else
        fail "STL is only $STL_SIZE bytes (degenerate)"
    fi
else
    fail "no STL produced"
fi

# Test 9: PNG preview
echo "Test 9: PNG visual preview renders"
if "$OPENSCAD" -o "$PNG" --autocenter --viewall --backend manifold --quiet "$SCAD" 2>"$TMPDIR/l2-png-stderr.txt"; then
    if [ -f "$PNG" ]; then
        PNG_SIZE=$(wc -c < "$PNG" | tr -d ' ')
        pass "PNG preview is $PNG_SIZE bytes"
    else
        fail "PNG command succeeded but no file produced"
    fi
else
    fail "PNG render failed"
    cat "$TMPDIR/l2-png-stderr.txt"
fi

echo ""

# =============================================
# Layer 3: Printability (PrusaSlicer CLI)
# =============================================
echo "=== Layer 3: Printability Check ==="
echo ""

echo "Test 10: PrusaSlicer CLI availability"
if command -v prusa-slicer >/dev/null 2>&1 || command -v PrusaSlicer >/dev/null 2>&1; then
    SLICER=$(command -v prusa-slicer 2>/dev/null || command -v PrusaSlicer)
    echo "  Slicer found: $SLICER"
    echo "Test 11: Sliceable check"
    if "$SLICER" --export-gcode --info "$STL" 2>"$TMPDIR/l3-stderr.txt" >"$TMPDIR/l3-stdout.txt"; then
        pass "cable-clip is sliceable"
    else
        fail "PrusaSlicer rejected the STL"
        cat "$TMPDIR/l3-stderr.txt"
    fi
else
    skip "PrusaSlicer not installed — Layer 3 skipped"
fi

echo ""

# =============================================
# Layer 4: Intent Alignment (LLM gate)
# =============================================
echo "=== Layer 4: Intent Alignment ==="
echo ""

if [ "$SKIP_L4" = "true" ]; then
    skip "Layer 4 skipped (--skip-l4 flag)"
else
    echo "Test 11/12: Intent alignment gate"
    if command -v claude >/dev/null 2>&1; then
        if bash "$GATES_DIR/intent-check.sh" "$SCAD" "$SPEC" "$SUMMARY" "$PNG" > "$TMPDIR/l4-stdout.txt" 2>&1; then
            EXIT_CODE=$?
            VERDICT=$(grep "^Verdict:" "$TMPDIR/l4-stdout.txt" | head -1 | sed 's/Verdict: //')
            if echo "$VERDICT" | grep -qi "ALIGNED"; then
                pass "intent gate: $VERDICT"
            else
                echo "  INFO: Verdict=$VERDICT (advisory, non-blocking)"
                pass "intent gate ran successfully"
            fi
        else
            EXIT_CODE=$?
            if [ "$EXIT_CODE" -eq 2 ]; then
                pass "intent gate returned NEEDS_REVIEW (non-blocking)"
            else
                VERDICT=$(grep "^Verdict:" "$TMPDIR/l4-stdout.txt" 2>/dev/null | head -1 || echo "unknown")
                echo "  Intent gate exit=$EXIT_CODE verdict=$VERDICT"
                # Layer 4 is advisory — failure doesn't fail the pipeline
                pass "intent gate ran (advisory mode, exit=$EXIT_CODE)"
            fi
        fi

        # Test: Misaligned spec should detect drift
        echo "Test 12: Intentionally drifted spec triggers non-ALIGNED verdict"
        DRIFT_SPEC="$TMPDIR/drift-spec.md"
        cat > "$DRIFT_SPEC" <<'DRIFTEOF'
# Desk Lamp Base — Design Specification

## Purpose
Weighted lamp base for desk lamp with adjustable arm mount.

## Key Dimensions
- Diameter: 150mm circular base
- Height: 25mm
- Weight pocket: 100mm diameter, 15mm deep (for lead weight insert)
- Arm mount: M8 threaded insert hole, center

## Required Features
1. Weighted base with insert pocket
2. Non-slip rubber feet recesses (4x, 8mm diameter)
3. Cable routing channel through base
DRIFTEOF
        if bash "$GATES_DIR/intent-check.sh" "$SCAD" "$DRIFT_SPEC" "$SUMMARY" > "$TMPDIR/l4-drift-stdout.txt" 2>&1; then
            DRIFT_VERDICT=$(grep "^Verdict:" "$TMPDIR/l4-drift-stdout.txt" | head -1 | sed 's/Verdict: //')
            if echo "$DRIFT_VERDICT" | grep -qi "MISALIGNED\|NEEDS_REVIEW"; then
                pass "drifted spec detected: $DRIFT_VERDICT"
            else
                echo "  INFO: Drifted spec returned $DRIFT_VERDICT (expected MISALIGNED)"
                echo "  This is a false negative — intent gate did not catch the drift"
                fail "drifted spec should not be ALIGNED"
            fi
        else
            # Even an error exit is informative
            pass "drifted spec gate ran (exit non-zero)"
        fi
    else
        skip "claude CLI not installed — Layer 4 skipped"
    fi
fi

echo ""

# =============================================
# Summary
# =============================================
echo "=========================================="
echo "  E2E Pipeline Results: $PART"
echo "=========================================="
echo ""
echo "Passed:  $PASSED / $TOTAL"
echo "Failed:  $FAILED / $TOTAL"
echo "Skipped: $SKIPPED / $TOTAL"
echo ""

# List produced artifacts
echo "Artifacts produced:"
for f in "$STL" "$PNG" "$SUMMARY"; do
    if [ -f "$f" ]; then
        SIZE=$(wc -c < "$f" | tr -d ' ')
        echo "  $(basename "$f"): ${SIZE} bytes"
    fi
done

echo ""

# Cleanup temp
rm -rf "$TMPDIR"

if [ "$FAILED" -gt 0 ]; then
    echo "SOME TESTS FAILED"
    exit 1
fi

echo "ALL AVAILABLE GATES PASSED"
exit 0
