## Summary (D.E.K.N.)

**Delta:** kb_ask_test.go hotspot acceleration is a false positive — 100% of additions (367/367) are birth churn from extracting kb_test.go (670 lines) into 4 domain-cohesive test files.

**Evidence:** `git log --numstat` shows exactly 1 commit (d25f9ccd4, Mar 10 extraction of kb.go monolith). The same commit deleted 670 lines from kb_test.go. File is 367 lines — well under any threshold.

**Knowledge:** Same birth-churn false positive pattern seen in status_infra.go, pidlock.go, mock_test.go, and many others. Extraction targets are the fix for hotspots, not hotspots themselves.

**Next:** Close as false positive. No action needed.

**Authority:** implementation - Tactical classification of false positive, no architectural impact.

---

# Investigation: Hotspot Acceleration — cmd/orch/kb_ask_test.go

**Question:** Is the +367 lines/30d acceleration in kb_ask_test.go a genuine hotspot risk or a false positive?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** orch-go-ix5mk
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation (same birth-churn FP pattern as prior hotspot investigations) | - | - | - |

## TLDR

kb_ask_test.go was born on 2026-03-10 from extracting the 670-line kb_test.go into 4 files. Its entire 367-line existence was counted as 30-day growth. This is a false positive — the file is the product of a hotspot fix, not a hotspot.

## What I Tried

1. `git log --numstat --follow -- cmd/orch/kb_ask_test.go` — single commit, 367 additions, 0 deletions
2. `git log --numstat --follow -- cmd/orch/kb_test.go` — same commit deleted 670 lines from kb_test.go
3. `git show d25f9ccd4 --stat` — extraction commit split kb.go (1138→280) and kb_test.go into 4+4 files
4. `go test ./cmd/orch/ -run "TestKBContext|TestWriteContext|TestGenerateSlug|TestBuildSynthesis|TestReadArtifact|TestExtractKeywords|TestHasResults"` — all 7 tests pass

## What I Observed

- **Single commit:** d25f9ccd4 (2026-03-10) — "refactor: extract kb.go (1138→280 lines) into 4 cohesive files"
- **Birth churn:** 367/367 lines (100%) from file creation
- **Net growth:** 0 new test lines — tests moved from kb_test.go to kb_ask_test.go
- **File size:** 367 lines, well under 800-line advisory threshold
- **Tests healthy:** 7 test functions, all passing (0.39s)

## Test Performed

Verified via git history that the file was created by extraction, not organic growth. Cross-checked that kb_test.go shrank by 670 lines in the same commit. Ran the tests to confirm the file is healthy.

## Conclusion

**False positive.** The hotspot detector counted file-birth from extraction as organic growth. kb_ask_test.go has had zero modifications since creation and is less than half the advisory threshold. No action needed.
