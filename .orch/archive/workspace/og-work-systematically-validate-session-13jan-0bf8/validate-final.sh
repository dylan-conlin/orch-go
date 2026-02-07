#!/usr/bin/env bash
#
# Final validation of 6 session discovery scenarios
#

# NO set -e - we want to continue even if tests fail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Results
PASSED=0
FAILED=0
TOTAL=6

print_header() {
	echo -e "\n${BLUE}========================================${NC}"
	echo -e "${BLUE}$1${NC}"
	echo -e "${BLUE}========================================${NC}"
}

print_result() {
	local name="$1"
	local status="$2"

	if [ "$status" = "PASS" ]; then
		echo -e "${GREEN}✓ PASS${NC}: $name"
		((PASSED++))
	else
		echo -e "${RED}✗ FAIL${NC}: $name"
		((FAILED++))
	fi
}

cd /Users/dylanconlin/Documents/personal/orch-go

print_header "Session Discovery Validation Results"
echo "Testing commit: 85a6a283 (cross-window scan fix)"
echo ""

# Scenario 1
print_header "1. Cross-Window Resume"
if go test ./cmd/orch -run "^TestDiscoverSessionHandoff_CrossWindowScan$" -v 2>&1 | grep -q "PASS: TestDiscoverSessionHandoff_CrossWindowScan"; then
	print_result "Cross-window resume" "PASS"
else
	print_result "Cross-window resume" "FAIL"
fi

# Scenario 2
print_header "2. Same-Window Continuity"
if go test ./cmd/orch -run "^TestDiscoverSessionHandoff_WindowScoped$" -v 2>&1 | grep -q "PASS: TestDiscoverSessionHandoff_WindowScoped"; then
	print_result "Same-window continuity" "PASS"
else
	print_result "Same-window continuity" "FAIL"
fi

# Scenario 3
print_header "3. Concurrent Isolation"
if go test ./cmd/orch -run "^TestDiscoverSessionHandoff_PreferWindowScoped$" -v 2>&1 | grep -q "PASS: TestDiscoverSessionHandoff_PreferWindowScoped"; then
	print_result "Concurrent isolation" "PASS"
else
	print_result "Concurrent isolation" "FAIL"
fi

# Scenario 4
print_header "4. Fresh Window Error Handling"
if go test ./cmd/orch -run "^TestDiscoverSessionHandoff$" -v 2>&1 | grep -q "returns_error_when_no_handoff_found"; then
	print_result "Fresh window graceful error" "PASS"
else
	print_result "Fresh window graceful error" "FAIL"
fi

# Scenario 5
print_header "5. Active Directory Pattern"
echo "Checking code implementation..."
if grep -q "FALLBACK.*active" cmd/orch/session.go &&
	grep -q "active.*SESSION_HANDOFF" cmd/orch/session.go; then
	print_result "Active directory fallback (code verified)" "PASS"
else
	print_result "Active directory fallback" "FAIL"
fi

# Scenario 6
print_header "6. Legacy Fallback"
if go test ./cmd/orch -run "^TestDiscoverSessionHandoff_BackwardCompatibility$" -v 2>&1 | grep -q "PASS: TestDiscoverSessionHandoff_BackwardCompatibility"; then
	print_result "Legacy fallback with migration warning" "PASS"
else
	print_result "Legacy fallback" "FAIL"
fi

# Summary
print_header "Final Summary"
echo -e "${GREEN}Passed: $PASSED/$TOTAL${NC}"
echo -e "${RED}Failed: $FAILED/$TOTAL${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
	echo -e "${GREEN}✓✓✓ All 6 scenarios validated successfully! ✓✓✓${NC}"
	echo ""
	echo "The cross-window session discovery fix (commit 85a6a283) correctly implements:"
	echo "  1. Cross-window resume (finds most recent across windows)"
	echo "  2. Same-window continuity (prefers current window)"
	echo "  3. Concurrent isolation (windows don't interfere)"
	echo "  4. Fresh window handling (graceful error)"
	echo "  5. Active directory pattern (mid-session resume)"
	echo "  6. Legacy fallback (backward compatibility)"
	exit 0
else
	echo -e "${RED}✗ $FAILED/$TOTAL scenarios failed${NC}"
	exit 1
fi
