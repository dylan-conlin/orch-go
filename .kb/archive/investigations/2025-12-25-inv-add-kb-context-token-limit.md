<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added token limit to KB context formatting to prevent spawn context bloat.

**Evidence:** Implemented `MaxKBContextChars = 80000` (~20k tokens) with priority-based truncation and 7 passing tests.

**Knowledge:** KB context was previously unbounded and could explode to 60k+ tokens; truncation prioritizes constraints > decisions > investigations.

**Next:** None - implementation complete.

**Confidence:** High (90%) - Tests pass, code reviewed, follows investigation recommendations.

---

# Investigation: Add KB Context Token Limit

**Question:** How to implement a token limit for KB context to prevent spawn context bloat?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** OpenCode agent (og-feat-add-kb-context-25dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

**Extracted-From:** .kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md

---

## Findings

### Finding 1: Prior investigation identified KB context as primary bloat source

**Evidence:** Investigation `2025-12-23-inv-token-limit-explosion-headless-spawn.md` found:
- KB context can inject 2,000+ lines with broad queries
- Example: og-inv-pre-spawn-kb-22dec had 2,838-line spawn context (2,437 lines of KB matches = 86%)
- Recommended: "Add MaxKBContextTokens constant (e.g., 20k tokens)" with truncation

**Source:** `.kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md:204-206`

**Significance:** Clear prior guidance on the fix - just needed implementation.

---

### Finding 2: Existing token estimation infrastructure

**Evidence:** `pkg/spawn/tokens.go` already had:
- `CharsPerToken = 4.0` constant
- `EstimateTokens(charCount int) int` function
- `TokenEstimate` struct with threshold handling

**Source:** `pkg/spawn/tokens.go:10-90`

**Significance:** Could reuse existing infrastructure rather than duplicating.

---

### Finding 3: FormatContextForSpawn had no size limits

**Evidence:** Original function (`pkg/spawn/kbcontext.go:422-479`) simply formatted all matches without any truncation or size checking.

**Source:** `pkg/spawn/kbcontext.go:422-479` (before modification)

**Significance:** Confirmed the bloat vulnerability - no guard rails existed.

---

## Implementation

### Changes Made

1. **Added `MaxKBContextChars` constant** (pkg/spawn/kbcontext.go:34-37)
   - Set to 80,000 chars (~20k tokens using 4 chars/token)
   - Leaves room for skills, CLAUDE.md, template

2. **Added `KBContextFormatResult` struct** (pkg/spawn/kbcontext.go:51-60)
   - Returns truncation status, match counts, estimated tokens
   - Enables callers to display warnings

3. **Implemented `FormatContextForSpawnWithLimit`** (pkg/spawn/kbcontext.go:441-533)
   - Priority-based truncation: investigations (lowest) → decisions → constraints (highest)
   - Adds truncation warning to content when truncated
   - Backwards compatible: `FormatContextForSpawn` wraps with default limit

4. **Added comprehensive tests** (pkg/spawn/kbcontext_test.go:496-654)
   - Tests nil/empty handling
   - Tests no truncation when under limit
   - Tests investigation-first truncation priority
   - Tests decision-before-constraints truncation
   - Tests token estimation
   - Tests default limit wrapper

---

## Confidence Assessment

**Current Confidence:** High (90%)

**What's certain:**

- ✅ Code compiles and all 59 spawn package tests pass
- ✅ Priority-based truncation follows investigation recommendation
- ✅ Backwards compatible - existing callers unaffected

**What's uncertain:**

- ⚠️ 80k chars (~20k tokens) limit chosen based on investigation recommendation - may need tuning
- ⚠️ Truncation removes entire matches, not partial content - could be more granular

---

## References

**Files Examined:**
- pkg/spawn/kbcontext.go - Main implementation
- pkg/spawn/tokens.go - Existing token estimation
- pkg/spawn/kbcontext_test.go - Added tests

**Commands Run:**
```bash
go build ./...
go test ./pkg/spawn/... -v
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-23-inv-token-limit-explosion-headless-spawn.md - Source of recommendations

---

## Investigation History

**2025-12-25 18:45:** Investigation started
- Initial question: How to add KB context token limit
- Context: Prior investigation recommended this as item 2 of 3-part fix

**2025-12-25 19:00:** Implementation completed
- Added MaxKBContextChars, FormatContextForSpawnWithLimit, tests
- All tests passing

**2025-12-25 19:05:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: KB context now limited to ~20k tokens with priority-based truncation
