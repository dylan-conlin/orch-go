# Codebase Audit: Tests

**TLDR:** Testing-focused audit identifying coverage gaps, flaky tests, missing test types, and test quality issues.

**Status:** STUB - To be fleshed out when needed

**When to use:** Flaky CI builds, low confidence in tests, missing edge case coverage, test suite maintenance needed

**Output:** Investigation file with testing gaps, risk assessment, coverage metrics, and test improvement roadmap

---

## Focus Areas (To be expanded)

1. **Coverage Gaps** - Modules without tests, uncovered edge cases
2. **Flaky Tests** - Time-dependent, random, inconsistent results
3. **Missing Test Types** - Unit/integration/e2e gaps
4. **Test Quality** - No assertions, over-mocking, brittle tests
5. **Test Organization** - Poor structure, hard to maintain
6. **Test Performance** - Slow tests, inefficient setup/teardown

---

## Pattern Search Commands (To be expanded)

```bash
# Modules without test files
comm -23 <(find . -name "*.py" | grep -v test | sort) \
         <(find . -name "test_*.py" | sed 's/test_//' | sort)

# Flaky test indicators (sleep, random, time-based)
rg "sleep|time\.sleep|random\.|datetime\.now" tests/

# Tests without assertions
rg "def test_" tests/ -l | xargs rg "assert" -L

# Large test files (potential god test class)
find tests/ -name "*.py" | xargs wc -l | sort -rn | head -10

# Over-mocking indicators
rg "Mock|patch|MagicMock" tests/ -c | sort -rn | head -10
```

---

*This skill stub establishes testing audit structure. Expand with coverage analysis, flaky test patterns, and test quality metrics when testing audit is needed.*
