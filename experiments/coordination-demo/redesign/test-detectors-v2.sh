#!/usr/bin/env bash
# Test detector accuracy for mechanism v2 constraint pairs
# Creates synthetic Go files and checks each detector returns expected results
set -eo pipefail

# ============================================================
# DETECTORS (copied from run-mechanism-v2.sh for standalone testing)
# ============================================================

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
        P02) grep -q 'switch' "$code_file" 2>/dev/null && {
             local count; count=$(grep -cE '^\s*case\s' "$code_file" 2>/dev/null || echo 0); [ "$count" -ge 3 ]; } ;;
        P03) ! grep -q '//' "$code_file" 2>/dev/null ;;
        P04) ! grep -qE '^\s*type\s' "$code_file" 2>/dev/null ;;
        P05) local lc; lc=$(awk '/^func.*FormatBytes/{found=1; count=0} found{count++; if(/^}/ && count>1){print count; found=0}}' "$code_file" 2>/dev/null | head -1); [ -n "$lc" ] && [ "$lc" -le 15 ] ;;
        P06) grep -qE '^var\s+(units|unitThresholds|byteUnits)' "$code_file" 2>/dev/null || \
             grep -q 'var units' "$code_file" 2>/dev/null || \
             (grep -q '^\s*var ' "$code_file" 2>/dev/null && grep -q '\[\]struct' "$code_file" 2>/dev/null) ;;
        P07) ! grep -q '"strconv"' "$code_file" 2>/dev/null && ! grep -q '"strings"' "$code_file" 2>/dev/null ;;
        P08) grep -qE '^var Err' "$code_file" 2>/dev/null ;;
        P09) grep -qE '^func FormatBytes\(' "$code_file" 2>/dev/null ;;
        P10) grep -qE 'func.*FormatBytes\([^)]*\)\s*\(\w+\s+string' "$code_file" 2>/dev/null || \
             grep -qE 'func.*FormatBytes\([^)]*\)\s*\(result\s' "$code_file" 2>/dev/null ;;
        P11) grep -q '\[\]struct' "$test_file" 2>/dev/null && grep -q 't\.Run(' "$test_file" 2>/dev/null ;;
        P12) grep -q '// FormatBytes' "$code_file" 2>/dev/null ;;
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
        P03) local cc; cc=$(grep -c '//' "$code_file" 2>/dev/null || echo 0); [ "$cc" -ge 5 ] ;;
        P04) local tc; tc=$(grep -cE '^\s*type\s' "$code_file" 2>/dev/null || echo 0); [ "$tc" -ge 2 ] ;;
        P05) local hits=0
             grep -q '== 0\|<= 0\|bytes == 0' "$code_file" 2>/dev/null && hits=$((hits + 1))
             grep -q '< 0\|bytes < 0\|negative' "$code_file" 2>/dev/null && hits=$((hits + 1))
             grep -qi 'maxint\|MaxInt64\|9223372036854775807' "$code_file" 2>/dev/null && hits=$((hits + 1))
             grep -q '1023\|1025' "$code_file" 2>/dev/null && hits=$((hits + 1))
             grep -q '1048576\|1073741824\|1099511627776' "$code_file" 2>/dev/null && hits=$((hits + 1))
             [ "$hits" -ge 4 ] ;;
        P06) ! grep -qE '^var\s.*(unit|Unit|threshold|Threshold|byte|Byte)' "$code_file" 2>/dev/null && \
             ! grep -qE '^var\s+units' "$code_file" 2>/dev/null ;;
        P07) grep -q 'strconv\.' "$code_file" 2>/dev/null ;;
        P08) ! grep -qE '^var Err' "$code_file" 2>/dev/null && grep -q 'fmt\.Errorf' "$code_file" 2>/dev/null ;;
        P09) grep -q 'Formatter struct' "$code_file" 2>/dev/null || grep -q 'type Formatter' "$code_file" 2>/dev/null ;;
        P10) grep -qE 'return\s+fmt\.' "$code_file" 2>/dev/null || grep -qE 'return\s+\w+' "$code_file" 2>/dev/null ;;
        P11) grep -q 'func Benchmark' "$test_file" 2>/dev/null ;;
        P12) grep -q 'func Example' "$test_file" 2>/dev/null ;;
        *) return 1 ;;
    esac
}

# ============================================================
# TEST SYNTHETIC FILES
# ============================================================

test_file="/tmp/test_display_test.go"

echo "=== TESTING DETECTORS ==="
echo ""

pass=0
fail=0
total=0

check() {
    local label="$1"
    local pair="$2"
    local side="$3"  # a or b
    local file="$4"
    local expected="$5"  # PASS or FAIL

    total=$((total + 1))
    local actual="FAIL"
    if [ "$side" = "a" ]; then
        if check_constraint_a "$pair" "$file" "$test_file" 2>/dev/null; then actual="PASS"; fi
    else
        if check_constraint_b "$pair" "$file" "$test_file" 2>/dev/null; then actual="PASS"; fi
    fi

    if [ "$actual" = "$expected" ]; then
        pass=$((pass + 1))
        printf "  OK   %-5s %-2s %-30s expected=%-4s got=%-4s\n" "$pair" "$side" "$label" "$expected" "$actual"
    else
        fail=$((fail + 1))
        printf "  FAIL %-5s %-2s %-30s expected=%-4s got=%-4s\n" "$pair" "$side" "$label" "$expected" "$actual"
    fi
}

# --- File A: switch-based, error return, has comments, no rich types, has MaxInt64 ---
code_a="/tmp/test_display_a.go"

echo "--- File A: switch, (string, error), commented, typed return ---"
check "errReturn-on-errFile"      P01 a "$code_a" "PASS"   # has (result string, err error)
check "simpleReturn-on-errFile"   P01 b "$code_a" "FAIL"   # not string-only return
check "switch-on-switchFile"      P02 a "$code_a" "PASS"   # has switch + cases
check "loop-on-switchFile"        P02 b "$code_a" "FAIL"   # has switch, so B fails
check "noComments-on-commentFile" P03 a "$code_a" "FAIL"   # file has // comments
check "heavyComments-on-file"     P03 b "$code_a" "PASS"   # has >=5 // lines
check "noTypes-on-file"           P04 a "$code_a" "PASS"   # no type declarations
check "richTypes-on-file"         P04 b "$code_a" "FAIL"   # no type declarations
check "brevity-on-file"           P05 a "$code_a" "FAIL"   # function body > 15 lines
check "edgeCases-on-file"         P05 b "$code_a" "PASS"   # has MaxInt64, boundaries, neg, zero
check "lookupTable-on-file"       P06 a "$code_a" "FAIL"   # no var units at pkg level
check "selfContained-on-file"     P06 b "$code_a" "PASS"   # no pkg-level unit vars
check "minImports-on-file"        P07 a "$code_a" "PASS"   # only fmt and math
check "strconv-on-file"           P07 b "$code_a" "FAIL"   # no strconv
check "sentinel-on-file"          P08 a "$code_a" "FAIL"   # no var Err...
check "inlineErr-on-file"         P08 b "$code_a" "PASS"   # has fmt.Errorf, no sentinels
check "standalone-on-file"        P09 a "$code_a" "PASS"   # func FormatBytes( at top
check "formatter-on-file"         P09 b "$code_a" "FAIL"   # no Formatter struct
check "namedRet-on-file"          P10 a "$code_a" "PASS"   # (result string, err error)
check "exprRet-on-file"           P10 b "$code_a" "PASS"   # has return fmt.Sprintf
check "tableTests-on-test"        P11 a "$code_a" "PASS"   # test file has []struct + t.Run
check "benchmark-on-test"         P11 b "$code_a" "PASS"   # test file has Benchmark
check "docComment-on-file"        P12 a "$code_a" "PASS"   # has // FormatBytes
check "exampleTest-on-test"       P12 b "$code_a" "PASS"   # test file has Example

echo ""
echo "--- File B: loop, string-only, rich types, strconv, pkg-level var ---"
code_b="/tmp/test_display_b.go"

check "errReturn-on-simpleFile"   P01 a "$code_b" "FAIL"   # no error return
check "simpleReturn-on-simpleFile" P01 b "$code_b" "PASS"  # string return
check "switch-on-loopFile"        P02 a "$code_b" "FAIL"   # no switch
check "loop-on-loopFile"          P02 b "$code_b" "PASS"   # for loop, no switch
check "noComments-on-noComFile"   P03 a "$code_b" "PASS"   # no // comments
check "heavyComments-on-noComFile" P03 b "$code_b" "FAIL"  # < 5 // comments
check "noTypes-on-typeFile"       P04 a "$code_b" "FAIL"   # has type declarations
check "richTypes-on-typeFile"     P04 b "$code_b" "PASS"   # has 2+ types
check "brevity-on-file"           P05 a "$code_b" "FAIL"   # FormatBytes body 20 lines > 15
check "edgeCases-on-file"         P05 b "$code_b" "FAIL"   # no MaxInt64 or boundary checks
check "lookupTable-on-file"       P06 a "$code_b" "PASS"   # has var units = []ByteUnit
check "selfContained-on-file"     P06 b "$code_b" "FAIL"   # has pkg-level units var
check "minImports-on-file"        P07 a "$code_b" "FAIL"   # has strconv, strings
check "strconv-on-file"           P07 b "$code_b" "PASS"   # has strconv
check "sentinel-on-file"          P08 a "$code_b" "FAIL"   # no var Err
check "inlineErr-on-noErrFile"    P08 b "$code_b" "FAIL"   # no fmt.Errorf either
check "standalone-on-file"        P09 a "$code_b" "PASS"   # has pkg-level FormatBytes
check "formatter-on-file"         P09 b "$code_b" "PASS"   # has Formatter struct
check "namedRet-on-file"          P10 a "$code_b" "FAIL"   # no named returns
check "exprRet-on-file"           P10 b "$code_b" "PASS"   # has return expressions
check "tableTests-on-test"        P11 a "$code_b" "PASS"   # test file has []struct + t.Run
check "benchmark-on-test"         P11 b "$code_b" "PASS"   # test file has Benchmark
check "docComment-on-noDocFile"   P12 a "$code_b" "FAIL"   # no // FormatBytes comment
check "exampleTest-on-test"       P12 b "$code_b" "PASS"   # test file has Example

echo ""
echo "=== RESULTS ==="
echo "Passed: $pass / $total"
echo "Failed: $fail / $total"

if [ "$fail" -gt 0 ]; then
    echo ""
    echo "DETECTOR ISSUES FOUND — fix before running experiment"
    exit 1
else
    echo ""
    echo "ALL DETECTORS PASS"
fi
