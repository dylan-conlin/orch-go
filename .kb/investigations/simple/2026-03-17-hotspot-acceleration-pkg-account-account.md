---
title: "Hotspot Acceleration: pkg/account/account_test.go"
date: 2026-03-17
status: Complete
type: investigation
---

## TLDR

`account_test.go` (1452 lines) has capacity/auto-switch tests (~816 lines) that belong with `capacity.go`. Extracting to `capacity_test.go` drops both files well under the 1500-line threshold.

## D.E.K.N. Summary

- **Delta:** Extracted ~816 lines of capacity-related tests from `account_test.go` into `capacity_test.go`, reducing `account_test.go` to ~636 lines.
- **Evidence:** All tests pass after extraction. Test functions map cleanly to source files.
- **Knowledge:** The test file grew because capacity.go functions (CapacityInfo, AutoSwitch, RecommendAccount) had their tests in account_test.go instead of the natural capacity_test.go.
- **Next:** No follow-up needed. Both files are well under the 1500-line threshold.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-pkg-account-capacity.md | related (sibling hotspot) | pending | - |

## Question

How should `pkg/account/account_test.go` (1452 lines, +649 lines/30d) be extracted to prevent crossing the 1500-line critical threshold?

## Findings

### Finding 1: Test function analysis

37 test functions in `account_test.go`. Natural grouping by source file:

**Tests for `account.go` (stay in `account_test.go`):**
- Config CRUD: `TestLoadConfig_NotExist`, `TestConfigOperations`, `TestSaveAndLoadConfig` (3 variants)
- Field tests: `TestGetConfigDir`
- Error types: `TestTokenRefreshError` (defined in account.go)
- OpenCodeAuth: `TestSaveOpenCodeAuth_*` (2 tests, `SaveOpenCodeAuth` is in account.go)
- AddAccount: `TestAddAccountPreservesMetadata_*`, `TestAddAccountNoMetadata_*`
- Atomic: `TestLoadAndSaveConfig_AtomicModify`
- Total: ~636 lines

**Tests for `capacity.go` (extract to `capacity_test.go`):**
- CapacityInfo: `TestCapacityInfo_IsHealthy`, `_IsLow`, `_IsCritical`
- Capacity errors: `TestGetCurrentCapacity_NoAuthFile`, `TestGetAccountCapacity_NotFound`, `TestCapacityError`
- Auto-switch: `TestDefaultAutoSwitchThresholds`, `TestAutoSwitchResult_*`, `TestMin`, `TestAutoSwitchThresholdLogic`, `TestAutoSwitchHeadroomCalculation`, `TestAutoSwitchMinHeadroomDelta`, `TestAutoSwitchDecisionScenarios`, `TestAutoSwitchCustomThresholds`
- RecommendAccount: all `TestRecommendAccount_*` variants (7 tests)
- ParseTier: `TestParseTierMultiplier`
- Total: ~816 lines

### Finding 2: Existing test files confirm pattern

- `oauth_test.go` (200 lines) — tests for `oauth.go`
- `cache_test.go` (108 lines) — tests for `cache.go`
- `capacity_test.go` does NOT exist yet — this is the natural home

## Test Performed

- Extracted capacity/auto-switch/recommend tests to `capacity_test.go`
- Ran `go test ./pkg/account/...`

## Conclusion

Clean extraction. All capacity.go function tests move to capacity_test.go. Both files land well under 1500 lines.
