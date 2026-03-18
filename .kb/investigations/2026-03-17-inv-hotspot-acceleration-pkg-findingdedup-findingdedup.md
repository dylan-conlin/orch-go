## Summary (D.E.K.N.)

**Delta:** pkg/findingdedup/findingdedup_test.go hotspot is a false positive — the file was created 2 days ago in a single commit (289 lines born at creation), not accreting over time.

**Evidence:** `git log --follow` shows exactly 1 commit (3f20b8fc3, 2026-03-15). File is 289 lines covering 7 test functions + 2 helpers for a 600-line implementation. All tests pass.

**Knowledge:** Hotspot acceleration detector flags high absolute line growth in 30-day windows but doesn't distinguish new-file creation from accretion. Files born recently will always appear as hotspots until the window rolls past their creation date.

**Next:** Close as false positive. No extraction needed — file is cohesive, well under 1500-line boundary, and has no accretion pattern.

**Authority:** implementation - Straightforward false-positive classification, no cross-boundary impact.

---

# Investigation: Hotspot Acceleration — pkg/findingdedup/findingdedup_test.go

**Question:** Is pkg/findingdedup/findingdedup_test.go (+289 lines/30d) a genuine accretion hotspot requiring extraction?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** orch-go-c799l
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: File was created in a single commit 2 days ago

**Evidence:** `git log --oneline --follow -- pkg/findingdedup/findingdedup_test.go` shows exactly one commit: `3f20b8fc3 2026-03-15 feat: add finding deduplication detector — anti-coherence check for KB (orch-go-oh66x)`. The entire 289 lines were born in this commit.

**Source:** git log for pkg/findingdedup/findingdedup_test.go

**Significance:** The +289 lines/30d metric is 100% attributable to file creation, not accretion. There is no multi-commit growth pattern.

---

### Finding 2: File is well-structured and cohesive at 289 lines

**Evidence:** The test file contains 7 test functions (TestExtractFindings_InvestigationFormat, TestExtractFindings_SynthesisKnowledge, TestExtractFindings_SynthesisEvidence, TestTokenize, TestJaccardSimilarity, TestFindDuplicateClusters, TestFindDuplicateClusters_BelowMinSize, TestScanDir) and 2 small helpers (writeFile, containsStr/findSubstring). Each test covers a distinct function in the 600-line implementation file. The package has only 2 files total.

**Source:** pkg/findingdedup/findingdedup_test.go (289 lines), pkg/findingdedup/findingdedup.go (599 lines)

**Significance:** At 289 lines, the file is well under the 1500-line accretion boundary. The test-to-implementation ratio (~0.48) is healthy. No extraction would improve cohesion — all tests belong together.

---

### Finding 3: All tests pass

**Evidence:** `go test ./pkg/findingdedup/ -v -count=1` → PASS, 7/7 tests passing in 0.37s.

**Source:** go test output

**Significance:** The test suite is healthy and functional. No quality concerns that would motivate restructuring.

---

## Synthesis

**Key Insights:**

1. **False positive from new-file creation** - The hotspot acceleration detector measures absolute line growth over a 30-day window. Newly created files always appear as hotspots until the window rolls past their creation date. This is a known limitation.

2. **No action needed** - The file is small (289 lines), cohesive (single package, all related tests), well under the accretion boundary (1500 lines), and has only 1 commit. Extraction would be over-engineering.

**Answer to Investigation Question:**

No, this is not a genuine accretion hotspot. The +289 lines/30d growth is entirely from the file being created on 2026-03-15 in a single commit. The file is healthy, cohesive, and well under any extraction threshold. This is a false positive — the hotspot detector flagged it because it can't distinguish new-file creation from accretion.

---

## Structured Uncertainty

**What's tested:**

- ✅ Git history confirms single-commit creation (verified: git log --follow)
- ✅ All 7 tests pass (verified: go test -v)
- ✅ File structure is cohesive — 7 tests + 2 helpers for 1 implementation file (verified: read both files)

**What's untested:**

- ⚠️ Whether the hotspot detector should filter out new-file creation (not investigated — that's a separate improvement)

**What would change this:**

- Finding would be wrong if additional commits existed adding lines after creation (but git log confirms only 1 commit)

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Close as false positive, no extraction | implementation | Stays inside scope — classification decision, no architectural impact |

### Recommended Approach ⭐

**Close as false positive** - No action needed on the test file itself.

**Why this approach:**
- File is 289 lines, well under 1500-line accretion boundary
- Single commit, no growth pattern
- Tests are cohesive and all pass

**Trade-offs accepted:**
- The hotspot detector will continue to flag new files until the 30-day window passes
- Acceptable because this is a known limitation, not worth fixing for infrequent false positives

---

## References

**Files Examined:**
- `pkg/findingdedup/findingdedup_test.go` - Full read to assess structure and cohesion
- `pkg/findingdedup/findingdedup.go` - Full read to understand test-to-implementation ratio

**Commands Run:**
```bash
# Git history for the test file
git log --oneline --follow -- pkg/findingdedup/findingdedup_test.go
# Result: 1 commit (3f20b8fc3, 2026-03-15)

# Run tests
go test ./pkg/findingdedup/ -v -count=1
# Result: PASS, 7/7 tests, 0.37s

# Line counts
wc -l pkg/findingdedup/findingdedup_test.go pkg/findingdedup/findingdedup.go
# Result: 289 + 599 = 888 total
```
