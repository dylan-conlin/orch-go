#!/bin/bash
# Layer 3 — Printability Validation Gate (PrusaSlicer CLI)
#
# Validates that an STL file is printable via FDM by running it through
# PrusaSlicer's slicing pipeline. Checks:
#   1. Model info (manifold, dimensions, volume)
#   2. Sliceability (PrusaSlicer can generate gcode)
#   3. Print metrics (filament usage, estimated time)
#   4. Support ratio (support material vs part volume)
#
# Usage: printability-check.sh [--json] <file.stl>
#
# Environment:
#   PRUSA_SLICER_BIN    Path to PrusaSlicer binary (auto-detected)
#   PRINTER_PROFILE     Printer profile name (optional)
#   MAX_SUPPORT_RATIO   Max support/part volume ratio (default: 0.30)
#   GCODE_OUTPUT_DIR    Directory for gcode output (default: temp)
#
# Exit codes:
#   0 = printable (GATE PASS)
#   1 = not printable or error (GATE FAIL)
#
# Output format:
#   Structured text with GATE PASS/FAIL prefix, parseable metrics.
#   Use --json for machine-readable JSON output.

set -euo pipefail

# --- Argument parsing ---

JSON_MODE=false
STL_FILE=""

for arg in "$@"; do
    case "$arg" in
        --json) JSON_MODE=true ;;
        -*) echo "GATE FAIL: Unknown flag: $arg"; exit 1 ;;
        *) STL_FILE="$arg" ;;
    esac
done

if [ -z "$STL_FILE" ]; then
    echo "Usage: printability-check.sh [--json] <file.stl>"
    exit 1
fi

if [ ! -f "$STL_FILE" ]; then
    echo "GATE FAIL: File not found: $STL_FILE"
    exit 1
fi

# --- Locate PrusaSlicer ---

SLICER="${PRUSA_SLICER_BIN:-}"
if [ -z "$SLICER" ]; then
    if command -v prusa-slicer &>/dev/null; then
        SLICER="prusa-slicer"
    elif command -v PrusaSlicer &>/dev/null; then
        SLICER="PrusaSlicer"
    elif [ -x "/Applications/PrusaSlicer.app/Contents/MacOS/PrusaSlicer" ]; then
        SLICER="/Applications/PrusaSlicer.app/Contents/MacOS/PrusaSlicer"
    fi
fi

if [ -z "$SLICER" ] || ! [ -x "$SLICER" ]; then
    echo "GATE FAIL: PrusaSlicer not found. Set PRUSA_SLICER_BIN or install PrusaSlicer."
    exit 1
fi

# --- Configuration ---

MAX_SUPPORT_RATIO="${MAX_SUPPORT_RATIO:-0.30}"
GCODE_DIR="${GCODE_OUTPUT_DIR:-$(mktemp -d)}"
GCODE_FILE="$GCODE_DIR/$(basename "${STL_FILE%.stl}").gcode"
TEMP_CLEANUP=false
if [ -z "${GCODE_OUTPUT_DIR:-}" ]; then
    TEMP_CLEANUP=true
fi

cleanup() {
    if $TEMP_CLEANUP && [ -d "$GCODE_DIR" ]; then
        rm -rf "$GCODE_DIR"
    fi
}
trap cleanup EXIT

# --- Step 1: Model info ---

INFO_OUTPUT=$("$SLICER" --info "$STL_FILE" 2>&1) || {
    echo "GATE FAIL: PrusaSlicer --info failed for $STL_FILE"
    echo "$INFO_OUTPUT"
    exit 1
}

# Parse info fields
MANIFOLD=$(echo "$INFO_OUTPUT" | grep "^manifold" | awk '{print $3}')
VOLUME=$(echo "$INFO_OUTPUT" | grep "^volume" | awk '{print $3}')
FACETS=$(echo "$INFO_OUTPUT" | grep "^number_of_facets" | awk '{print $3}')
PARTS=$(echo "$INFO_OUTPUT" | grep "^number_of_parts" | awk '{print $NF}')
SIZE_X=$(echo "$INFO_OUTPUT" | grep "^size_x" | awk '{print $3}')
SIZE_Y=$(echo "$INFO_OUTPUT" | grep "^size_y" | awk '{print $3}')
SIZE_Z=$(echo "$INFO_OUTPUT" | grep "^size_z" | awk '{print $3}')

# Check manifold
if [ "$MANIFOLD" != "yes" ]; then
    if $JSON_MODE; then
        cat <<JSON
{"gate":"printability","verdict":"FAIL","reason":"non-manifold geometry","manifold":"$MANIFOLD","file":"$STL_FILE"}
JSON
    else
        echo "GATE FAIL: Non-manifold geometry (manifold = $MANIFOLD)"
    fi
    exit 1
fi

# Check zero volume
if [ "$(echo "$VOLUME" | awk '{printf "%d", $1}')" -eq 0 ]; then
    if $JSON_MODE; then
        cat <<JSON
{"gate":"printability","verdict":"FAIL","reason":"zero volume","volume":$VOLUME,"file":"$STL_FILE"}
JSON
    else
        echo "GATE FAIL: Zero volume model"
    fi
    exit 1
fi

# --- Step 2: Slice (printability check) ---

SLICE_ARGS=("--export-gcode" "$STL_FILE" "-o" "$GCODE_FILE")
if [ -n "${PRINTER_PROFILE:-}" ]; then
    SLICE_ARGS+=("--printer-profile" "$PRINTER_PROFILE")
fi

SLICE_OUTPUT=$("$SLICER" "${SLICE_ARGS[@]}" 2>&1) || {
    SLICE_EXIT=$?
    # Check for "print is empty" — definitive unprintable signal
    if echo "$SLICE_OUTPUT" | grep -qi "print is empty\|not printable"; then
        if $JSON_MODE; then
            cat <<JSON
{"gate":"printability","verdict":"FAIL","reason":"unprintable","slicer_message":"$(echo "$SLICE_OUTPUT" | tail -1)","manifold":"$MANIFOLD","volume":$VOLUME,"facets":$FACETS,"file":"$STL_FILE"}
JSON
        else
            echo "GATE FAIL: Model is not printable"
            echo "$SLICE_OUTPUT" | grep -i "empty\|not printable"
        fi
        exit 1
    fi
    # Other slicer failure
    if $JSON_MODE; then
        cat <<JSON
{"gate":"printability","verdict":"FAIL","reason":"slicer_error","exit_code":$SLICE_EXIT,"file":"$STL_FILE"}
JSON
    else
        echo "GATE FAIL: PrusaSlicer slicing failed (exit $SLICE_EXIT)"
        echo "$SLICE_OUTPUT" | tail -5
    fi
    exit 1
}

# --- Step 3: Parse gcode metrics ---

FILAMENT_MM=""
FILAMENT_CM3=""
PRINT_TIME=""

if [ -f "$GCODE_FILE" ]; then
    FILAMENT_MM=$(grep "^; filament used \[mm\]" "$GCODE_FILE" | head -1 | awk '{print $NF}')
    FILAMENT_CM3=$(grep "^; filament used \[cm3\]" "$GCODE_FILE" | head -1 | awk '{print $NF}')
    PRINT_TIME=$(grep "^; estimated printing time (normal mode)" "$GCODE_FILE" | head -1 | sed 's/.*= //')
fi

# --- Step 4: Support ratio check ---
# Slice with supports and compare filament usage

SUPPORT_RATIO="0"
SUPPORT_WARNING=false

if [ -n "$FILAMENT_CM3" ] && [ "$(echo "$FILAMENT_CM3 > 0" | bc -l 2>/dev/null || echo 0)" = "1" ]; then
    SUPPORT_GCODE="$GCODE_DIR/$(basename "${STL_FILE%.stl}")-supported.gcode"
    SUPPORT_SLICE_ARGS=("--export-gcode" "--support-material" "$STL_FILE" "-o" "$SUPPORT_GCODE")
    if [ -n "${PRINTER_PROFILE:-}" ]; then
        SUPPORT_SLICE_ARGS+=("--printer-profile" "$PRINTER_PROFILE")
    fi

    SUPPORT_OUTPUT=$("$SLICER" "${SUPPORT_SLICE_ARGS[@]}" 2>&1) || true

    if [ -f "$SUPPORT_GCODE" ]; then
        SUPPORT_CM3=$(grep "^; filament used \[cm3\]" "$SUPPORT_GCODE" | head -1 | awk '{print $NF}')
        if [ -n "$SUPPORT_CM3" ]; then
            SUPPORT_DELTA=$(echo "$SUPPORT_CM3 - $FILAMENT_CM3" | bc -l 2>/dev/null || echo "0")
            SUPPORT_RATIO=$(echo "scale=4; $SUPPORT_DELTA / $FILAMENT_CM3" | bc -l 2>/dev/null || echo "0")
            # Check if ratio exceeds threshold
            EXCEEDS=$(echo "$SUPPORT_RATIO > $MAX_SUPPORT_RATIO" | bc -l 2>/dev/null || echo "0")
            if [ "$EXCEEDS" = "1" ]; then
                SUPPORT_WARNING=true
            fi
        fi
    fi
fi

# --- Output ---

if $JSON_MODE; then
    cat <<JSON
{
  "gate": "printability",
  "verdict": "PASS",
  "file": "$STL_FILE",
  "manifold": "$MANIFOLD",
  "volume": $VOLUME,
  "facets": $FACETS,
  "parts": $PARTS,
  "dimensions": {"x": $SIZE_X, "y": $SIZE_Y, "z": $SIZE_Z},
  "filament_mm": ${FILAMENT_MM:-0},
  "filament_cm3": ${FILAMENT_CM3:-0},
  "print_time": "${PRINT_TIME:-unknown}",
  "support_ratio": $SUPPORT_RATIO,
  "support_warning": $SUPPORT_WARNING,
  "max_support_ratio": $MAX_SUPPORT_RATIO
}
JSON
else
    echo "=== Layer 3: Printability Gate ==="
    echo "File: $STL_FILE"
    echo "manifold = $MANIFOLD"
    echo "volume = $VOLUME mm³"
    echo "facets = $FACETS"
    echo "parts = $PARTS"
    echo "dimensions = ${SIZE_X} x ${SIZE_Y} x ${SIZE_Z} mm"
    if [ -n "$FILAMENT_CM3" ]; then
        echo "filament = ${FILAMENT_CM3} cm³ (${FILAMENT_MM:-?} mm)"
    fi
    if [ -n "$PRINT_TIME" ]; then
        echo "print_time = $PRINT_TIME"
    fi
    if [ "$SUPPORT_RATIO" != "0" ]; then
        echo "support_ratio = $SUPPORT_RATIO (max: $MAX_SUPPORT_RATIO)"
    fi
    if $SUPPORT_WARNING; then
        echo "GATE WARN: Support ratio $SUPPORT_RATIO exceeds threshold $MAX_SUPPORT_RATIO"
    fi
    echo "=== Layer 3 PASS ==="
fi

exit 0
