# Hotspot Acceleration: cmd/orch/plan_cmd_test.go

**TLDR:** Genuine extraction — 314 of 387 lines tested `pkg/plan` functions from the wrong package. Moved 7 unique tests to `pkg/plan/plan_test.go`, removed 2 duplicates. Result: plan_cmd_test.go reduced from 387 → 77 lines (80% reduction).

**Status:** Complete
**Date:** 2026-03-17

## D.E.K.N. Summary

- **Delta:** Extracted 7 misplaced `pkg/plan` tests from `cmd/orch/plan_cmd_test.go` to `pkg/plan/plan_test.go`, removed 2 duplicate tests. File reduced 387 → 77 lines.
- **Evidence:** 14 tests pass in `pkg/plan/` (7 existing + 7 moved), 2 format tests pass in `cmd/orch/`. Zero test coverage lost.
- **Knowledge:** Tests for ParseContent, ScanDir, FilterByStatus, CollectAllBeadsIDs, ParseBeadsLine were placed in cmd/orch because the types were originally defined there. After the Mar 13 extraction to pkg/plan, the tests weren't relocated.
- **Next:** Close. No further action needed.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/simple/2026-03-17-hotspot-acceleration-cmd-orch-plan.md | extends | yes | no — that inv covers plan_cmd.go, this covers plan_cmd_test.go |

## Question

Is cmd/orch/plan_cmd_test.go a genuine hotspot requiring extraction, or a false positive?

## Findings

### Finding 1: Most tests are in the wrong package

Of 387 lines in `cmd/orch/plan_cmd_test.go`, the tests break down by what they actually test:

**Tests for `pkg/plan` functions (should be in `pkg/plan/plan_test.go`):**
- `TestParsePlanFile` (12-131): Tests `plan.ParseContent` — unique coverage for owner, filename, projects, phase deps
- `TestParsePlanFile_MinimalContent` (133-151): Tests `plan.ParseContent` edge case
- `TestParsePlanFile_SupersededStatus` (153-169): Tests `plan.ParseContent` superseded field
- `TestScanPlansDir` (171-206): Tests `plan.ScanDir` — DUPLICATE of existing TestScanDir in pkg/plan
- `TestScanPlansDir_FilterActive` (208-239): Tests `plan.ScanDir` + `plan.FilterByStatus` — DUPLICATE combo
- `TestCollectAllBeadsIDs` (241-254): Tests `plan.CollectAllBeadsIDs`
- `TestParsePlanFile_CoordinationPlanTitle` (330-345): Tests `plan.ParseContent` "Coordination Plan:" prefix
- `TestParsePlanFile_StatusNotOverriddenByPhaseStatus` (347-362): Tests `plan.ParseContent` status precedence
- `TestParsePlanPhaseBeads` (364-387): Tests `plan.ParseBeadsLine`

**Tests for cmd/orch functions (belong here):**
- `TestFormatPlanShow` (256-293): Tests local `formatPlanShow`
- `TestFormatPlanStatus` (295-324): Tests local `formatPlanStatus`
- `planTestContains` helper (326-328)

**Split: ~314 lines misplaced, ~73 lines correctly placed.**

### Finding 2: Root cause — type extraction without test relocation

The Mar 13 commit (`02bb680`) extracted types and parsing from `cmd/orch/plan_cmd.go` to `pkg/plan/`. The source tests in `cmd/orch/plan_cmd_test.go` were left behind using the type aliases `PlanFile = plan.File` and `PlanPhase = plan.Phase`. This compiled but left tests in the wrong package.

## Test Performed

```bash
go test ./pkg/plan/ -v -count=1
# 14 tests PASS (7 existing + 7 moved)

go test ./cmd/orch/ -run "TestFormat" -v -count=1
# 2 format tests PASS
```

## Conclusion

**Genuine extraction.** Unlike the sister investigation on `plan_cmd.go` (false positive), the test file had a real structural issue: tests for `pkg/plan` functions were testing from `cmd/orch` via type aliases. Moved 7 unique tests, removed 2 duplicates, kept 2 format tests. File reduced 80% (387 → 77 lines).
