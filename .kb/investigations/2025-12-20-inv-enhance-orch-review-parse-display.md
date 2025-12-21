**TLDR:** Enhanced orch review to parse and display D.E.K.N. sections from SYNTHESIS.md. The review command now shows condensed Synthesis Cards with TLDR, outcome, delta summary, and next actions for each completed agent. High confidence (95%) - validated with comprehensive tests.

---

# Investigation: Enhance orch review to parse and display SYNTHESIS.md

**Question:** How to enhance orch review to parse D.E.K.N. sections and display condensed Synthesis Cards?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (95%)

---

## Findings

### Finding 1: Existing Synthesis struct was minimal

**Evidence:** The original `verify.Synthesis` struct only had `TLDR` and `NextActions` fields. It didn't capture the D.E.K.N. structure (Delta, Evidence, Knowledge, Next) that agents produce.

**Source:** `pkg/verify/check.go:117-120` (before changes)

**Significance:** Need to extend the struct to capture all D.E.K.N. sections plus header metadata (Agent, Issue, Duration, Outcome).

---

### Finding 2: ParseSynthesis used simple regex for TLDR only

**Evidence:** Original `ParseSynthesis()` function used basic regex to extract TLDR and Next Actions sections, but didn't handle the full D.E.K.N. format with variants like "## Delta (What Changed)".

**Source:** `pkg/verify/check.go:137-168` (before changes)

**Significance:** Need more robust section extraction that handles variant section headers and properly terminates at section boundaries.

---

### Finding 3: review.go already had Synthesis integration point

**Evidence:** The `getCompletionsForReview()` function already called `verify.ParseSynthesis()` and the display logic showed TLDR and NextActions. This provided a clear extension point.

**Source:** `cmd/orch/review.go:100-104`

**Significance:** Adding the Synthesis Card display was straightforward - just needed to enhance what was already being parsed and displayed.

---

## Synthesis

**Key Insights:**

1. **D.E.K.N. structure is standard** - All SYNTHESIS.md files follow the Delta/Evidence/Knowledge/Next pattern, making parsing predictable.

2. **Section headers have variants** - Headers like "## Delta" can also be "## Delta (What Changed)", requiring flexible regex patterns.

3. **Condensed display is key** - Full D.E.K.N. content is too verbose for review listing; need summarization (file counts, truncated TLDR, limited next actions).

**Answer to Investigation Question:**

Enhanced orch review by:

1. Extending `verify.Synthesis` struct with D.E.K.N. fields plus header metadata
2. Rewriting `ParseSynthesis()` with robust section extraction supporting header variants
3. Adding `printSynthesisCard()` function to display condensed D.E.K.N. info
4. Adding `summarizeDelta()` to extract file/commit counts from Delta section

---

## Confidence Assessment

**Current Confidence:** High (95%)

**Why this level?**

Comprehensive tests covering both old and new SYNTHESIS.md formats pass. The implementation follows established patterns in the codebase.

**What's certain:**

- ✅ D.E.K.N. sections are correctly extracted (tested with real SYNTHESIS.md format)
- ✅ Backward compatible with minimal SYNTHESIS.md files (TLDR + Next Actions only)
- ✅ Condensed display fits review listing format

**What's uncertain:**

- ⚠️ Edge cases with malformed SYNTHESIS.md files (graceful degradation not fully tested)

**What would increase confidence to Very High (95%+):**

- Test with more real-world SYNTHESIS.md files from completed agents

---

## Implementation Recommendations

**Purpose:** Document the implementation approach taken.

### Recommended Approach ⭐

**Extend existing infrastructure** - Enhanced Synthesis struct and ParseSynthesis function rather than creating new abstractions.

**Why this approach:**

- Minimal changes to existing code
- Maintains backward compatibility
- Uses established patterns in the codebase

**Implementation sequence:**

1. Extended Synthesis struct with D.E.K.N. fields
2. Rewrote ParseSynthesis with robust section extraction
3. Added Synthesis Card display to review output
4. Added summarizeDelta helper for condensed Delta display

---

## References

**Files Examined:**

- `pkg/verify/check.go` - Synthesis struct and ParseSynthesis function
- `cmd/orch/review.go` - Review command display logic
- `.orch/workspace/og-arch-alpha-opus-synthesis-20dec/SYNTHESIS.md` - Example D.E.K.N. format

**Commands Run:**

```bash
# Run tests
go test ./pkg/verify/... -v
go test ./cmd/orch/... -v
go test ./...
```

---

## Investigation History

**2025-12-20:** Investigation started

- Initial question: How to enhance orch review to parse D.E.K.N. sections?
- Context: Need better visibility into agent work via SYNTHESIS.md parsing

**2025-12-20:** Implementation complete

- Extended Synthesis struct with D.E.K.N. fields
- Rewrote ParseSynthesis with robust section extraction
- Added Synthesis Card display to review
- All tests passing (26 tests across verify and review packages)

**2025-12-20:** Investigation completed

- Final confidence: High (95%)
- Status: Complete
- Key outcome: orch review now displays condensed Synthesis Cards with D.E.K.N. info
