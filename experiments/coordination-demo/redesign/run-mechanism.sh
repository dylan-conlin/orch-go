#!/usr/bin/env bash
# Mechanism Discrimination Experiment — Gate/Attractor Failure Modes
#
# Tests WHY behavioral constraints stop working at scale by varying
# co-resident constraint count from 1→20 and measuring per-constraint
# compliance. Three candidate mechanisms predict different failure signatures:
#
#   1. Resource competition  → gradual degradation, uniform random dropout
#   2. Interference          → pair-specific failures, set-dependent compliance
#   3. Threshold collapse    → sharp cutoff at critical N, all-or-nothing
#
# Phases:
#   1. Scaling curve  — N constraints from {1,2,4,6,8,10,14,20}, 5 trials each
#   2. Set comparison — At N_critical, compare harmonious vs clustered sets
#   3. Removal test   — Remove constraints from failing set, measure recovery
#
# Usage:
#   ./run-mechanism.sh                                    # Phase 1 (default)
#   ./run-mechanism.sh --phase 1                          # Scaling curve
#   ./run-mechanism.sh --phase 2 --critical-n 10          # Set comparison at N=10
#   ./run-mechanism.sh --phase 3 --critical-n 10          # Removal at N=10
#   ./run-mechanism.sh --trials 3                         # Fewer trials
#   ./run-mechanism.sh --model sonnet                     # Different model
#   ./run-mechanism.sh --dry-run                          # Print prompts, don't run

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

RESULTS_DIR="$RESULTS_BASE/mechanism-p${PHASE}-$TIMESTAMP"
mkdir -p "$RESULTS_DIR"

BASELINE_COMMIT=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

# ============================================================
# CONSTRAINT POOL (20 co-satisfiable constraints)
#
# Each constraint:
#   - Has a unique ID (C01-C20)
#   - Has prompt text (instruction to the agent)
#   - Has a detector (grep/awk that returns 0=compliant, 1=non-compliant)
#   - Can be satisfied SIMULTANEOUSLY with all others
#
# Categories:
#   A: Error handling (C01, C10, C11, C12)
#   B: Function style (C02, C03, C14, C20)
#   C: Documentation (C04, C09)
#   D: Testing (C05, C06, C07, C08, C18)
#   E: Type system (C13, C15, C16, C17, C19)
# ============================================================

CONSTRAINT_IDS=(
    C01 C02 C03 C04 C05 C06 C07 C08 C09 C10
    C11 C12 C13 C14 C15 C16 C17 C18 C19 C20
)

CONSTRAINT_NAMES=(
    "error_wrap"      "named_returns"   "no_else"         "const_block"
    "example_test"    "table_test"      "subtests"        "benchmark"
    "doc_comment"     "input_validation"
    "custom_error"    "sentinel_error"  "constructor"     "helper_func"
    "type_def"        "method_receiver" "interface_decl"  "testmain"
    "init_func"       "variadic_param"
)

CONSTRAINT_CATEGORIES=(
    A B B C D D D D C A
    A A E B E E E D E B
)

# Constraint prompt texts
declare -A CONSTRAINT_TEXT
CONSTRAINT_TEXT[C01]="ERROR WRAPPING: All returned errors must use fmt.Errorf() with %%w wrapping. Example: return fmt.Errorf(\"formatting bytes: %%w\", err). Never return bare errors."
CONSTRAINT_TEXT[C02]="NAMED RETURNS: All function signatures must use named return values. Example: func FormatBytes(bytes int64) (result string, err error) — not func FormatBytes(bytes int64) (string, error)."
CONSTRAINT_TEXT[C03]="NO ELSE: Never use the 'else' keyword anywhere in your code. Restructure all conditional logic using early returns, switch statements, or guard clauses instead."
CONSTRAINT_TEXT[C04]="CONST BLOCK: All numeric literals (1024, etc.) must be defined as named constants inside a const() block at package level. No magic numbers in function bodies."
CONSTRAINT_TEXT[C05]="EXAMPLE TEST: Include at least one ExampleFormatBytes() function in the test file that demonstrates usage with // Output: comments."
CONSTRAINT_TEXT[C06]="TABLE TESTS: Structure your tests as table-driven tests using a []struct{} test cases slice. Each test case should have name, input, and expected fields."
CONSTRAINT_TEXT[C07]="SUBTESTS: Use t.Run(tc.name, func(t *testing.T) { ... }) to create named subtests for each test case in your table."
CONSTRAINT_TEXT[C08]="BENCHMARK: Include a BenchmarkFormatBytes(b *testing.B) function that benchmarks the FormatBytes function with representative inputs."
CONSTRAINT_TEXT[C09]="DOC COMMENTS: Every exported function and type must have a godoc comment that starts with the name of the thing being documented. Example: // FormatBytes formats a byte count..."
CONSTRAINT_TEXT[C10]="INPUT VALIDATION: Validate all inputs explicitly. If bytes is negative, return an error (not just a string). Use the word 'invalid' in error messages. Change signature to return (string, error)."
CONSTRAINT_TEXT[C11]="CUSTOM ERROR TYPE: Define a custom error type: type FormatError struct { Value int64; Reason string }. Implement the error interface. Use it for validation errors."
CONSTRAINT_TEXT[C12]="SENTINEL ERRORS: Define sentinel error variables at package level: var ErrInvalidInput = errors.New(\"invalid input\"). Use errors.Is() for error checking in tests."
CONSTRAINT_TEXT[C13]="CONSTRUCTOR: Define a Formatter struct and a NewFormatter() constructor function that returns *Formatter. Make FormatBytes a method on Formatter instead of a standalone function."
CONSTRAINT_TEXT[C14]="HELPER FUNCTION: Extract at least one unexported helper function (lowercase name, e.g., formatUnit) that handles part of the formatting logic. Avoid duplicating code."
CONSTRAINT_TEXT[C15]="TYPE DEFINITION: Define a named type: type ByteCount int64. Add a method Format() string on ByteCount that provides the same formatting as FormatBytes."
CONSTRAINT_TEXT[C16]="METHOD RECEIVER: Implement at least one method with a pointer receiver: func (f *Formatter) Method(). Methods should modify or read receiver state."
CONSTRAINT_TEXT[C17]="INTERFACE: Define an interface: type ByteFormatter interface { Format(int64) string }. Ensure your Formatter struct satisfies this interface."
CONSTRAINT_TEXT[C18]="TESTMAIN: Include a TestMain(m *testing.M) function in the test file that calls os.Exit(m.Run()). Use it to set up any test fixtures."
CONSTRAINT_TEXT[C19]="INIT FUNCTION: Include an init() function in the code file that initializes any package-level state (e.g., pre-computing unit thresholds into a slice)."
CONSTRAINT_TEXT[C20]="VARIADIC PARAMS: Add a companion function FormatBytesAll(values ...int64) []string that accepts variadic arguments and returns formatted strings for each value."

# ============================================================
# CONSTRAINT DETECTORS
# Each detector checks code_file and/or test_file
# Returns: 0 = compliant, 1 = non-compliant
# ============================================================

check_constraint() {
    local constraint_id="$1"
    local code_file="$2"
    local test_file="$3"

    case "$constraint_id" in
        C01) # error_wrap: fmt.Errorf with %w
            grep -q 'fmt\.Errorf.*%w' "$code_file" 2>/dev/null
            ;;
        C02) # named_returns: named return values in func signatures
            # Check that new funcs (FormatBytes or related) use named returns
            grep -qP 'func\s+(\(.*?\)\s+)?\w+\([^)]*\)\s+\(\w+\s+\w+' "$code_file" 2>/dev/null
            ;;
        C03) # no_else: no 'else' keyword
            # Must NOT find 'else' — success = no match
            ! grep -qw 'else' "$code_file" 2>/dev/null
            ;;
        C04) # const_block: const() block present
            grep -q 'const (' "$code_file" 2>/dev/null
            ;;
        C05) # example_test: ExampleXxx function
            grep -q 'func Example' "$test_file" 2>/dev/null
            ;;
        C06) # table_test: []struct in test
            grep -q '\[\]struct' "$test_file" 2>/dev/null
            ;;
        C07) # subtests: t.Run(
            grep -q 't\.Run(' "$test_file" 2>/dev/null
            ;;
        C08) # benchmark: BenchmarkXxx function
            grep -q 'func Benchmark' "$test_file" 2>/dev/null
            ;;
        C09) # doc_comment: exported funcs have doc comments
            # Check that FormatBytes has a preceding // comment
            grep -qB1 'func FormatBytes' "$code_file" 2>/dev/null | grep -q '//' 2>/dev/null
            if [ $? -ne 0 ]; then
                # Fallback: check any // FormatBytes comment exists
                grep -q '// FormatBytes' "$code_file" 2>/dev/null
            fi
            ;;
        C10) # input_validation: 'invalid' in error messages
            grep -qi 'invalid' "$code_file" 2>/dev/null
            ;;
        C11) # custom_error: type XxxError struct
            grep -q 'Error struct' "$code_file" 2>/dev/null
            ;;
        C12) # sentinel_error: var ErrXxx
            grep -q 'var Err' "$code_file" 2>/dev/null
            ;;
        C13) # constructor: func New
            grep -q 'func New' "$code_file" 2>/dev/null
            ;;
        C14) # helper_func: unexported helper (func lowercase, not init/main)
            grep -qP '^func [a-z](?!nit\b)' "$code_file" 2>/dev/null
            ;;
        C15) # type_def: type Xxx int64/float64/string (not interface/struct)
            grep -qP 'type \w+ (int|float|string|byte)' "$code_file" 2>/dev/null
            ;;
        C16) # method_receiver: func (x *Xxx) or func (x Xxx)
            grep -qP 'func \([a-z]+ \*?\w+\)' "$code_file" 2>/dev/null
            ;;
        C17) # interface: type Xxx interface {
            grep -q 'interface {' "$code_file" 2>/dev/null
            ;;
        C18) # testmain: func TestMain
            grep -q 'func TestMain' "$test_file" 2>/dev/null
            ;;
        C19) # init_func: func init()
            grep -q 'func init()' "$code_file" 2>/dev/null
            ;;
        C20) # variadic: func with ...
            grep -qP 'func.*\.\.\.' "$code_file" 2>/dev/null
            ;;
        *)
            echo "Unknown constraint: $constraint_id" >&2
            return 1
            ;;
    esac
}

# ============================================================
# PROMPT BUILDER
# ============================================================

build_mechanism_prompt() {
    local constraint_ids_str="$1"  # Comma-separated constraint IDs

    local base_prompt
    base_prompt=$(cat "$PROMPTS_DIR/base-task.md")

    IFS=',' read -ra selected_constraints <<< "$constraint_ids_str"

    local constraint_section="

## ADDITIONAL CODING CONSTRAINTS

You MUST follow ALL of the constraints below. These are mandatory requirements for your implementation. Failure to follow any constraint is unacceptable.

"
    local n=1
    for cid in "${selected_constraints[@]}"; do
        constraint_section+="${n}. ${CONSTRAINT_TEXT[$cid]}
"
        n=$((n + 1))
    done

    constraint_section+="
### Constraint Compliance

- Every constraint above is MANDATORY — do not skip any
- If a constraint conflicts with the base task requirements, satisfy BOTH by adapting your approach
- After implementing, verify that each numbered constraint is satisfied in your code
"
    echo "${base_prompt}${constraint_section}"
}

# ============================================================
# AGENT RUNNER (reuses existing pattern)
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
# SCORING
# ============================================================

score_trial() {
    local trial_dir="$1"
    local constraint_ids_str="$2"

    local code_file="$trial_dir/display.go"
    local test_file="$trial_dir/display_test.go"

    IFS=',' read -ra selected_constraints <<< "$constraint_ids_str"

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

    # Score each constraint
    local scores=()
    local total_compliant=0
    local total_constraints=${#selected_constraints[@]}

    for cid in "${selected_constraints[@]}"; do
        if check_constraint "$cid" "$code_file" "$test_file"; then
            scores+=("$cid:1")
            total_compliant=$((total_compliant + 1))
        else
            scores+=("$cid:0")
        fi
    done

    local compliance_rate
    if [ "$total_constraints" -gt 0 ]; then
        compliance_rate=$(echo "scale=3; $total_compliant / $total_constraints" | bc)
    else
        compliance_rate="1.000"
    fi

    # Write scores
    cat > "$trial_dir/scores.json" << EOJSON
{
  "task_complete": $task_complete,
  "build_ok": $build_ok,
  "tests_pass": $tests_pass,
  "total_constraints": $total_constraints,
  "total_compliant": $total_compliant,
  "compliance_rate": $compliance_rate,
  "per_constraint": {
$(for s in "${scores[@]}"; do
    IFS=':' read -r cid val <<< "$s"
    echo "    \"$cid\": $val,"
done | sed '$ s/,$//')
  }
}
EOJSON

    echo "  Score: $total_compliant/$total_constraints ($compliance_rate) | task=$task_complete build=$build_ok tests=$tests_pass"
}

# ============================================================
# CONSTRAINT SET SELECTION
# ============================================================

# Select N random constraints from pool using seed
select_random_constraints() {
    local n="$1"
    local trial_seed="$2"

    # Use seed to deterministically shuffle and select
    local shuffled
    shuffled=$(printf '%s\n' "${CONSTRAINT_IDS[@]}" | awk -v seed="$trial_seed" '
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

# Phase 2: Predefined constraint sets for comparison
# Set A: Distributed across categories (one from each)
DISTRIBUTED_SET="C01,C03,C04,C05,C07,C09,C13,C15,C19,C20"
# Set B: Clustered in error+type domain (semantic overlap)
CLUSTERED_SET="C01,C02,C10,C11,C12,C13,C15,C16,C17,C19"
# Set C: Clustered in testing domain
TESTING_SET="C05,C06,C07,C08,C18,C03,C04,C09,C14,C20"

# ============================================================
# PHASE 1: SCALING CURVE
# ============================================================

run_phase1() {
    local n_values=(1 2 4 6 8 10 14 20)

    echo "================================================"
    echo "  PHASE 1: CONSTRAINT SCALING CURVE"
    echo "================================================"
    echo "N values:  ${n_values[*]}"
    echo "Trials:    $TRIALS per N"
    echo "Total:     $((${#n_values[@]} * TRIALS)) trials"
    echo ""

    for n in "${n_values[@]}"; do
        echo ""
        echo "=== N=$n constraints ==="

        for trial in $(seq 1 "$TRIALS"); do
            local trial_seed=$((SEED + n * 1000 + trial))
            local constraints
            constraints=$(select_random_constraints "$n" "$trial_seed")

            local trial_dir="$RESULTS_DIR/phase1/n${n}/trial-${trial}"
            mkdir -p "$trial_dir"

            echo ""
            echo "--- N=$n, trial $trial (seed=$trial_seed) ---"
            echo "  Constraints: $constraints"

            # Save trial metadata
            cat > "$trial_dir/metadata.json" << EOJSON
{
  "phase": 1,
  "n": $n,
  "trial": $trial,
  "seed": $trial_seed,
  "constraints": "$(echo "$constraints")",
  "constraint_count": $n
}
EOJSON

            if [ "$DRY_RUN" = true ]; then
                local prompt
                prompt=$(build_mechanism_prompt "$constraints")
                echo "$prompt" > "$trial_dir/prompt.md"
                echo "  [DRY RUN] Prompt saved to $trial_dir/prompt.md"
                # Create fake scores for dry run
                echo '{"dry_run": true}' > "$trial_dir/scores.json"
                continue
            fi

            # Create worktree
            local wt="/tmp/mech-n${n}-t${trial}-$$"
            local branch="exp/mech-n${n}-t${trial}-$$"
            cd "$PROJECT_DIR"
            git worktree add -b "$branch" "$wt" "$BASELINE_COMMIT" 2>/dev/null

            # Build and save prompt
            local prompt
            prompt=$(build_mechanism_prompt "$constraints")
            echo "$prompt" > "$trial_dir/prompt.md"

            # Run agent
            run_agent "$wt" "$prompt" "$trial_dir"

            # Score
            score_trial "$trial_dir" "$constraints"

            # Cleanup
            cd "$PROJECT_DIR"
            git worktree remove "$wt" --force 2>/dev/null || rm -rf "$wt"
            git branch -D "$branch" 2>/dev/null || true
        done
    done
}

# ============================================================
# PHASE 2: SET SPECIFICITY (at critical N)
# ============================================================

run_phase2() {
    if [ -z "$CRITICAL_N" ]; then
        echo "ERROR: Phase 2 requires --critical-n <N>"
        echo "Run Phase 1 first to determine the critical N where degradation appears."
        exit 1
    fi

    local n="$CRITICAL_N"
    local sets=("distributed" "clustered" "testing")
    local set_constraints=("$DISTRIBUTED_SET" "$CLUSTERED_SET" "$TESTING_SET")

    echo "================================================"
    echo "  PHASE 2: SET SPECIFICITY (N=$n)"
    echo "================================================"
    echo "Sets:      ${sets[*]}"
    echo "Trials:    $TRIALS per set"
    echo "Total:     $((${#sets[@]} * TRIALS)) trials"
    echo ""

    for s_idx in "${!sets[@]}"; do
        local set_name="${sets[$s_idx]}"
        local constraints="${set_constraints[$s_idx]}"

        # Trim to critical N if needed
        local constraint_count
        constraint_count=$(echo "$constraints" | tr ',' '\n' | wc -l | tr -d ' ')
        if [ "$constraint_count" -gt "$n" ]; then
            constraints=$(echo "$constraints" | tr ',' '\n' | head -n "$n" | paste -sd ',' -)
        fi

        echo ""
        echo "=== Set: $set_name (N=$n) ==="
        echo "  Constraints: $constraints"

        for trial in $(seq 1 "$TRIALS"); do
            local trial_dir="$RESULTS_DIR/phase2/${set_name}/trial-${trial}"
            mkdir -p "$trial_dir"

            echo ""
            echo "--- $set_name, trial $trial ---"

            # Save metadata
            cat > "$trial_dir/metadata.json" << EOJSON
{
  "phase": 2,
  "set_name": "$set_name",
  "n": $n,
  "trial": $trial,
  "constraints": "$constraints"
}
EOJSON

            if [ "$DRY_RUN" = true ]; then
                local prompt
                prompt=$(build_mechanism_prompt "$constraints")
                echo "$prompt" > "$trial_dir/prompt.md"
                echo "  [DRY RUN] Prompt saved"
                echo '{"dry_run": true}' > "$trial_dir/scores.json"
                continue
            fi

            local wt="/tmp/mech-p2-${set_name}-t${trial}-$$"
            local branch="exp/mech-p2-${set_name}-t${trial}-$$"
            cd "$PROJECT_DIR"
            git worktree add -b "$branch" "$wt" "$BASELINE_COMMIT" 2>/dev/null

            local prompt
            prompt=$(build_mechanism_prompt "$constraints")
            echo "$prompt" > "$trial_dir/prompt.md"

            run_agent "$wt" "$prompt" "$trial_dir"
            score_trial "$trial_dir" "$constraints"

            cd "$PROJECT_DIR"
            git worktree remove "$wt" --force 2>/dev/null || rm -rf "$wt"
            git branch -D "$branch" 2>/dev/null || true
        done
    done
}

# ============================================================
# PHASE 3: REMOVAL TEST (at critical N)
# ============================================================

run_phase3() {
    if [ -z "$CRITICAL_N" ]; then
        echo "ERROR: Phase 3 requires --critical-n <N>"
        exit 1
    fi

    local n="$CRITICAL_N"

    # Use the clustered set (most likely to show interference)
    local full_set="$CLUSTERED_SET"
    local full_set_arr
    IFS=',' read -ra full_set_arr <<< "$full_set"

    # Trim to N
    local trimmed_arr=("${full_set_arr[@]:0:$n}")
    local full_constraints
    full_constraints=$(IFS=','; echo "${trimmed_arr[*]}")

    echo "================================================"
    echo "  PHASE 3: REMOVAL TEST (N=$n)"
    echo "================================================"
    echo "Full set:  $full_constraints"
    echo "Removals:  ${#trimmed_arr[@]} (one at a time)"
    echo "Trials:    $TRIALS per removal"
    echo ""

    # First: run full set as baseline
    echo "=== Baseline (all $n constraints) ==="
    for trial in $(seq 1 "$TRIALS"); do
        local trial_dir="$RESULTS_DIR/phase3/baseline/trial-${trial}"
        mkdir -p "$trial_dir"

        echo "--- baseline, trial $trial ---"

        cat > "$trial_dir/metadata.json" << EOJSON
{
  "phase": 3,
  "removal": "none",
  "n": $n,
  "trial": $trial,
  "constraints": "$full_constraints"
}
EOJSON

        if [ "$DRY_RUN" = true ]; then
            echo '{"dry_run": true}' > "$trial_dir/scores.json"
            continue
        fi

        local wt="/tmp/mech-p3-base-t${trial}-$$"
        local branch="exp/mech-p3-base-t${trial}-$$"
        cd "$PROJECT_DIR"
        git worktree add -b "$branch" "$wt" "$BASELINE_COMMIT" 2>/dev/null

        local prompt
        prompt=$(build_mechanism_prompt "$full_constraints")
        echo "$prompt" > "$trial_dir/prompt.md"

        run_agent "$wt" "$prompt" "$trial_dir"
        score_trial "$trial_dir" "$full_constraints"

        cd "$PROJECT_DIR"
        git worktree remove "$wt" --force 2>/dev/null || rm -rf "$wt"
        git branch -D "$branch" 2>/dev/null || true
    done

    # Then: remove each constraint one at a time
    for remove_idx in "${!trimmed_arr[@]}"; do
        local removed_cid="${trimmed_arr[$remove_idx]}"
        local remaining=()
        for i in "${!trimmed_arr[@]}"; do
            if [ "$i" -ne "$remove_idx" ]; then
                remaining+=("${trimmed_arr[$i]}")
            fi
        done
        local remaining_str
        remaining_str=$(IFS=','; echo "${remaining[*]}")

        echo ""
        echo "=== Remove $removed_cid (N=$((n-1)) remaining) ==="

        for trial in $(seq 1 "$TRIALS"); do
            local trial_dir="$RESULTS_DIR/phase3/remove-${removed_cid}/trial-${trial}"
            mkdir -p "$trial_dir"

            echo "--- remove $removed_cid, trial $trial ---"

            cat > "$trial_dir/metadata.json" << EOJSON
{
  "phase": 3,
  "removal": "$removed_cid",
  "removed_category": "${CONSTRAINT_CATEGORIES[$remove_idx]}",
  "n": $((n - 1)),
  "trial": $trial,
  "constraints": "$remaining_str"
}
EOJSON

            if [ "$DRY_RUN" = true ]; then
                echo '{"dry_run": true}' > "$trial_dir/scores.json"
                continue
            fi

            local wt="/tmp/mech-p3-rm${removed_cid}-t${trial}-$$"
            local branch="exp/mech-p3-rm${removed_cid}-t${trial}-$$"
            cd "$PROJECT_DIR"
            git worktree add -b "$branch" "$wt" "$BASELINE_COMMIT" 2>/dev/null

            local prompt
            prompt=$(build_mechanism_prompt "$remaining_str")
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
for orphan in /tmp/mech-*; do
    [ -d "$orphan" ] || continue
    echo "Cleaning orphan worktree: $orphan"
    git worktree remove "$orphan" --force 2>/dev/null || rm -rf "$orphan"
done

echo "================================================"
echo "  MECHANISM DISCRIMINATION EXPERIMENT"
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

# Save experiment metadata
cat > "$RESULTS_DIR/metadata.json" << EOJSON
{
  "experiment": "mechanism-discrimination",
  "phase": $PHASE,
  "timestamp": "$TIMESTAMP",
  "baseline_commit": "$BASELINE_COMMIT",
  "model": "$MODEL",
  "model_full": "$MODEL_FULL",
  "trials": $TRIALS,
  "seed": $SEED,
  "critical_n": ${CRITICAL_N:-null},
  "dry_run": $DRY_RUN,
  "constraint_pool_size": ${#CONSTRAINT_IDS[@]}
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
echo "  1. Analyze: ./score-mechanism.sh $RESULTS_DIR"
if [ "$PHASE" -eq 1 ]; then
    echo "  2. Identify N_critical from the degradation curve"
    echo "  3. Run Phase 2: ./run-mechanism.sh --phase 2 --critical-n <N>"
elif [ "$PHASE" -eq 2 ]; then
    echo "  2. Compare set compliance rates"
    echo "  3. Run Phase 3: ./run-mechanism.sh --phase 3 --critical-n <N>"
fi
