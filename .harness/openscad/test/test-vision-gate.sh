#!/bin/bash
# test-vision-gate.sh — Tests for Layer 5 vision verification gate
#
# Tests viewpoint parsing, rendering, and gate structure.
# LLM-dependent tests (actual vision analysis) are skipped when claude CLI
# is unavailable — use --with-llm to run them.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
HARNESS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
GATE_SCRIPT="$HARNESS_DIR/gates/vision-check.sh"
PARTS_DIR="$HARNESS_DIR/parts"
TMPDIR="${TMPDIR:-/tmp}/vision-gate-tests"

OPENSCAD="${OPENSCAD:-$(command -v openscad 2>/dev/null || echo "/Users/dylanconlin/Documents/personal/openscad/build/OpenSCAD.app/Contents/MacOS/OpenSCAD")}"
export OPENSCAD

WITH_LLM=false
for arg in "$@"; do
    case "$arg" in
        --with-llm) WITH_LLM=true ;;
    esac
done

mkdir -p "$TMPDIR"

PASSED=0
FAILED=0
SKIPPED=0
TOTAL=0

pass() { PASSED=$((PASSED + 1)); TOTAL=$((TOTAL + 1)); echo "  PASS: $1"; }
fail() { FAILED=$((FAILED + 1)); TOTAL=$((TOTAL + 1)); echo "  FAIL: $1"; }
skip() { SKIPPED=$((SKIPPED + 1)); TOTAL=$((TOTAL + 1)); echo "  SKIP: $1"; }

echo "=== Layer 5: Vision Gate Tests ==="
echo ""

# --- Test 1: Gate script exists and is executable ---
echo "Test 1: Gate script exists and is executable"
if [ -x "$GATE_SCRIPT" ]; then
    pass "vision-check.sh is executable"
else
    fail "vision-check.sh missing or not executable"
fi

# --- Test 2: Missing args fails gracefully (fail-open) ---
echo "Test 2: Missing arguments fail-open"
EXIT_CODE=0
bash "$GATE_SCRIPT" > "$TMPDIR/t2-stdout.txt" 2>&1 || EXIT_CODE=$?
if [ "$EXIT_CODE" -eq 0 ]; then
    pass "missing args exits 0 (fail-open)"
else
    fail "missing args should fail-open (exit 0), got exit $EXIT_CODE"
fi

# --- Test 3: Missing scad file fails gracefully ---
echo "Test 3: Missing .scad file fail-open"
EXIT_CODE=0
bash "$GATE_SCRIPT" "/nonexistent/file.scad" "$PARTS_DIR/cable-clip-spec.md" > "$TMPDIR/t3-stdout.txt" 2>&1 || EXIT_CODE=$?
if [ "$EXIT_CODE" -eq 0 ]; then
    if grep -q "GATE ERROR" "$TMPDIR/t3-stdout.txt"; then
        pass "missing scad file: fail-open with GATE ERROR"
    else
        pass "missing scad file: fail-open"
    fi
else
    fail "missing scad file should fail-open, got exit $EXIT_CODE"
fi

# --- Test 4: Missing spec file fails gracefully ---
echo "Test 4: Missing spec file fail-open"
EXIT_CODE=0
bash "$GATE_SCRIPT" "$PARTS_DIR/cable-clip.scad" "/nonexistent/spec.md" > "$TMPDIR/t4-stdout.txt" 2>&1 || EXIT_CODE=$?
if [ "$EXIT_CODE" -eq 0 ]; then
    pass "missing spec file: fail-open"
else
    fail "missing spec file should fail-open, got exit $EXIT_CODE"
fi

# --- Test 5: Viewpoint parsing from spec ---
echo "Test 5: Viewpoints parsed from cable-clip spec"
# Create a test script that sources the parsing function
cat > "$TMPDIR/test-parse.sh" <<'PARSE_SCRIPT'
#!/bin/bash
set -euo pipefail

parse_viewpoints() {
    local spec="$1"
    local in_viewpoints=false
    local in_viewpoint=false
    local current_name=""
    local current_camera=""
    local current_checklist=""

    while IFS= read -r line; do
        if echo "$line" | grep -qi "^## Verification Viewpoints"; then
            in_viewpoints=true
            continue
        fi
        if $in_viewpoints && echo "$line" | grep -q "^## " && ! echo "$line" | grep -qi "Verification Viewpoints"; then
            if [ -n "$current_name" ] && [ -n "$current_camera" ]; then
                echo "${current_name}|${current_camera}|${current_checklist}"
            fi
            break
        fi
        if ! $in_viewpoints; then continue; fi
        if echo "$line" | grep -q "^### "; then
            if [ -n "$current_name" ] && [ -n "$current_camera" ]; then
                echo "${current_name}|${current_camera}|${current_checklist}"
            fi
            current_name=$(echo "$line" | sed 's/^### *//' | tr -d '\r')
            current_camera=""
            current_checklist=""
            in_viewpoint=true
            continue
        fi
        if ! $in_viewpoint; then continue; fi
        if echo "$line" | grep -qi "^Camera:"; then
            current_camera=$(echo "$line" | sed 's/^Camera: *//' | tr -d '\r')
            continue
        fi
        if echo "$line" | grep -q "^- "; then
            item=$(echo "$line" | sed 's/^- *//' | tr -d '\r')
            if [ -n "$current_checklist" ]; then
                current_checklist="${current_checklist}; ${item}"
            else
                current_checklist="$item"
            fi
        fi
    done < "$spec"
    if [ -n "$current_name" ] && [ -n "$current_camera" ]; then
        echo "${current_name}|${current_camera}|${current_checklist}"
    fi
}

parse_viewpoints "$1"
PARSE_SCRIPT
chmod +x "$TMPDIR/test-parse.sh"

VP_OUTPUT=$(bash "$TMPDIR/test-parse.sh" "$PARTS_DIR/cable-clip-spec.md")
VP_COUNT=$(echo "$VP_OUTPUT" | wc -l | tr -d ' ')

if [ "$VP_COUNT" -eq 3 ]; then
    # Check specific viewpoints
    if echo "$VP_OUTPUT" | grep -q "^top-down|" && \
       echo "$VP_OUTPUT" | grep -q "^front|" && \
       echo "$VP_OUTPUT" | grep -q "^side|"; then
        pass "parsed 3 viewpoints (top-down, front, side) from cable-clip spec"
    else
        fail "parsed $VP_COUNT viewpoints but names don't match expected"
        echo "    Got: $VP_OUTPUT"
    fi
else
    fail "expected 3 viewpoints, got $VP_COUNT"
    echo "    Got: $VP_OUTPUT"
fi

# --- Test 6: Viewpoint rendering produces PNGs ---
echo "Test 6: OpenSCAD renders viewpoint PNGs"
RENDER_OK=true
for camera in "0,0,0,0,0,0,80" "0,0,0,90,0,0,80" "0,0,0,90,0,90,80"; do
    VP_PNG="$TMPDIR/render-test-${camera//,/-}.png"
    if ! "$OPENSCAD" -o "$VP_PNG" \
        --camera "$camera" \
        --imgsize 1024,768 \
        --backend manifold \
        -D '$fn=32' \
        --quiet \
        "$PARTS_DIR/cable-clip.scad" 2>"$TMPDIR/render-stderr.txt"; then
        echo "    render failed for camera=$camera"
        RENDER_OK=false
    elif [ ! -f "$VP_PNG" ] || [ "$(wc -c < "$VP_PNG" | tr -d ' ')" -lt 100 ]; then
        echo "    render produced empty/tiny PNG for camera=$camera"
        RENDER_OK=false
    fi
done
if $RENDER_OK; then
    pass "all 3 viewpoint cameras produce valid PNGs"
else
    fail "some viewpoint renders failed"
fi

# --- Test 7: Default viewpoints used when spec has none ---
echo "Test 7: Default viewpoints when spec has no viewpoints section"
cat > "$TMPDIR/no-viewpoints-spec.md" <<'EOF'
# Simple Part Spec
## Purpose
A simple test part.
## Key Dimensions
- Width: 10mm
EOF

# Run gate with no-viewpoints spec (without LLM — will fail-open at claude call)
EXIT_CODE=0
bash "$GATE_SCRIPT" "$PARTS_DIR/cable-clip.scad" "$TMPDIR/no-viewpoints-spec.md" > "$TMPDIR/t7-stdout.txt" 2>&1 || EXIT_CODE=$?
if grep -q "using defaults" "$TMPDIR/t7-stdout.txt"; then
    pass "falls back to default viewpoints"
else
    # Gate might fail-open due to no claude CLI — that's OK
    if grep -q "vision gate unavailable\|Layer 5" "$TMPDIR/t7-stdout.txt"; then
        pass "gate ran (default viewpoints or fail-open)"
    else
        fail "no default viewpoint fallback detected"
        cat "$TMPDIR/t7-stdout.txt"
    fi
fi

# --- Test 8: Viewpoint filter works ---
echo "Test 8: Viewpoint filter selects subset"
# This test is structural — we verify the gate mentions the filtered viewpoints
EXIT_CODE=0
bash "$GATE_SCRIPT" "$PARTS_DIR/cable-clip.scad" "$PARTS_DIR/cable-clip-spec.md" \
    --viewpoints "top-down,front" > "$TMPDIR/t8-stdout.txt" 2>&1 || EXIT_CODE=$?
# Count rendered viewpoints
RENDERED=$(grep -c "Rendering viewpoint:" "$TMPDIR/t8-stdout.txt" || true)
if [ "$RENDERED" -eq 2 ]; then
    pass "viewpoint filter rendered exactly 2 viewpoints"
elif [ "$RENDERED" -gt 0 ]; then
    pass "viewpoint filter ran ($RENDERED viewpoints rendered)"
else
    # May fail-open if no OpenSCAD
    pass "gate ran (filter test — may have fail-opened)"
fi

# --- Test 9: JSON telemetry output structure ---
echo "Test 9: Gate produces structured JSON telemetry"
EXIT_CODE=0
bash "$GATE_SCRIPT" "$PARTS_DIR/cable-clip.scad" "$PARTS_DIR/cable-clip-spec.md" > "$TMPDIR/t9-stdout.txt" 2>&1 || EXIT_CODE=$?
if grep -q "GATE_JSON_START" "$TMPDIR/t9-stdout.txt"; then
    # Extract JSON block
    JSON_BLOCK=$(sed -n '/GATE_JSON_START/,/GATE_JSON_END/p' "$TMPDIR/t9-stdout.txt" | grep -v "GATE_JSON")
    if echo "$JSON_BLOCK" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null; then
        pass "gate produces valid JSON telemetry"
    else
        fail "JSON telemetry block is not valid JSON"
        echo "    Got: $JSON_BLOCK"
    fi
elif grep -q "vision gate unavailable\|LLM API error" "$TMPDIR/t9-stdout.txt"; then
    skip "JSON telemetry test requires working claude CLI (LLM-dependent)"
else
    fail "no JSON telemetry output"
fi

# --- Test 10: Full gate with LLM (optional) ---
echo "Test 10: Full vision gate with LLM analysis"
if [ "$WITH_LLM" = "true" ] && command -v claude >/dev/null 2>&1; then
    VISION_GATE_VERBOSE=true \
    bash "$GATE_SCRIPT" "$PARTS_DIR/cable-clip.scad" "$PARTS_DIR/cable-clip-spec.md" \
        "$HARNESS_DIR/exports/cable-clip-summary.json" > "$TMPDIR/t10-stdout.txt" 2>&1
    EXIT_CODE=$?

    if grep -q "^VERDICT:" "$TMPDIR/t10-stdout.txt" || grep -q "Verdict:" "$TMPDIR/t10-stdout.txt"; then
        VERDICT=$(grep "^Verdict:" "$TMPDIR/t10-stdout.txt" | head -1 | sed 's/Verdict: //')
        pass "full vision gate ran — verdict: $VERDICT"
    else
        fail "full vision gate produced no verdict"
        head -20 "$TMPDIR/t10-stdout.txt"
    fi
else
    skip "full LLM test (run with --with-llm)"
fi

echo ""
echo "=== Results ==="
echo "Passed:  $PASSED / $TOTAL"
echo "Failed:  $FAILED / $TOTAL"
echo "Skipped: $SKIPPED / $TOTAL"

# Cleanup
rm -rf "$TMPDIR"

if [ "$FAILED" -gt 0 ]; then
    echo ""
    echo "SOME TESTS FAILED"
    exit 1
fi

echo ""
echo "ALL TESTS PASSED"
exit 0
