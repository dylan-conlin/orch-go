#!/usr/bin/env bash
# Analyze Phase 1 collision patterns to identify gravitational insertion points.
#
# Input: Phase 1 results directory with agent-a/full_diff.txt and agent-b/full_diff.txt
# Output: collision_analysis.json with identified collision points
#
# Usage: ./analyze_collisions.sh PHASE1_DIR SOURCE_FILE
#   PHASE1_DIR: directory containing trial-*/agent-{a,b}/full_diff.txt
#   SOURCE_FILE: path to the original source file (e.g., pkg/display/display.go)

set -euo pipefail

PHASE1_DIR="${1:?Usage: analyze_collisions.sh PHASE1_DIR SOURCE_FILE}"
SOURCE_FILE="${2:?Usage: analyze_collisions.sh PHASE1_DIR SOURCE_FILE}"
OUTPUT="$PHASE1_DIR/collision_analysis.json"

# Extract hunk start lines from a unified diff for a specific file
extract_hunk_starts() {
    local diff_file="$1"
    local target_file="$2"
    local filename
    filename=$(basename "$target_file")

    # Find hunks for the target file, extract the original-file start line
    local in_file=false
    while IFS= read -r line; do
        if echo "$line" | grep -q "^diff --git.*${filename}"; then
            in_file=true
        elif echo "$line" | grep -q "^diff --git" && [ "$in_file" = true ]; then
            break
        elif [ "$in_file" = true ] && echo "$line" | grep -qE '@@.*@@'; then
            # Extract the start line from the original file side: @@ -START,COUNT +START,COUNT @@
            echo "$line" | sed -E 's/^@@ -([0-9]+).*/\1/'
        fi
    done < "$diff_file"
}

# Find the function that ends at or just before a given line number
find_function_at_line() {
    local source="$1"
    local target_line="$2"

    # List all function declarations with line numbers
    local prev_func=""
    local prev_end=0
    while IFS=: read -r line_num content; do
        line_num=$(echo "$line_num" | tr -d ' ')
        if [ "$line_num" -le "$target_line" ]; then
            prev_func=$(echo "$content" | sed -E 's/^func ([A-Za-z_][A-Za-z0-9_]*).*/\1/')
            prev_end="$line_num"
        fi
    done < <(grep -n '^func ' "$source")

    echo "$prev_func"
}

# List all function declarations in the source file with their line ranges
list_functions() {
    local source="$1"
    grep -n '^func ' "$source" | sed -E 's/^([0-9]+):func ([A-Za-z_][A-Za-z0-9_]*).*/\1:\2/'
}

echo "=== Collision Analysis ==="
echo "Phase 1 dir: $PHASE1_DIR"
echo "Source file: $SOURCE_FILE"

# Collect collision points across all trials
declare -A collision_counts
total_trials=0
collision_trials=0

for trial_dir in "$PHASE1_DIR"/trial-*/; do
    [ ! -d "$trial_dir" ] && continue
    total_trials=$((total_trials + 1))

    diff_a="$trial_dir/agent-a/full_diff.txt"
    diff_b="$trial_dir/agent-b/full_diff.txt"

    if [ ! -f "$diff_a" ] || [ ! -f "$diff_b" ]; then
        echo "  $(basename "$trial_dir"): SKIP (missing diffs)"
        continue
    fi

    # Check if this trial had a merge conflict
    if [ -f "$trial_dir/merge_result.csv" ]; then
        result=$(cut -d',' -f1 "$trial_dir/merge_result.csv")
        if [ "$result" != "conflict" ]; then
            echo "  $(basename "$trial_dir"): NO CONFLICT"
            continue
        fi
    fi

    collision_trials=$((collision_trials + 1))

    # Extract hunk starts for the target file
    hunks_a=$(extract_hunk_starts "$diff_a" "$SOURCE_FILE")
    hunks_b=$(extract_hunk_starts "$diff_b" "$SOURCE_FILE")

    # Find functions at collision points
    for hunk_line in $hunks_a; do
        func_name=$(find_function_at_line "$SOURCE_FILE" "$hunk_line")
        if [ -n "$func_name" ]; then
            collision_counts["$func_name"]=$(( ${collision_counts["$func_name"]:-0} + 1 ))
        fi
    done

    echo "  $(basename "$trial_dir"): Agent A hunks=[$hunks_a] Agent B hunks=[$hunks_b]"
done

echo ""
echo "=== Collision Point Summary ==="
echo "Total trials: $total_trials"
echo "Collision trials: $collision_trials"
echo ""

# Find the gravitational point (most common collision function)
gravitational_func=""
max_count=0
for func in "${!collision_counts[@]}"; do
    count=${collision_counts[$func]}
    echo "  $func: $count collisions"
    if [ "$count" -gt "$max_count" ]; then
        max_count=$count
        gravitational_func=$func
    fi
done

echo ""
echo "Gravitational function: $gravitational_func ($max_count collisions)"

# List all available insertion points
echo ""
echo "=== Available Insertion Points ==="
all_functions=$(list_functions "$SOURCE_FILE")
echo "$all_functions"

# Write analysis output
cat > "$OUTPUT" << ENDJSON
{
  "source_file": "$SOURCE_FILE",
  "total_trials": $total_trials,
  "collision_trials": $collision_trials,
  "gravitational_function": "$gravitational_func",
  "gravitational_collisions": $max_count,
  "all_functions": [
$(list_functions "$SOURCE_FILE" | while IFS=: read -r lnum fname; do
    echo "    {\"line\": $lnum, \"name\": \"$fname\"},"
done | sed '$ s/,$//')
  ]
}
ENDJSON

echo ""
echo "Analysis written to: $OUTPUT"
