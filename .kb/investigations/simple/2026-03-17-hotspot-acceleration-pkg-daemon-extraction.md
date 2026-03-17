---
title: "Hotspot acceleration: pkg/daemon/extraction_test.go"
status: Complete
date: 2026-03-17
---

## TLDR

extraction_test.go (739 lines) split into two files: unit tests for extraction helpers remain (500 lines), daemon OnceExcluding integration tests moved to extraction_integration_test.go (237 lines). Duplicate mockDaemonHotspotChecker eliminated — integration tests now reuse mockHotspotChecker.

## D.E.K.N. Summary

- **Delta:** extraction_test.go reduced from 739 to 500 lines (-32%). Daemon auto-extraction integration tests split to extraction_integration_test.go.
- **Evidence:** `go test ./pkg/daemon/ -run "TestInferTargetFiles|TestFindCritical|TestMatchesFile|TestGenerateExtraction|TestInferConcern|TestInferTarget|TestCheckExtraction|TestParseBeadsID|TestOnceExcluding_AutoExtraction"` — all 30 test cases pass, 1.17s.
- **Knowledge:** The file had a clean seam at line ~500: pure function unit tests vs daemon OODA cycle integration tests. The duplicate mock (mockDaemonHotspotChecker) was identical to mockHotspotChecker already in the file.
- **Next:** No follow-up needed. At 500 lines and 237 lines, neither file is near the 1500-line threshold. Growth rate would need to persist ~18 more months to re-trigger.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A - novel investigation | - | - | - |

## Question

Can extraction_test.go (739 lines, +331/30d) be split along a natural seam to prevent it from reaching the 1500-line critical hotspot threshold?

## Findings

### Finding 1: Clear structural seam at line ~500

The file had two distinct test groups:

1. **Unit tests for extraction.go functions** (lines 10-500, ~490 lines):
   - TestInferTargetFilesFromIssue, TestFindCriticalHotspot, TestMatchesFilePath
   - TestGenerateExtractionTask, TestInferConcernFromIssue, TestInferTargetPackage
   - TestCheckExtractionNeeded, TestParseBeadsIDFromOutput
   - mockHotspotChecker (lines 445-452)

2. **Daemon integration tests for auto-extraction** (lines 502-739, ~237 lines):
   - mockDaemonHotspotChecker (duplicate of mockHotspotChecker!)
   - TestOnceExcluding_AutoExtraction_SpawnsExtractionWhenCriticalHotspot
   - TestOnceExcluding_AutoExtraction_SkipsWhenNoCriticalHotspot
   - TestOnceExcluding_AutoExtraction_FailsFastOnExtractionFailure
   - TestOnceExcluding_AutoExtraction_SkipsWhenNoHotspotChecker

### Finding 2: Duplicate mock eliminated

mockDaemonHotspotChecker (line 502) was functionally identical to mockHotspotChecker (line 445). The integration tests now reuse mockHotspotChecker from extraction_test.go (same package, visible across test files).

### Finding 3: proactive_extraction_test.go already exists

A separate file already exists for proactive extraction tests (267 lines), confirming the codebase pattern of splitting extraction concerns into focused files.

## Test performed

```
go test ./pkg/daemon/ -run "TestInferTargetFiles|TestFindCritical|TestMatchesFile|TestGenerateExtraction|TestInferConcern|TestInferTarget|TestCheckExtraction|TestParseBeadsID|TestOnceExcluding_AutoExtraction" -count=1 -v
```

All 30 test cases pass. Build clean (`go build ./pkg/daemon/`).

## Conclusion

Yes — the file had a natural seam between unit tests (pure function tests) and integration tests (daemon OODA cycle tests). Split extraction_test.go from 739→500 lines, new extraction_integration_test.go at 237 lines. Duplicate mock removed.
