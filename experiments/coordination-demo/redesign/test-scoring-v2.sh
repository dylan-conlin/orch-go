#!/usr/bin/env bash
# Test the scoring function with synthetic data
# Validates that score_trial produces correct JSON output
set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Create synthetic result directory
FAKE_DIR="/tmp/mech2-scoring-test"
rm -rf "$FAKE_DIR"
mkdir -p "$FAKE_DIR/phase1/n3/trial-1"

TRIAL_DIR="$FAKE_DIR/phase1/n3/trial-1"

# Use file B (loop, simple return, types, strconv, struct, no comments)
cp /tmp/test_display_b.go "$TRIAL_DIR/display.go"
cp /tmp/test_display_test.go "$TRIAL_DIR/display_test.go"
echo "ok  display  0.005s" > "$TRIAL_DIR/all_tests.txt"
touch "$TRIAL_DIR/build_output.txt"  # empty = build OK

# Create metadata
cat > "$FAKE_DIR/metadata.json" << 'EOF'
{
  "experiment": "scoring-test",
  "phase": 1,
  "tiers": {
    "hard": ["P01", "P02", "P03", "P04"],
    "medium": ["P05", "P06", "P07", "P08"],
    "easy": ["P09", "P10", "P11", "P12"]
  }
}
EOF

cat > "$TRIAL_DIR/metadata.json" << 'EOF'
{
  "phase": 1,
  "n_pairs": 3,
  "trial": 1,
  "pairs": "P01,P02,P09"
}
EOF

# Now run the score_trial by executing the scoring parts of the runner
# We'll use the dry-run trick: source just the functions we need

echo "=== SCORING TEST ==="
echo ""
echo "File B characteristics:"
echo "  - Simple return (string only) -> P01-B should win"
echo "  - Loop, no switch -> P02-B should win"
echo "  - Both standalone func + Formatter struct -> P09 both"
echo ""

# Extract and run the scoring function
cat > /tmp/score_test_runner.sh << 'OUTER'
#!/usr/bin/env bash
set -eo pipefail

declare -A PAIR_TIER
PAIR_TIER[P01]="HARD"
PAIR_TIER[P02]="HARD"
PAIR_TIER[P03]="HARD"
PAIR_TIER[P04]="HARD"
PAIR_TIER[P05]="MEDIUM"
PAIR_TIER[P06]="MEDIUM"
PAIR_TIER[P07]="MEDIUM"
PAIR_TIER[P08]="MEDIUM"
PAIR_TIER[P09]="EASY"
PAIR_TIER[P10]="EASY"
PAIR_TIER[P11]="EASY"
PAIR_TIER[P12]="EASY"

check_constraint_a() {
    local pair_id="$1"
    local code_file="$2"
    local test_file="$3"
    case "$pair_id" in
        P01) grep -qE 'func FormatBytes\([^)]*\)\s*\(string,\s*error\)' "$code_file" 2>/dev/null || \
             grep -qE 'func FormatBytes\([^)]*\)\s*\(\w+\s+string,\s*\w+\s+error\)' "$code_file" 2>/dev/null ;;
        P02) grep -q 'switch' "$code_file" 2>/dev/null && \
             grep -cE '^\s*case\s' "$code_file" 2>/dev/null | awk '{exit ($1 >= 3 ? 0 : 1)}' ;;
        P09) grep -qE '^func FormatBytes\(' "$code_file" 2>/dev/null ;;
        *) return 1 ;;
    esac
}

check_constraint_b() {
    local pair_id="$1"
    local code_file="$2"
    local test_file="$3"
    case "$pair_id" in
        P01) grep -qE 'func FormatBytes\([^)]*\)\s+string\s*\{' "$code_file" 2>/dev/null || \
             grep -qE 'func FormatBytes\([^)]*\)\s*\(result\s+string\)\s*\{' "$code_file" 2>/dev/null ;;
        P02) grep -qE '^\s*for\s' "$code_file" 2>/dev/null && ! grep -q 'switch' "$code_file" 2>/dev/null ;;
        P09) grep -q 'Formatter struct' "$code_file" 2>/dev/null || grep -q 'type Formatter' "$code_file" 2>/dev/null ;;
        *) return 1 ;;
    esac
}

score_trial() {
    local trial_dir="$1"
    local pair_ids_str="$2"
    local code_file="$trial_dir/display.go"
    local test_file="$trial_dir/display_test.go"
    IFS=',' read -ra selected_pairs <<< "$pair_ids_str"

    local task_complete=0
    if [ -f "$code_file" ] && grep -q 'FormatBytes' "$code_file" 2>/dev/null; then task_complete=1; fi
    local build_ok=0
    if [ -f "$trial_dir/build_output.txt" ] && [ ! -s "$trial_dir/build_output.txt" ]; then build_ok=1; fi
    local tests_pass=0
    if [ -f "$trial_dir/all_tests.txt" ] && grep -q '^ok' "$trial_dir/all_tests.txt" 2>/dev/null; then tests_pass=1; fi

    local pair_results=()
    local total_a_wins=0 total_b_wins=0 total_both=0 total_neither=0
    local total_pairs=${#selected_pairs[@]}

    for pid in "${selected_pairs[@]}"; do
        local a_sat=0 b_sat=0
        if check_constraint_a "$pid" "$code_file" "$test_file" 2>/dev/null; then a_sat=1; fi
        if check_constraint_b "$pid" "$code_file" "$test_file" 2>/dev/null; then b_sat=1; fi

        local resolution="neither"
        if [ "$a_sat" -eq 1 ] && [ "$b_sat" -eq 1 ]; then
            resolution="both"; total_both=$((total_both + 1))
        elif [ "$a_sat" -eq 1 ] && [ "$b_sat" -eq 0 ]; then
            resolution="a_wins"; total_a_wins=$((total_a_wins + 1))
        elif [ "$a_sat" -eq 0 ] && [ "$b_sat" -eq 1 ]; then
            resolution="b_wins"; total_b_wins=$((total_b_wins + 1))
        else
            total_neither=$((total_neither + 1))
        fi
        pair_results+=("${pid}:${a_sat}:${b_sat}:${resolution}")
    done

    local resolved=$((total_a_wins + total_b_wins + total_both))
    local resolve_rate
    if [ "$total_pairs" -gt 0 ]; then
        resolve_rate=$(echo "scale=3; $resolved / $total_pairs" | bc)
    else
        resolve_rate="1.000"
    fi

    cat > "$trial_dir/scores.json" << EOJSON
{
  "version": 2,
  "task_complete": $task_complete,
  "build_ok": $build_ok,
  "tests_pass": $tests_pass,
  "total_pairs": $total_pairs,
  "total_a_wins": $total_a_wins,
  "total_b_wins": $total_b_wins,
  "total_both": $total_both,
  "total_neither": $total_neither,
  "resolved": $resolved,
  "resolve_rate": $resolve_rate,
  "per_pair": {
$(for pr in "${pair_results[@]}"; do
    IFS=':' read -r pid a_val b_val res <<< "$pr"
    echo "    \"$pid\": {\"a\": $a_val, \"b\": $b_val, \"resolution\": \"$res\", \"tier\": \"${PAIR_TIER[$pid]}\"},"
done | sed '$ s/,$//')
  }
}
EOJSON

    echo "  Score: resolved=$resolved/$total_pairs (a=$total_a_wins b=$total_b_wins both=$total_both neither=$total_neither) | task=$task_complete build=$build_ok tests=$tests_pass"
}

# Run scoring
score_trial "$1" "$2"
OUTER

chmod +x /tmp/score_test_runner.sh
bash /tmp/score_test_runner.sh "$TRIAL_DIR" "P01,P02,P09"

echo ""
echo "=== Generated scores.json ==="
cat "$TRIAL_DIR/scores.json"

echo ""
echo "=== VERIFICATION ==="
# Verify expected results
p01_res=$(jq -r '.per_pair.P01.resolution' "$TRIAL_DIR/scores.json")
p02_res=$(jq -r '.per_pair.P02.resolution' "$TRIAL_DIR/scores.json")
p09_res=$(jq -r '.per_pair.P09.resolution' "$TRIAL_DIR/scores.json")

pass=0; fail=0
check() {
    local label="$1" actual="$2" expected="$3"
    if [ "$actual" = "$expected" ]; then
        echo "  OK   $label: $actual (expected $expected)"
        pass=$((pass + 1))
    else
        echo "  FAIL $label: $actual (expected $expected)"
        fail=$((fail + 1))
    fi
}

check "P01 (HARD: signature)" "$p01_res" "b_wins"    # File B has string return, not error
check "P02 (HARD: dispatch)" "$p02_res" "b_wins"     # File B has loop, no switch
check "P09 (EASY: func+struct)" "$p09_res" "both"    # File B has both standalone + Formatter

resolve=$(jq -r '.resolve_rate' "$TRIAL_DIR/scores.json")
check "Resolve rate" "$resolve" "1.000"               # All 3 pairs resolved

echo ""
echo "Passed: $pass / $((pass + fail))"
if [ "$fail" -gt 0 ]; then exit 1; fi
echo "SCORING VALIDATION PASSED"
