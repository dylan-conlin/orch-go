#!/bin/bash
# Tests for printability-check.sh
# Requires: openscad, PrusaSlicer
set -uo pipefail
# Note: no set -e — we need to capture non-zero exit codes

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GATE="$SCRIPT_DIR/printability-check.sh"
TEST_DIR="/tmp/printability-gate-test-$$"
mkdir -p "$TEST_DIR"
PASS=0
FAIL=0

# OpenSCAD may be a zsh alias; resolve to full path
OPENSCAD="${OPENSCAD_BIN:-}"
if [ -z "$OPENSCAD" ]; then
    if command -v openscad &>/dev/null; then
        OPENSCAD="openscad"
    elif [ -x "/Users/dylanconlin/Documents/personal/openscad/build/OpenSCAD.app/Contents/MacOS/OpenSCAD" ]; then
        OPENSCAD="/Users/dylanconlin/Documents/personal/openscad/build/OpenSCAD.app/Contents/MacOS/OpenSCAD"
    elif [ -x "/Applications/OpenSCAD.app/Contents/MacOS/OpenSCAD" ]; then
        OPENSCAD="/Applications/OpenSCAD.app/Contents/MacOS/OpenSCAD"
    else
        echo "SKIP: OpenSCAD not found (set OPENSCAD_BIN)"
        exit 0
    fi
fi

cleanup() { rm -rf "$TEST_DIR"; }
trap cleanup EXIT

# Run gate and capture both output and exit code
run_gate() {
    local output exit_code
    output=$("$GATE" "$@" 2>&1)
    exit_code=$?
    LAST_OUTPUT="$output"
    LAST_EXIT=$exit_code
}

assert_exit() {
    local expected="$1" name="$2"
    if [ "$expected" -eq "$LAST_EXIT" ]; then
        PASS=$((PASS + 1))
        echo "  PASS: $name"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $name (expected exit $expected, got $LAST_EXIT)"
    fi
}

assert_contains() {
    local pattern="$1" name="$2"
    if echo "$LAST_OUTPUT" | grep -qi "$pattern"; then
        PASS=$((PASS + 1))
        echo "  PASS: $name"
    else
        FAIL=$((FAIL + 1))
        echo "  FAIL: $name (expected output to contain '$pattern')"
    fi
}

# --- Fixtures ---

make_good_stl() {
    cat > "$TEST_DIR/good.scad" << 'SCAD'
cube([40, 20, 30]);
SCAD
    "$OPENSCAD" -o "$TEST_DIR/good.stl" --backend manifold --quiet "$TEST_DIR/good.scad" 2>/dev/null
    echo "$TEST_DIR/good.stl"
}

make_thin_wall_stl() {
    cat > "$TEST_DIR/thin.scad" << 'SCAD'
cube([40, 0.1, 30]);
SCAD
    "$OPENSCAD" -o "$TEST_DIR/thin.stl" --backend manifold --quiet "$TEST_DIR/thin.scad" 2>/dev/null
    echo "$TEST_DIR/thin.stl"
}

make_bracket_stl() {
    cat > "$TEST_DIR/bracket.scad" << 'SCAD'
wall = 3;
difference() {
    cube([40, 20, 30]);
    translate([wall, -1, wall])
        cube([40 - wall*2, 22, 30 - wall]);
}
SCAD
    "$OPENSCAD" -o "$TEST_DIR/bracket.stl" --backend manifold --quiet "$TEST_DIR/bracket.scad" 2>/dev/null
    echo "$TEST_DIR/bracket.stl"
}

# --- Tests ---

echo "=== printability-check.sh tests ==="

# Test 1: Good STL passes
echo ""
echo "Test 1: Good STL passes gate"
STL=$(make_good_stl)
run_gate "$STL"
assert_exit 0 "exit code 0 for printable model"
assert_contains "PASS" "output contains PASS"
assert_contains "manifold = yes" "output shows manifold status"

# Test 2: Thin wall STL fails (unprintable)
echo ""
echo "Test 2: Thin wall STL fails gate"
STL=$(make_thin_wall_stl)
run_gate "$STL"
assert_exit 1 "exit code 1 for unprintable model"
assert_contains "GATE FAIL" "output contains GATE FAIL"

# Test 3: Missing file fails
echo ""
echo "Test 3: Missing file fails gate"
run_gate "$TEST_DIR/nonexistent.stl"
assert_exit 1 "exit code 1 for missing file"
assert_contains "GATE FAIL" "output contains GATE FAIL for missing file"

# Test 4: No arguments fails
echo ""
echo "Test 4: No arguments shows usage"
run_gate
assert_exit 1 "exit code 1 with no args"
assert_contains "Usage" "output contains usage"

# Test 5: Bracket STL passes with metrics
echo ""
echo "Test 5: Bracket STL passes with metrics"
STL=$(make_bracket_stl)
run_gate "$STL"
assert_exit 0 "exit code 0 for bracket"
assert_contains "volume" "output includes volume"
assert_contains "facets" "output includes facet count"

# Test 6: JSON output mode
echo ""
echo "Test 6: JSON output mode"
STL=$(make_good_stl)
run_gate --json "$STL"
assert_exit 0 "exit code 0 with --json"
# Validate it's parseable JSON
if echo "$LAST_OUTPUT" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
    PASS=$((PASS + 1))
    echo "  PASS: output is valid JSON"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: output is not valid JSON"
fi

# Test 7: JSON output contains expected fields
echo ""
echo "Test 7: JSON output contains expected fields"
assert_contains '"verdict": "PASS"' "JSON has verdict field"
assert_contains '"manifold"' "JSON has manifold field"
assert_contains '"volume"' "JSON has volume field"
assert_contains '"filament_cm3"' "JSON has filament_cm3 field"
assert_contains '"print_time"' "JSON has print_time field"

# Test 8: Custom max support ratio
echo ""
echo "Test 8: MAX_SUPPORT_RATIO environment variable"
STL=$(make_good_stl)
LAST_OUTPUT=$(MAX_SUPPORT_RATIO=0.01 "$GATE" "$STL" 2>&1)
LAST_EXIT=$?
assert_exit 0 "exit code 0 with strict support ratio"

# Test 9: PrusaSlicer not found (simulated)
echo ""
echo "Test 9: PrusaSlicer not found fails gracefully"
LAST_OUTPUT=$(PRUSA_SLICER_BIN="/nonexistent/PrusaSlicer" "$GATE" "$STL" 2>&1)
LAST_EXIT=$?
assert_exit 1 "exit code 1 when slicer not found"
assert_contains "GATE FAIL" "output contains GATE FAIL"
assert_contains "not found" "mentions slicer not found"

# Test 10: JSON output for failure case
echo ""
echo "Test 10: JSON output for thin wall failure"
STL=$(make_thin_wall_stl)
run_gate --json "$STL"
assert_exit 1 "exit code 1 for unprintable in JSON mode"
if echo "$LAST_OUTPUT" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['verdict']=='FAIL'" 2>/dev/null; then
    PASS=$((PASS + 1))
    echo "  PASS: JSON verdict is FAIL"
else
    FAIL=$((FAIL + 1))
    echo "  FAIL: JSON verdict should be FAIL"
fi

# --- Summary ---
echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
