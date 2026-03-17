---
Status: Complete
Type: investigation
Date: 2026-03-17
---

# Hotspot Acceleration: pkg/daemonconfig/plist_test.go

**TLDR:** plist_test.go's 450-line "hotspot" is a false positive — the file was born at 450 lines across 3 commits over 14 days (creation, not accretion). Extracted a `testPlistData()` helper and converted to table-driven assertions anyway, reducing to 393 lines (−57, −12.7%). No architectural concern.

## D.E.K.N. Summary

- **Delta:** plist_test.go hotspot flagged as false positive (birth churn, not accretion). Extracted test helper and table-driven assertions for ~13% reduction.
- **Evidence:** Git log shows 3 commits (Feb 19 → Mar 5): file created at 288 lines, grew +64 then +98 from feature additions, not from unchecked growth. Test-to-production ratio is 1.69x (450/266), within normal range.
- **Knowledge:** The hotspot detector flags files that grow rapidly regardless of whether growth is from creation vs accretion. A 450-line test file for 266 lines of production code is healthy.
- **Next:** No action needed. File is well below the 1,500-line extraction threshold.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Question

Is plist_test.go a genuine hotspot requiring extraction, or a false positive from file creation churn?

## Findings

### Finding 1: Growth Pattern Analysis

The file was created on 2026-02-19 and grew across 3 commits:
- `b2041049b` (Feb 19): Initial creation — 288 lines (plist consolidation)
- `a44eda646` (Feb 25): +64 lines (further consolidation)
- `e8360b461` (Mar 5): +98 lines (launchd supervision features)

This is birth + feature growth, not unchecked accretion. The "+450 lines/30d" metric is misleading — it's 100% creation churn.

### Finding 2: Test-to-Production Ratio

- `plist.go` (production): 266 lines
- `plist_test.go` (tests): 450 lines
- Ratio: 1.69x — within normal range for Go test files

### Finding 3: Duplication Pattern (addressable)

Four tests created nearly identical `PlistData` structs (12 fields each, only 1-2 differing per test). ~48 lines of pure duplication. Extracted `testPlistData()` helper.

### Finding 4: Sequential Assertions → Table-Driven

`TestGeneratePlistXML` had 11 sequential `strings.Contains` checks. Converted to table-driven loop.

## Test Performed

```bash
go test ./pkg/daemonconfig/ -v -count=1
# 53 tests passed, 0 failed (0.345s)
```

All existing tests pass after refactoring. No behavioral changes.

## Conclusion

**False positive.** The file's rapid growth came from creation (0 → 450 in 14 days), not from unchecked accretion on an existing file. At 450 lines it's well below the 1,500-line threshold. The test-to-production ratio (1.69x) is healthy.

Extracted `testPlistData()` helper and converted to table-driven assertions as routine cleanup, reducing from 450 → 393 lines (−12.7%).
