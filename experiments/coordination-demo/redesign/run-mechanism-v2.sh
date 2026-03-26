#!/usr/bin/env bash
# Mechanism Discrimination Experiment v2 — Tension-Based Constraints
#
# Redesigned after Phase 1 null result: orthogonal additive constraints showed
# ~97% compliance at N=1 through N=20 (flat curve). This version uses TENSION
# PAIRS — constraints where satisfying A makes B harder or impossible.
#
# Three tension tiers with different expected behavior:
#   HARD   — logically contradictory (one side must lose)
#   MEDIUM — both satisfiable but practically competing
#   EASY   — both satisfiable with standard Go patterns (calibration control)
#
# Mechanism discrimination via tier comparison:
#   Resource competition → all tiers degrade uniformly with N
#   Interference         → hard/medium degrade, easy holds
#   Threshold collapse   → sharp cutoff at critical N across all tiers
#
# Phases:
#   1. Scaling curve   — P pairs from {1,2,3,4,6,8,10,12}, 5 trials each
#   2. Tier comparison — At N_critical: all-hard, all-medium, all-easy, mixed
#   3. Pair removal    — Remove pairs from failing set, measure recovery
#
# Usage:
#   ./run-mechanism-v2.sh                              # Phase 1 (default)
#   ./run-mechanism-v2.sh --phase 2 --critical-n 6     # Tier comparison
#   ./run-mechanism-v2.sh --phase 3 --critical-n 6     # Pair removal
#   ./run-mechanism-v2.sh --trials 3                   # Fewer trials
#   ./run-mechanism-v2.sh --model sonnet               # Different model
#   ./run-mechanism-v2.sh --dry-run                    # Print prompts only

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
PROMPTS_DIR="$SCRIPT_DIR/prompts/mechanism"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_BASE="$SCRIPT_DIR/results"

# Defaults
PHASE=1
TRIALS=5
MODEL="haiku"
MODEL_FULL="claude-haiku-4-5-20251001"
TIMEOUT_MINUTES=10
CRITICAL_N=""
DRY_RUN=false
SEED=""

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --phase) PHASE="$2"; shift 2 ;;
        --trials) TRIALS="$2"; shift 2 ;;
        --model) MODEL="$2"; shift 2 ;;
        --critical-n) CRITICAL_N="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        --seed) SEED="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

# Generate seed if not provided
if [ -z "$SEED" ]; then
    SEED=$(od -An -tu4 -N4 /dev/urandom | tr -d ' ')
fi

# Resolve model
case "$MODEL" in
    haiku)  MODEL_FULL="claude-haiku-4-5-20251001" ;;
    opus)   MODEL_FULL="claude-opus-4-5-20251101" ;;
    sonnet) MODEL_FULL="claude-sonnet-4-5-20250514" ;;
    *)      MODEL_FULL="$MODEL" ;;
esac

RESULTS_DIR="$RESULTS_BASE/mechanism-v2-p${PHASE}-$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

# ============================================================
# TENSION-BASED CONSTRAINT POOL (12 pairs, 24 constraints)
#
# Each pair creates genuine tension: following A makes B harder.
# Three tiers predict different mechanism signatures.
#
# Pair IDs: P01-P12
# Constraint IDs: T01-T24 (T01/T02 = pair P01, etc.)
# ============================================================

# --- Pair metadata ---
PAIR_IDS=(P01 P02 P03 P04 P05 P06 P07 P08 P09 P10 P11 P12)

declare -A PAIR_TIER
PAIR_TIER[P01]="HARD"    # Error return vs simple return
PAIR_TIER[P02]="HARD"    # Switch vs loop dispatch
PAIR_TIER[P03]="HARD"    # Zero comments vs heavy comments
PAIR_TIER[P04]="HARD"    # No new types vs rich types
PAIR_TIER[P05]="MEDIUM"  # Brevity (<=15 lines) vs exhaustive edge cases
PAIR_TIER[P06]="MEDIUM"  # Package-level lookup table vs self-contained function
PAIR_TIER[P07]="MEDIUM"  # Minimal imports (fmt only) vs strconv requirement
PAIR_TIER[P08]="MEDIUM"  # Sentinel error vars vs inline-only errors
PAIR_TIER[P09]="EASY"    # Standalone function + Formatter struct (standard pattern)
PAIR_TIER[P10]="EASY"    # Named returns + expression returns (compatible)
PAIR_TIER[P11]="EASY"    # Table-driven tests + benchmark function
PAIR_TIER[P12]="EASY"    # Doc comments + example test function

declare -A PAIR_NAME
PAIR_NAME[P01]="signature_conflict"
PAIR_NAME[P02]="dispatch_strategy"
PAIR_NAME[P03]="comment_policy"
PAIR_NAME[P04]="type_definitions"
PAIR_NAME[P05]="brevity_vs_coverage"
PAIR_NAME[P06]="table_vs_inline"
PAIR_NAME[P07]="import_restriction"
PAIR_NAME[P08]="error_placement"
PAIR_NAME[P09]="func_plus_struct"
PAIR_NAME[P10]="return_style"
PAIR_NAME[P11]="test_patterns"
PAIR_NAME[P12]="documentation"

# --- Constraint A (odd-numbered) texts ---
declare -A CONSTRAINT_TEXT_A
CONSTRAINT_TEXT_A[P01]="ERROR RETURN: The FormatBytes function MUST return (string, error). Negative values MUST produce an error, not a formatted string. Use: func FormatBytes(bytes int64) (string, error). Callers must check the error."
CONSTRAINT_TEXT_A[P02]="SWITCH DISPATCH: Use a switch statement for unit selection. Each unit (B, KiB, MiB, GiB, TiB) MUST have its own explicit case clause. Do not use a loop to iterate through units."
CONSTRAINT_TEXT_A[P03]="NO COMMENTS: Write ZERO comments in your new code — not even doc comments on exported functions. The code must be entirely self-documenting through clear naming. Any // line in new code is a violation."
CONSTRAINT_TEXT_A[P04]="NO NEW TYPES: Do NOT define any new type, struct, or interface declarations. Only define functions. The keyword 'type' must not appear in any new code you write."
CONSTRAINT_TEXT_A[P05]="BREVITY: The FormatBytes function body (from opening brace to closing brace, inclusive) MUST be 15 lines or fewer. Count every line including blank lines and comments. Conciseness is paramount."
CONSTRAINT_TEXT_A[P06]="LOOKUP TABLE: Define a package-level variable containing unit thresholds and names: var units = []struct{ threshold int64; name string }{ ... }. The FormatBytes function MUST iterate this table to select the appropriate unit. No hard-coded comparisons inside FormatBytes."
CONSTRAINT_TEXT_A[P07]="MINIMAL IMPORTS: Your new code may only use the 'fmt' package for string formatting and 'math' for numeric operations. Do NOT import 'strconv', 'strings', or any other package."
CONSTRAINT_TEXT_A[P08]="SENTINEL ERRORS: Define package-level sentinel error variables: var ErrNegativeInput = errors.New(\"negative byte value\") and var ErrOverflow = errors.New(\"value exceeds maximum\"). All error returns MUST use these sentinels with fmt.Errorf wrapping."
CONSTRAINT_TEXT_A[P09]="STANDALONE FUNCTION: FormatBytes MUST be callable as a package-level function: display.FormatBytes(1024). It must work without constructing any object or calling any constructor."
CONSTRAINT_TEXT_A[P10]="NAMED RETURNS: All new functions MUST use named return values in their signatures. Example: func FormatBytes(bytes int64) (result string) — not func FormatBytes(bytes int64) string."
CONSTRAINT_TEXT_A[P11]="TABLE TESTS: Structure tests as table-driven tests using a []struct{} slice with fields: name, input, expected. Iterate with t.Run(tc.name, ...) for each case."
CONSTRAINT_TEXT_A[P12]="DOC COMMENTS: Every new exported function MUST have a godoc-style comment starting with the function name. Example: // FormatBytes formats a byte count into a human-readable string."

# --- Constraint B (even-numbered) texts ---
declare -A CONSTRAINT_TEXT_B
CONSTRAINT_TEXT_B[P01]="SIMPLE SIGNATURE: The FormatBytes function MUST return only a string — never an error. The exact signature is: func FormatBytes(bytes int64) string. Handle negative values by prepending a '-' sign. Handle all inputs gracefully without errors."
CONSTRAINT_TEXT_B[P02]="LOOP REDUCTION: Use a for loop that divides the value by 1024 iteratively and advances through a slice of unit names. Do NOT use a switch statement anywhere in the implementation. The loop terminates when the value is small enough for the current unit."
CONSTRAINT_TEXT_B[P03]="HEAVY COMMENTS: Every new exported function MUST have a doc comment. Every non-trivial code block (conditionals, loops, calculations) MUST have an inline comment explaining WHY it exists. You must have at least 5 comment lines (lines containing //) in your new code."
CONSTRAINT_TEXT_B[P04]="RICH TYPES: Define at least TWO new types: (1) type ByteUnit struct { Name string; Threshold int64 } to represent a unit level, and (2) type ByteSize int64 as a named type with a String() method. Use these types in your implementation."
CONSTRAINT_TEXT_B[P05]="EXHAUSTIVE EDGE CASES: Your implementation MUST have explicit, dedicated code paths handling each of these cases: zero bytes, negative values, math.MaxInt64, exact unit boundaries (1024, 1048576, 1073741824, 1099511627776), and off-by-one near boundaries (1023, 1025). Each must produce correctly formatted output."
CONSTRAINT_TEXT_B[P06]="SELF-CONTAINED: The FormatBytes function must be entirely self-contained. Do NOT define any package-level variables, constants, or init() functions related to units or formatting. All logic — including unit names and thresholds — must live inside the function body."
CONSTRAINT_TEXT_B[P07]="STRCONV FORMATTING: Use strconv.FormatFloat() for converting the scaled numeric value to a string. Use strings.TrimRight() to remove unnecessary trailing zeros. Do NOT use fmt.Sprintf() for numeric formatting."
CONSTRAINT_TEXT_B[P08]="INLINE ERRORS ONLY: Do NOT define any package-level error variables (no var Err... declarations). ALL errors must be constructed inline at the point of use with fmt.Errorf(). Each error message must include the specific input value that caused it."
CONSTRAINT_TEXT_B[P09]="FORMATTER STRUCT: Create a Formatter struct with at least one configurable field (e.g., Precision int). Implement FormatBytes as a method on *Formatter: func (f *Formatter) FormatBytes(bytes int64) string. Provide a NewFormatter() constructor."
CONSTRAINT_TEXT_B[P10]="EXPRESSION RETURNS: Every return statement in your new functions MUST be a single expression. Use 'return fmt.Sprintf(...)' or 'return value' directly — never assign to a named return variable and then use a bare 'return'."
CONSTRAINT_TEXT_B[P11]="BENCHMARK: Include a BenchmarkFormatBytes(b *testing.B) function that benchmarks FormatBytes with at least 3 different input sizes (small bytes, MiB range, TiB range) using b.Run() sub-benchmarks."
CONSTRAINT_TEXT_B[P12]="EXAMPLE TEST: Include an ExampleFormatBytes() function in the test file that demonstrates usage with // Output: comments showing expected output for at least 3 different inputs."

# ============================================================
# CONSTRAINT DETECTORS
# Each detector checks code_file and/or test_file.
# Returns: 0 = constraint satisfied, 1 = not satisfied
# ============================================================

check_constraint_a() {
    local pair_id="$1"
    local code_file="$2"
    local test_file="$3"

    case "$pair_id" in
        P01) # Error return: signature has (string, error)
            grep -qE 'func FormatBytes\([^)]*\)\s*\(string,\s*error\)' "$code_file" 2>/dev/null || \
            grep -qE 'func FormatBytes\([^)]*\)\s*\(\w+\s+string,\s*\w+\s+error\)' "$code_file" 2>/dev/null
            ;;
        P02) # Switch dispatch: switch keyword with case clauses
            grep -q 'switch' "$code_file" 2>/dev/null && \
            grep -cE '^\s*case\s' "$code_file" 2>/dev/null | awk '{exit ($1 >= 3 ? 0 : 1)}'
            ;;
        P03) # No comments: zero // lines in new code (check FormatBytes area)
            # Count comment lines in the file's additions (simplified: check no // in func bodies)
            ! grep -q '//' "$code_file" 2>/dev/null
            ;;
        P04) # No new types: no 'type' keyword
            ! grep -qE '^\s*type\s' "$code_file" 2>/dev/null
            ;;
        P05) # Brevity: FormatBytes body <= 15 lines
            # Extract function body line count using awk
            local line_count
            line_count=$(awk '/^func.*FormatBytes/{found=1; count=0} found{count++; if(/^}/ && count>1){print count; found=0}}' "$code_file" 2>/dev/null | head -1)
            [ -n "$line_count" ] && [ "$line_count" -le 15 ]
            ;;
        P06) # Lookup table: package-level var with unit data
            grep -qE '^var\s+(units|unitThresholds|byteUnits)' "$code_file" 2>/dev/null || \
            grep -q 'var units' "$code_file" 2>/dev/null || \
            (grep -q '^\s*var ' "$code_file" 2>/dev/null && grep -q '\[\]struct' "$code_file" 2>/dev/null)
            ;;
        P07) # Minimal imports: no strconv or strings imported
            ! grep -q '"strconv"' "$code_file" 2>/dev/null && \
            ! grep -q '"strings"' "$code_file" 2>/dev/null
            ;;
        P08) # Sentinel errors: var Err at package level
            grep -qE '^var Err' "$code_file" 2>/dev/null
            ;;
        P09) # Standalone function: package-level FormatBytes (not a method)
            grep -qE '^func FormatBytes\(' "$code_file" 2>/dev/null
            ;;
        P10) # Named returns: named return values in FormatBytes signature
            grep -qE 'func.*FormatBytes\([^)]*\)\s*\(\w+\s+string' "$code_file" 2>/dev/null || \
            grep -qE 'func.*FormatBytes\([^)]*\)\s*\(result\s' "$code_file" 2>/dev/null
            ;;
        P11) # Table tests: []struct in test file
            grep -q '\[\]struct' "$test_file" 2>/dev/null && \
            grep -q 't\.Run(' "$test_file" 2>/dev/null
            ;;
        P12) # Doc comments: // FormatBytes comment above function
            grep -q '// FormatBytes' "$code_file" 2>/dev/null
            ;;
        *)
            echo "Unknown pair: $pair_id" >&2
            return 1
            ;;
    esac
}

check_constraint_b() {
    local pair_id="$1"
    local code_file="$2"
    local test_file="$3"

    case "$pair_id" in
        P01) # Simple signature: returns string only (no error)
            grep -qE 'func FormatBytes\([^)]*\)\s+string\s*\{' "$code_file" 2>/dev/null || \
            grep -qE 'func FormatBytes\([^)]*\)\s*\(result\s+string\)\s*\{' "$code_file" 2>/dev/null
            ;;
        P02) # Loop reduction: for loop without switch
            grep -qE '^\s*for\s' "$code_file" 2>/dev/null && \
            ! grep -q 'switch' "$code_file" 2>/dev/null
            ;;
        P03) # Heavy comments: >= 5 comment lines
            local comment_count
            comment_count=$(grep -c '//' "$code_file" 2>/dev/null || echo 0)
            [ "$comment_count" -ge 5 ]
            ;;
        P04) # Rich types: at least 2 type declarations
            local type_count
            type_count=$(grep -cE '^\s*type\s' "$code_file" 2>/dev/null || echo 0)
            [ "$type_count" -ge 2 ]
            ;;
        P05) # Exhaustive edge cases: handles boundary values
            # Check for at least 4 of: zero check, negative check, MaxInt64, boundary values
            local hits=0
            grep -q '== 0\|<= 0\|bytes == 0' "$code_file" 2>/dev/null && hits=$((hits + 1))
            grep -q '< 0\|bytes < 0\|negative' "$code_file" 2>/dev/null && hits=$((hits + 1))
            grep -qi 'maxint\|MaxInt64\|9223372036854775807' "$code_file" 2>/dev/null && hits=$((hits + 1))
            grep -q '1023\|1025' "$code_file" 2>/dev/null && hits=$((hits + 1))
            grep -q '1048576\|1073741824\|1099511627776' "$code_file" 2>/dev/null && hits=$((hits + 1))
            [ "$hits" -ge 4 ]
            ;;
        P06) # Self-contained: no package-level var/const for units
            # Check that no var or const block contains unit-related terms
            ! grep -qE '^var\s.*(unit|Unit|threshold|Threshold|byte|Byte)' "$code_file" 2>/dev/null && \
            ! grep -qE '^var\s+units' "$code_file" 2>/dev/null
            ;;
        P07) # Strconv: uses strconv.FormatFloat
            grep -q 'strconv\.' "$code_file" 2>/dev/null
            ;;
        P08) # Inline errors: no package-level Err vars, uses fmt.Errorf in function
            ! grep -qE '^var Err' "$code_file" 2>/dev/null && \
            grep -q 'fmt\.Errorf' "$code_file" 2>/dev/null
            ;;
        P09) # Formatter struct: struct + method + constructor
            grep -q 'Formatter struct' "$code_file" 2>/dev/null || \
            grep -q 'type Formatter' "$code_file" 2>/dev/null
            ;;
        P10) # Expression returns: return with expression (not bare return)
            # Check that FormatBytes has return with a value, not bare return
            grep -qE 'return\s+fmt\.' "$code_file" 2>/dev/null || \
            grep -qE 'return\s+\w+' "$code_file" 2>/dev/null
            ;;
        P11) # Benchmark function
            grep -q 'func Benchmark' "$test_file" 2>/dev/null
            ;;
        P12) # Example test function
            grep -q 'func Example' "$test_file" 2>/dev/null
            ;;
        *)
            echo "Unknown pair: $pair_id" >&2
            return 1
            ;;
    esac
}

# ============================================================
# PROMPT BUILDER
# ============================================================

build_tension_prompt() {
    local pair_ids_str="$1"  # Comma-separated pair IDs

    local base_prompt
    base_prompt=$(cat "$PROMPTS_DIR/base-task.md")

    IFS=',' read -ra selected_pairs <<< "$pair_ids_str"

    local constraint_section="

## ADDITIONAL CODING CONSTRAINTS

You MUST follow ALL of the constraints below. These are mandatory requirements for your implementation. Failure to follow any constraint is unacceptable.

"
    local n=1
    for pid in "${selected_pairs[@]}"; do
        constraint_section+="${n}. ${CONSTRAINT_TEXT_A[$pid]}
"
        n=$((n + 1))
        constraint_section+="${n}. ${CONSTRAINT_TEXT_B[$pid]}
"
        n=$((n + 1))
    done

    constraint_section+="
### Constraint Compliance

- Every constraint above is MANDATORY — do not skip any
- If two constraints seem to conflict, you MUST still attempt to satisfy both
- Explain your resolution strategy in a code comment if you make tradeoffs
- After implementing, verify that each numbered constraint is satisfied in your code
"
    echo "${base_prompt}${constraint_section}"
}

# ============================================================
# AGENT RUNNER (reuses pattern from v1)
# ============================================================

run_agent() {
    local worktree="$1"
    local prompt="$2"
    local result_dir="$3"

    mkdir -p "$result_dir"

    local start_time=$(date +%s)
    echo "$start_time" > "$result_dir/start_time"

    cd "$worktree"
    timeout "${TIMEOUT_MINUTES}m" env -u CLAUDECODE BEADS_NO_DAEMON=1 claude \
        --model "$MODEL_FULL" \
        --dangerously-skip-permissions \
        -p "$prompt" \
        > "$result_dir/stdout.log" 2>"$result_dir/stderr.log" || true

    local end_time=$(date +%s)
    echo "$end_time" > "$result_dir/end_time"
    local duration=$((end_time - start_time))
    echo "$duration" > "$result_dir/duration_seconds"

    # Stage and commit agent work
    cd "$worktree"
    git add pkg/display/ 2>/dev/null || true
    git diff --cached --quiet 2>/dev/null || \
        git commit -m "agent work" --no-verify 2>/dev/null || true

    # Capture results
    cd "$worktree"
    git diff --stat "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/diff_stat.txt" 2>/dev/null || true
    git diff "$BASELINE_COMMIT"..HEAD -- ':!.beads/' > "$result_dir/full_diff.txt" 2>/dev/null || true

    # Run tests
    go test ./pkg/display/ -v > "$result_dir/all_tests.txt" 2>&1 || true
    go build ./... > "$result_dir/build_output.txt" 2>&1 || true

    # Copy modified files for scoring
    cp "$worktree/pkg/display/display.go" "$result_dir/display.go" 2>/dev/null || true
    cp "$worktree/pkg/display/display_test.go" "$result_dir/display_test.go" 2>/dev/null || true
}

# ============================================================
# SCORING — per-pair resolution
# ============================================================

score_trial() {
    local trial_dir="$1"
    local pair_ids_str="$2"

    local code_file="$trial_dir/display.go"
    local test_file="$trial_dir/display_test.go"

    IFS=',' read -ra selected_pairs <<< "$pair_ids_str"

    # Check basic task completion
    local task_complete=0
    if [ -f "$code_file" ] && grep -q 'FormatBytes' "$code_file" 2>/dev/null; then
        task_complete=1
    fi

    local build_ok=0
    if [ -f "$trial_dir/build_output.txt" ] && [ ! -s "$trial_dir/build_output.txt" ]; then
        build_ok=1
    fi

    local tests_pass=0
    if [ -f "$trial_dir/all_tests.txt" ] && grep -q '^ok' "$trial_dir/all_tests.txt" 2>/dev/null; then
        tests_pass=1
    fi

    # Score each pair: A-side and B-side independently
    local pair_results=()
    local total_a_wins=0
    local total_b_wins=0
    local total_both=0
    local total_neither=0
    local total_pairs=${#selected_pairs[@]}

    for pid in "${selected_pairs[@]}"; do
        local a_sat=0
        local b_sat=0

        if check_constraint_a "$pid" "$code_file" "$test_file" 2>/dev/null; then
            a_sat=1
        fi
        if check_constraint_b "$pid" "$code_file" "$test_file" 2>/dev/null; then
            b_sat=1
        fi

        local resolution="neither"
        if [ "$a_sat" -eq 1 ] && [ "$b_sat" -eq 1 ]; then
            resolution="both"
            total_both=$((total_both + 1))
        elif [ "$a_sat" -eq 1 ] && [ "$b_sat" -eq 0 ]; then
            resolution="a_wins"
            total_a_wins=$((total_a_wins + 1))
        elif [ "$a_sat" -eq 0 ] && [ "$b_sat" -eq 1 ]; then
            resolution="b_wins"
            total_b_wins=$((total_b_wins + 1))
        else
            total_neither=$((total_neither + 1))
        fi

        pair_results+=("${pid}:${a_sat}:${b_sat}:${resolution}")
    done

    # Composite scores
    local resolved=$((total_a_wins + total_b_wins + total_both))
    local resolve_rate
    if [ "$total_pairs" -gt 0 ]; then
        resolve_rate=$(echo "scale=3; $resolved / $total_pairs" | bc)
    else
        resolve_rate="1.000"
    fi

    # Write scores JSON
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

# ============================================================
# PAIR SET SELECTION
# ============================================================

# Select N random pairs from pool using seed
select_random_pairs() {
    local n="$1"
    local trial_seed="$2"

    local shuffled
    shuffled=$(printf '%s\n' "${PAIR_IDS[@]}" | awk -v seed="$trial_seed" '
        BEGIN { srand(seed) }
        { a[NR] = $0 }
        END {
            for (i = NR; i > 1; i--) {
                j = int(rand() * i) + 1
                tmp = a[i]; a[i] = a[j]; a[j] = tmp
            }
            for (i = 1; i <= NR; i++) print a[i]
        }
    ')

    echo "$shuffled" | head -n "$n" | paste -sd ',' -
}

# Phase 2: Predefined pair sets by tension tier
HARD_SET="P01,P02,P03,P04"
MEDIUM_SET="P05,P06,P07,P08"
EASY_SET="P09,P10,P11,P12"
MIXED_SET="P01,P05,P09,P02,P06,P10"  # 2 from each tier

# ============================================================
# PHASE 1: TENSION SCALING CURVE
# ============================================================

run_phase1() {
    local n_values=(1 2 3 4 6 8 10 12)

    echo "================================================"
    echo "  PHASE 1: TENSION PAIR SCALING CURVE"
    echo "================================================"
    echo "N values (pairs): ${n_values[*]}"
    echo "Constraints per trial: 2*N (both sides of each pair)"
    echo "Trials:    $TRIALS per N"
    echo "Total:     $((${#n_values[@]} * TRIALS)) trials"
    echo ""

    for n in "${n_values[@]}"; do
        echo ""
        echo "=== N=$n pairs ($((n * 2)) constraints) ==="

        for trial in $(seq 1 "$TRIALS"); do
            local trial_seed=$((SEED + n * 1000 + trial))
            local pairs
            pairs=$(select_random_pairs "$n" "$trial_seed")

            local trial_dir="$RESULTS_DIR/phase1/n${n}/trial-${trial}"
            mkdir -p "$trial_dir"

            echo ""
            echo "--- N=$n pairs, trial $trial (seed=$trial_seed) ---"
            echo "  Pairs: $pairs"

            # Save trial metadata
            cat > "$trial_dir/metadata.json" << EOJSON
{
  "phase": 1,
  "n_pairs": $n,
  "n_constraints": $((n * 2)),
  "trial": $trial,
  "seed": $trial_seed,
  "pairs": "$(echo "$pairs")",
  "pair_tiers": "$(IFS=',' read -ra pids <<< "$pairs"; for pid in "${pids[@]}"; do echo -n "${PAIR_TIER[$pid]},"; done | sed 's/,$//')"
}
EOJSON

            if [ "$DRY_RUN" = true ]; then
                local prompt
                prompt=$(build_tension_prompt "$pairs")
                echo "$prompt" > "$trial_dir/prompt.md"
                echo "  [DRY RUN] Prompt saved to $trial_dir/prompt.md"
                echo '{"dry_run": true}' > "$trial_dir/scores.json"
                continue
            fi

            # Create worktree
            local wt="/tmp/mech2-n${n}-t${trial}-$$"
            local branch="exp/mech2-n${n}-t${trial}-$$"
            cd "$PROJECT_DIR"
            git worktree add -b "$branch" "$wt" "$BASELINE_COMMIT" 2>/dev/null

            # Build and save prompt
            local prompt
            prompt=$(build_tension_prompt "$pairs")
            echo "$prompt" > "$trial_dir/prompt.md"

            # Run agent
            run_agent "$wt" "$prompt" "$trial_dir"

            # Score
            score_trial "$trial_dir" "$pairs"

            # Cleanup
            cd "$PROJECT_DIR"
            git worktree remove "$wt" --force 2>/dev/null || rm -rf "$wt"
            git branch -D "$branch" 2>/dev/null || true
        done
    done
}

# ============================================================
# PHASE 2: TIER COMPARISON (at critical N)
# ============================================================

run_phase2() {
    if [ -z "$CRITICAL_N" ]; then
        echo "ERROR: Phase 2 requires --critical-n <N>"
        echo "Run Phase 1 first to determine the critical N."
        exit 1
    fi

    local n="$CRITICAL_N"
    local sets=("hard" "medium" "easy" "mixed")
    local set_pairs=("$HARD_SET" "$MEDIUM_SET" "$EASY_SET" "$MIXED_SET")

    echo "================================================"
    echo "  PHASE 2: TIER COMPARISON (N=$n pairs)"
    echo "================================================"
    echo "Sets:      ${sets[*]}"
    echo "Trials:    $TRIALS per set"
    echo "Total:     $((${#sets[@]} * TRIALS)) trials"
    echo ""

    for s_idx in "${!sets[@]}"; do
        local set_name="${sets[$s_idx]}"
        local pairs="${set_pairs[$s_idx]}"

        # Trim to critical N if needed
        local pair_count
        pair_count=$(echo "$pairs" | tr ',' '\n' | wc -l | tr -d ' ')
        if [ "$pair_count" -gt "$n" ]; then
            pairs=$(echo "$pairs" | tr ',' '\n' | head -n "$n" | paste -sd ',' -)
        fi

        echo ""
        echo "=== Set: $set_name (N=$n pairs) ==="
        echo "  Pairs: $pairs"

        for trial in $(seq 1 "$TRIALS"); do
            local trial_dir="$RESULTS_DIR/phase2/${set_name}/trial-${trial}"
            mkdir -p "$trial_dir"

            echo ""
            echo "--- $set_name, trial $trial ---"

            cat > "$trial_dir/metadata.json" << EOJSON
{
  "phase": 2,
  "set_name": "$set_name",
  "n_pairs": $n,
  "trial": $trial,
  "pairs": "$pairs"
}
EOJSON

            if [ "$DRY_RUN" = true ]; then
                local prompt
                prompt=$(build_tension_prompt "$pairs")
                echo "$prompt" > "$trial_dir/prompt.md"
                echo "  [DRY RUN] Prompt saved"
                echo '{"dry_run": true}' > "$trial_dir/scores.json"
                continue
            fi

            local wt="/tmp/mech2-p2-${set_name}-t${trial}-$$"
            local branch="exp/mech2-p2-${set_name}-t${trial}-$$"
            cd "$PROJECT_DIR"
            git worktree add -b "$branch" "$wt" "$BASELINE_COMMIT" 2>/dev/null

            local prompt
            prompt=$(build_tension_prompt "$pairs")
            echo "$prompt" > "$trial_dir/prompt.md"

            run_agent "$wt" "$prompt" "$trial_dir"
            score_trial "$trial_dir" "$pairs"

            cd "$PROJECT_DIR"
            git worktree remove "$wt" --force 2>/dev/null || rm -rf "$wt"
            git branch -D "$branch" 2>/dev/null || true
        done
    done
}

# ============================================================
# PHASE 3: PAIR REMOVAL TEST
# ============================================================

run_phase3() {
    if [ -z "$CRITICAL_N" ]; then
        echo "ERROR: Phase 3 requires --critical-n <N>"
        exit 1
    fi

    local n="$CRITICAL_N"

    # Use mixed set spanning all tiers
    local full_pairs="P01,P02,P03,P05,P06,P07"
    local full_pairs_arr
    IFS=',' read -ra full_pairs_arr <<< "$full_pairs"

    # Trim to N
    local trimmed_arr=("${full_pairs_arr[@]:0:$n}")
    local full_str
    full_str=$(IFS=','; echo "${trimmed_arr[*]}")

    echo "================================================"
    echo "  PHASE 3: PAIR REMOVAL TEST (N=$n pairs)"
    echo "================================================"
    echo "Full set:  $full_str"
    echo "Removals:  ${#trimmed_arr[@]} (one at a time)"
    echo "Trials:    $TRIALS per removal"
    echo ""

    # Baseline with all pairs
    echo "=== Baseline (all $n pairs) ==="
    for trial in $(seq 1 "$TRIALS"); do
        local trial_dir="$RESULTS_DIR/phase3/baseline/trial-${trial}"
        mkdir -p "$trial_dir"

        echo "--- baseline, trial $trial ---"

        cat > "$trial_dir/metadata.json" << EOJSON
{
  "phase": 3,
  "removal": "none",
  "n_pairs": $n,
  "trial": $trial,
  "pairs": "$full_str"
}
EOJSON

        if [ "$DRY_RUN" = true ]; then
            echo '{"dry_run": true}' > "$trial_dir/scores.json"
            continue
        fi

        local wt="/tmp/mech2-p3-base-t${trial}-$$"
        local branch="exp/mech2-p3-base-t${trial}-$$"
        cd "$PROJECT_DIR"
        git worktree add -b "$branch" "$wt" "$BASELINE_COMMIT" 2>/dev/null

        local prompt
        prompt=$(build_tension_prompt "$full_str")
        echo "$prompt" > "$trial_dir/prompt.md"

        run_agent "$wt" "$prompt" "$trial_dir"
        score_trial "$trial_dir" "$full_str"

        cd "$PROJECT_DIR"
        git worktree remove "$wt" --force 2>/dev/null || rm -rf "$wt"
        git branch -D "$branch" 2>/dev/null || true
    done

    # Remove each pair one at a time
    for remove_idx in "${!trimmed_arr[@]}"; do
        local removed_pid="${trimmed_arr[$remove_idx]}"
        local remaining=()
        for i in "${!trimmed_arr[@]}"; do
            if [ "$i" -ne "$remove_idx" ]; then
                remaining+=("${trimmed_arr[$i]}")
            fi
        done
        local remaining_str
        remaining_str=$(IFS=','; echo "${remaining[*]}")

        echo ""
        echo "=== Remove $removed_pid [${PAIR_TIER[$removed_pid]}] (N=$((n-1)) remaining) ==="

        for trial in $(seq 1 "$TRIALS"); do
            local trial_dir="$RESULTS_DIR/phase3/remove-${removed_pid}/trial-${trial}"
            mkdir -p "$trial_dir"

            echo "--- remove $removed_pid, trial $trial ---"

            cat > "$trial_dir/metadata.json" << EOJSON
{
  "phase": 3,
  "removal": "$removed_pid",
  "removed_tier": "${PAIR_TIER[$removed_pid]}",
  "n_pairs": $((n - 1)),
  "trial": $trial,
  "pairs": "$remaining_str"
}
EOJSON

            if [ "$DRY_RUN" = true ]; then
                echo '{"dry_run": true}' > "$trial_dir/scores.json"
                continue
            fi

            local wt="/tmp/mech2-p3-rm${removed_pid}-t${trial}-$$"
            local branch="exp/mech2-p3-rm${removed_pid}-t${trial}-$$"
            cd "$PROJECT_DIR"
            git worktree add -b "$branch" "$wt" "$BASELINE_COMMIT" 2>/dev/null

            local prompt
            prompt=$(build_tension_prompt "$remaining_str")
            echo "$prompt" > "$trial_dir/prompt.md"

            run_agent "$wt" "$prompt" "$trial_dir"
            score_trial "$trial_dir" "$remaining_str"

            cd "$PROJECT_DIR"
            git worktree remove "$wt" --force 2>/dev/null || rm -rf "$wt"
            git branch -D "$branch" 2>/dev/null || true
        done
    done
}

# ============================================================
# MAIN
# ============================================================

# Clean up orphan worktrees
cd "$PROJECT_DIR"
git worktree prune 2>/dev/null || true
for orphan in /tmp/mech2-*; do
    [ -d "$orphan" ] || continue
    echo "Cleaning orphan worktree: $orphan"
    git worktree remove "$orphan" --force 2>/dev/null || rm -rf "$orphan"
done

echo "================================================"
echo "  MECHANISM DISCRIMINATION v2 — TENSION-BASED"
echo "================================================"
echo "Phase:      $PHASE"
echo "Project:    $PROJECT_DIR"
echo "Baseline:   $BASELINE_COMMIT"
echo "Model:      $MODEL ($MODEL_FULL)"
echo "Trials:     $TRIALS per condition"
echo "Seed:       $SEED"
echo "Results:    $RESULTS_DIR"
echo "Dry run:    $DRY_RUN"
echo ""
echo "Constraint pool: 12 pairs (24 constraints)"
echo "  HARD:   P01-P04 (logically contradictory)"
echo "  MEDIUM: P05-P08 (practically competing)"
echo "  EASY:   P09-P12 (both satisfiable)"
echo ""

# Save experiment metadata
cat > "$RESULTS_DIR/metadata.json" << EOJSON
{
  "experiment": "mechanism-discrimination-v2-tension",
  "phase": $PHASE,
  "timestamp": "$TIMESTAMP",
  "baseline_commit": "$BASELINE_COMMIT",
  "model": "$MODEL",
  "model_full": "$MODEL_FULL",
  "trials": $TRIALS,
  "seed": $SEED,
  "critical_n": ${CRITICAL_N:-null},
  "dry_run": $DRY_RUN,
  "constraint_pool": "tension-pairs",
  "total_pairs": ${#PAIR_IDS[@]},
  "total_constraints": $((${#PAIR_IDS[@]} * 2)),
  "tiers": {
    "hard": ["P01", "P02", "P03", "P04"],
    "medium": ["P05", "P06", "P07", "P08"],
    "easy": ["P09", "P10", "P11", "P12"]
  }
}
EOJSON

case "$PHASE" in
    1) run_phase1 ;;
    2) run_phase2 ;;
    3) run_phase3 ;;
    *) echo "Unknown phase: $PHASE"; exit 1 ;;
esac

echo ""
echo "================================================"
echo "  EXPERIMENT COMPLETE"
echo "================================================"
echo "Results:    $RESULTS_DIR"
echo ""
echo "Next steps:"
echo "  1. Analyze: ./score-mechanism-v2.sh $RESULTS_DIR"
if [ "$PHASE" -eq 1 ]; then
    echo "  2. Identify N_critical from tension resolution curve"
    echo "  3. Compare tier-specific degradation patterns"
    echo "  4. Run Phase 2: ./run-mechanism-v2.sh --phase 2 --critical-n <N>"
elif [ "$PHASE" -eq 2 ]; then
    echo "  2. Compare HARD vs MEDIUM vs EASY resolution rates"
    echo "  3. Run Phase 3: ./run-mechanism-v2.sh --phase 3 --critical-n <N>"
fi
