#!/bin/bash
# gates/vision-check.sh — Layer 5 vision verification gate
#
# Renders multi-angle PNGs of a 3D model via OpenSCAD camera positions,
# sends renders + functional checklist to Claude vision for structured
# judgment of functional correctness.
#
# Differs from Layer 4 (intent-check): Layer 4 analyzes source code for
# intent alignment. Layer 5 analyzes rendered output for functional
# correctness — catches things code review cannot (e.g., geometrically
# valid but functionally broken designs).
#
# Usage:
#   ./gates/vision-check.sh <scad-file> <spec-file> [summary-file] \
#       [--viewpoints "top-down,side,bottom"] [--model sonnet]
#
# Viewpoint definitions are read from the spec file's
# "## Verification Viewpoints" section. Each viewpoint defines:
#   - Name (used in verdict output)
#   - Camera string (OpenSCAD --camera format)
#   - Checklist items (what to verify in the render)
#
# If the spec has no viewpoints section, falls back to 3 default
# viewpoints (top, front, isometric).
#
# Exit codes:
#   0 = PASS or gate unavailable (fail-open)
#   1 = FAIL (only when VISION_GATE_BLOCKING=true)
#   2 = NEEDS_REVIEW
#
# Environment:
#   VISION_GATE_MODEL     - Claude model (default: sonnet)
#   VISION_GATE_BLOCKING  - "true" to block on FAIL (default: advisory)
#   VISION_GATE_VERBOSE   - "true" for full output
#   VISION_GATE_RENDER_FN - $fn for renders (default: 64)
#   VISION_CALIBRATION_DIR - path to calibration data (optional)
#   OPENSCAD              - path to openscad binary
#   OPENSCAD_DFLAGS       - additional -D flags (space-separated)

set -euo pipefail

# --- Argument parsing ---

SCAD_FILE=""
SPEC_FILE=""
SUMMARY_FILE=""
VIEWPOINT_FILTER=""

while [ $# -gt 0 ]; do
    case "$1" in
        --viewpoints)
            VIEWPOINT_FILTER="$2"
            shift 2
            ;;
        --model)
            VISION_GATE_MODEL="$2"
            shift 2
            ;;
        --blocking)
            VISION_GATE_BLOCKING="true"
            shift
            ;;
        --verbose)
            VISION_GATE_VERBOSE="true"
            shift
            ;;
        -*)
            echo "GATE ERROR: Unknown flag: $1"
            exit 0  # Fail-open
            ;;
        *)
            if [ -z "$SCAD_FILE" ]; then
                SCAD_FILE="$1"
            elif [ -z "$SPEC_FILE" ]; then
                SPEC_FILE="$1"
            elif [ -z "$SUMMARY_FILE" ]; then
                SUMMARY_FILE="$1"
            fi
            shift
            ;;
    esac
done

if [ -z "$SCAD_FILE" ] || [ -z "$SPEC_FILE" ]; then
    echo "Usage: vision-check.sh <scad-file> <spec-file> [summary-file] [--viewpoints ...] [--model ...]"
    exit 0  # Fail-open
fi

MODEL="${VISION_GATE_MODEL:-sonnet}"
BLOCKING="${VISION_GATE_BLOCKING:-false}"
VERBOSE="${VISION_GATE_VERBOSE:-false}"
RENDER_FN="${VISION_GATE_RENDER_FN:-64}"
CALIBRATION_DIR="${VISION_CALIBRATION_DIR:-}"

# Resolve openscad binary
OPENSCAD="${OPENSCAD:-$(command -v openscad 2>/dev/null || echo "")}"
if [ -z "$OPENSCAD" ]; then
    echo "GATE WARN: OpenSCAD not found — vision gate unavailable"
    exit 0  # Fail-open
fi

# Pass-through -D flags
DFLAGS=()
if [ -n "${OPENSCAD_DFLAGS:-}" ]; then
    read -ra DFLAGS <<< "$OPENSCAD_DFLAGS"
fi

# Validate inputs
if [ ! -f "$SCAD_FILE" ]; then
    echo "GATE ERROR: .scad file not found: $SCAD_FILE"
    exit 0
fi
if [ ! -f "$SPEC_FILE" ]; then
    echo "GATE ERROR: spec file not found: $SPEC_FILE"
    exit 0
fi

# Check claude CLI
if ! command -v claude >/dev/null 2>&1; then
    echo "GATE WARN: claude CLI not found — vision gate unavailable"
    exit 0
fi

# --- Setup temp directory ---

TMPDIR="${TMPDIR:-/tmp}/vision-gate-$$"
mkdir -p "$TMPDIR"
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

NAME=$(basename "$SCAD_FILE" .scad)

# --- Parse viewpoints from spec ---
#
# Expected format in spec file:
#
# ## Verification Viewpoints
#
# ### top-down
# Camera: 0,0,0,0,0,0,100
# Checklist:
# - LED cavity visible and unobstructed
# - No geometry crossing cavity boundary
#
# ### side-cross-section
# Camera: 100,0,30,0,0,0,120
# Checklist:
# - Wall thickness consistent
# - Internal features visible

parse_viewpoints() {
    local spec="$1"
    local in_viewpoints=false
    local in_viewpoint=false
    local current_name=""
    local current_camera=""
    local current_checklist=""

    while IFS= read -r line; do
        # Detect start of verification viewpoints section
        if echo "$line" | grep -qi "^## Verification Viewpoints"; then
            in_viewpoints=true
            continue
        fi

        # Detect end of viewpoints section (next ## heading)
        if $in_viewpoints && echo "$line" | grep -q "^## " && ! echo "$line" | grep -qi "Verification Viewpoints"; then
            # Emit last viewpoint
            if [ -n "$current_name" ] && [ -n "$current_camera" ]; then
                echo "${current_name}|${current_camera}|${current_checklist}"
            fi
            break
        fi

        if ! $in_viewpoints; then
            continue
        fi

        # New viewpoint heading (### name)
        if echo "$line" | grep -q "^### "; then
            # Emit previous viewpoint
            if [ -n "$current_name" ] && [ -n "$current_camera" ]; then
                echo "${current_name}|${current_camera}|${current_checklist}"
            fi
            current_name=$(echo "$line" | sed 's/^### *//' | tr -d '\r')
            current_camera=""
            current_checklist=""
            in_viewpoint=true
            continue
        fi

        if ! $in_viewpoint; then
            continue
        fi

        # Camera line
        if echo "$line" | grep -qi "^Camera:"; then
            current_camera=$(echo "$line" | sed 's/^Camera: *//' | tr -d '\r')
            continue
        fi

        # Checklist items (- bullet points)
        if echo "$line" | grep -q "^- "; then
            item=$(echo "$line" | sed 's/^- *//' | tr -d '\r')
            if [ -n "$current_checklist" ]; then
                current_checklist="${current_checklist}; ${item}"
            else
                current_checklist="$item"
            fi
        fi
    done < "$spec"

    # Emit last viewpoint if file ended inside viewpoints section
    if [ -n "$current_name" ] && [ -n "$current_camera" ]; then
        echo "${current_name}|${current_camera}|${current_checklist}"
    fi
}

# Default viewpoints when spec has none
default_viewpoints() {
    echo "top-down|0,0,0,55,0,0,150|Overall shape visible from above; Features centered and symmetric"
    echo "front|0,0,0,90,0,0,150|Front profile matches expected shape; Wall thickness appears adequate"
    echo "isometric|0,0,0,55,0,25,150|3D proportions look correct; No degenerate or collapsed geometry"
}

# Collect viewpoints
VIEWPOINTS=$(parse_viewpoints "$SPEC_FILE")
if [ -z "$VIEWPOINTS" ]; then
    VIEWPOINTS=$(default_viewpoints)
    echo "INFO: No viewpoints in spec — using defaults (top-down, front, isometric)"
fi

# Apply viewpoint filter if specified
if [ -n "$VIEWPOINT_FILTER" ]; then
    FILTERED=""
    IFS=',' read -ra FILTER_NAMES <<< "$VIEWPOINT_FILTER"
    while IFS= read -r vp; do
        vp_name=$(echo "$vp" | cut -d'|' -f1)
        for fname in "${FILTER_NAMES[@]}"; do
            if [ "$vp_name" = "$fname" ]; then
                if [ -n "$FILTERED" ]; then
                    FILTERED="${FILTERED}
${vp}"
                else
                    FILTERED="$vp"
                fi
            fi
        done
    done <<< "$VIEWPOINTS"
    if [ -n "$FILTERED" ]; then
        VIEWPOINTS="$FILTERED"
    else
        echo "GATE WARN: No viewpoints matched filter '$VIEWPOINT_FILTER' — using all"
    fi
fi

echo "=== Layer 5: Vision Verification Gate ==="
echo "File: $SCAD_FILE"
echo "Spec: $SPEC_FILE"
echo "Model: $MODEL"

# --- Render viewpoints ---

RENDER_COUNT=0
RENDER_FILES=""
VIEWPOINT_DATA=""

while IFS= read -r vp; do
    vp_name=$(echo "$vp" | cut -d'|' -f1)
    vp_camera=$(echo "$vp" | cut -d'|' -f2)
    vp_checklist=$(echo "$vp" | cut -d'|' -f3)

    RENDER_COUNT=$((RENDER_COUNT + 1))
    PNG_OUT="$TMPDIR/${NAME}-vp-${vp_name}.png"

    echo "  Rendering viewpoint: $vp_name (camera: $vp_camera)"

    if "$OPENSCAD" -o "$PNG_OUT" \
        --camera "$vp_camera" \
        --imgsize 1024,768 \
        --backend manifold \
        -D "\$fn=$RENDER_FN" \
        ${DFLAGS[@]+"${DFLAGS[@]}"} \
        --quiet \
        "$SCAD_FILE" 2>"$TMPDIR/render-stderr-${vp_name}.txt"; then

        if [ -f "$PNG_OUT" ] && [ "$(wc -c < "$PNG_OUT" | tr -d ' ')" -gt 100 ]; then
            RENDER_FILES="${RENDER_FILES} ${PNG_OUT}"
            VIEWPOINT_DATA="${VIEWPOINT_DATA}
VIEWPOINT: ${vp_name}
IMAGE: ${PNG_OUT}
CHECKLIST: ${vp_checklist}
---"
        else
            echo "  WARN: Render produced empty/tiny image for $vp_name"
        fi
    else
        echo "  WARN: Render failed for viewpoint $vp_name"
        if [ "$VERBOSE" = "true" ]; then
            cat "$TMPDIR/render-stderr-${vp_name}.txt"
        fi
    fi
done <<< "$VIEWPOINTS"

if [ "$RENDER_COUNT" -eq 0 ]; then
    echo "GATE WARN: No viewpoints to render"
    exit 0
fi

RENDERED_COUNT=$(echo "$RENDER_FILES" | wc -w | tr -d ' ')
if [ "$RENDERED_COUNT" -eq 0 ]; then
    echo "GATE WARN: All renders failed — vision gate cannot proceed"
    exit 0
fi

echo "  Rendered $RENDERED_COUNT / $RENDER_COUNT viewpoints"

# --- Read spec for functional checklist ---

SPEC_CONTENT=$(cat "$SPEC_FILE")

SUMMARY_CONTENT="{}"
if [ -n "$SUMMARY_FILE" ] && [ -f "$SUMMARY_FILE" ]; then
    SUMMARY_CONTENT=$(cat "$SUMMARY_FILE")
fi

# --- Load calibration context if available ---

CALIBRATION_CONTEXT=""
if [ -n "$CALIBRATION_DIR" ] && [ -d "$CALIBRATION_DIR" ]; then
    # Look for calibration notes
    if [ -f "$CALIBRATION_DIR/notes.md" ]; then
        CALIBRATION_CONTEXT="
CALIBRATION NOTES (known failure patterns for this project):
---
$(cat "$CALIBRATION_DIR/notes.md")
---"
    fi
fi

# --- Build vision prompt ---

GATE_PROMPT="You are a vision-based functional verification gate for 3D-printable OpenSCAD parts. You are examining rendered PNG images of a 3D model from multiple viewpoints. Your job is to verify that the rendered geometry meets the functional requirements described in the spec.

IMPORTANT: You are reviewing RENDERED OUTPUT, not source code. Judge what you SEE in the images, not what the code says. A design can look correct in code but be functionally broken in the render (disconnected channels, misaligned features, degenerate geometry).

DESIGN SPECIFICATION:
---
$SPEC_CONTENT
---

GEOMETRY METADATA:
---
$SUMMARY_CONTENT
---
$CALIBRATION_CONTEXT

You will receive $RENDERED_COUNT rendered viewpoint images. For each viewpoint, verify the checklist items by examining what is visible in the image.

VIEWPOINTS AND CHECKLISTS:
$VIEWPOINT_DATA

For each viewpoint, judge PASS/FAIL/UNCLEAR on each checklist item based on what is visible in the render.

Respond in EXACTLY this format (no other text):
VERDICT: PASS
CONFIDENCE: 0.85
VIEWPOINTS:
  <viewpoint-name>: PASS
    - <checklist item>: PASS|FAIL|UNCLEAR - <1-sentence observation>
  <viewpoint-name>: FAIL
    - <checklist item>: PASS|FAIL|UNCLEAR - <1-sentence observation>
REASONING: One paragraph summarizing the functional assessment.

Valid VERDICT values: PASS, FAIL, NEEDS_REVIEW
Valid per-item values: PASS, FAIL, UNCLEAR
CONFIDENCE must be between 0.0 and 1.0
Overall VERDICT is FAIL if ANY checklist item is FAIL with CONFIDENCE > 0.7"

# --- Call Claude vision ---

echo "  Sending $RENDERED_COUNT renders to Claude vision ($MODEL)..."

# Build claude CLI args with image files
CLAUDE_ARGS=(--print --model "$MODEL")
for img in $RENDER_FILES; do
    CLAUDE_ARGS+=(--file "$img")
done

RESULT=$(echo "$GATE_PROMPT" | claude "${CLAUDE_ARGS[@]}" 2>/dev/null) || {
    echo "GATE WARN: Vision gate unavailable (LLM API error)"
    echo "Proceeding without vision check (fail-open)"
    exit 0
}

# --- Parse verdict ---

VERDICT=$(echo "$RESULT" | grep "^VERDICT:" | head -1 | sed 's/VERDICT: *//')
CONFIDENCE=$(echo "$RESULT" | grep "^CONFIDENCE:" | head -1 | sed 's/CONFIDENCE: *//')

if [ -z "$VERDICT" ]; then
    echo "GATE WARN: Could not parse vision verdict"
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

# Emit structured output for measurement/telemetry
echo "---GATE_OUTPUT_START---"
echo "$RESULT"
echo "---GATE_OUTPUT_END---"

# --- JSON output for telemetry ---

# Count per-viewpoint results
VP_PASS=$(echo "$RESULT" | grep -c "^  [a-z].*: PASS$" || true)
VP_FAIL=$(echo "$RESULT" | grep -c "^  [a-z].*: FAIL$" || true)

echo "---GATE_JSON_START---"
cat <<JSON
{
  "gate": "vision-check",
  "layer": 5,
  "verdict": "$VERDICT",
  "confidence": ${CONFIDENCE:-0},
  "viewpoints_rendered": $RENDERED_COUNT,
  "viewpoints_total": $RENDER_COUNT,
  "viewpoints_pass": $VP_PASS,
  "viewpoints_fail": $VP_FAIL,
  "model": "$MODEL",
  "scad_file": "$SCAD_FILE",
  "spec_file": "$SPEC_FILE"
}
JSON
echo "---GATE_JSON_END---"

# --- Exit based on verdict ---

case "$VERDICT" in
    PASS)
        echo "=== Layer 5 PASS ==="
        exit 0
        ;;
    FAIL)
        if [ "$BLOCKING" = "true" ]; then
            echo "=== Layer 5 FAIL (blocking mode) ==="
            exit 1
        else
            echo "=== Layer 5 WARN (advisory: FAIL) ==="
            exit 0
        fi
        ;;
    NEEDS_REVIEW)
        echo "=== Layer 5 WARN (needs human review) ==="
        exit 2
        ;;
    *)
        echo "GATE WARN: Unknown verdict: $VERDICT"
        exit 0
        ;;
esac
