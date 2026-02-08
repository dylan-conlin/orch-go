## Summary (D.E.K.N.)

**Delta:** Pre-spawn token estimation implemented with warning at 100k tokens and hard block at 150k tokens.

**Evidence:** All tests pass (70+ tests in spawn package), including token estimation, validation, and warning functions.

**Knowledge:** Using chars/4 for token estimation; KB context already has 80k char limit; skill content is usually the largest component.

**Next:** Close issue - feature complete and tested.

**Confidence:** High (90%) - Implementation complete, tests comprehensive, edge case of component breakdown verified.

---

# Investigation: Pre-Spawn Token Estimation to Prevent Context Overflow

**Question:** How can we prevent agents from being spawned with contexts so large they fail or run out of context during work?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** orch-go
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Token Estimation Uses Character Count / 4

**Evidence:** Claude uses roughly 4 characters per token on average for English text. This is the same ratio already used in `kbcontext.go` for KB context truncation.

**Source:** `pkg/spawn/kbcontext.go:16` - `CharsPerToken = 4`

**Significance:** Consistent estimation approach across the codebase; no new constants needed.

---

### Finding 2: Warning and Error Thresholds

**Evidence:** Claude's context window is 200k tokens. Setting warning at 100k (50%) leaves room for:
- Agent's working memory during the session
- Tool results and file contents that get added
- Back-and-forth conversation

Error threshold at 150k (75%) blocks spawns that would likely fail.

**Source:** `pkg/spawn/tokens.go:11-22`

**Significance:** Thresholds chosen to balance catching problems early vs. not blocking valid large contexts.

---

### Finding 3: Component Breakdown Enables Actionable Guidance

**Evidence:** By tracking tokens per component (template, task, skill, kb_context, server_context), warnings can suggest specific remediation:
- Large skill: "Consider using a more focused skill or --skip-artifact-check"
- Large KB context: "Consider --skip-artifact-check to reduce KB context"

**Source:** `pkg/spawn/tokens.go:88-131` (EstimateContextTokens), `pkg/spawn/tokens.go:194-220` (ShouldWarnAboutSize)

**Significance:** Users get actionable guidance, not just "context too large".

---

## Synthesis

**Key Insights:**

1. **Pre-validation catches problems early** - By checking before spawn, users get feedback immediately rather than watching an agent fail 20 minutes in.

2. **Component tracking enables debugging** - Knowing which component is largest helps users fix the issue (trim skill, reduce KB context, etc.).

3. **Warning vs Error distinction** - Warning at 100k lets work proceed but alerts user; Error at 150k prevents certain failure.

**Answer to Investigation Question:**

Pre-spawn token estimation prevents context overflow by:
1. Estimating total tokens from spawn config (task, skill, kb_context, server_context, template)
2. Warning when estimated tokens exceed 100k (leaves headroom for agent work)
3. Blocking spawn when estimated tokens exceed 150k (would likely fail)
4. Providing component breakdown so users know what to reduce

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

All tests pass, implementation is straightforward, and the approach matches existing patterns in the codebase.

**What's certain:**

- Token estimation works correctly (verified by tests)
- Integration with spawn command works (tested manually)
- Warning and error thresholds are reasonable (based on Claude's 200k window)

**What's uncertain:**

- Actual token counts may vary from estimates (char/4 is approximate)
- Thresholds may need tuning based on real-world usage

**What would increase confidence to Very High:**

- Monitor production usage to see if thresholds are correct
- Compare estimated vs actual token counts from Claude API

---

## Implementation Summary

**Files created:**
- `pkg/spawn/tokens.go` - Token estimation functions
- `pkg/spawn/tokens_test.go` - Comprehensive tests

**Files modified:**
- `pkg/spawn/kbcontext.go` - Added `CharsPerToken` constant (moved from inline)
- `cmd/orch/main.go` - Added pre-spawn validation calls

**Key functions:**
- `EstimateContextTokens(cfg *Config) *TokenEstimate` - Estimates tokens with component breakdown
- `ValidateContextSize(cfg *Config) error` - Returns error if context too large
- `ShouldWarnAboutSize(cfg *Config) (bool, string)` - Returns warning message if approaching limit

---

## References

**Files Created/Modified:**
- `pkg/spawn/tokens.go` - Core token estimation logic
- `pkg/spawn/tokens_test.go` - Tests for token estimation
- `pkg/spawn/kbcontext.go` - CharsPerToken constant
- `cmd/orch/main.go:1169` - Integration with spawn command

**Commands Run:**
```bash
# Run tests
go test ./pkg/spawn/... -v

# Build
go build ./cmd/orch/...
```

---

## Investigation History

**2025-12-25:** Investigation started
- Initial question: How to prevent context overflow in spawned agents?
- Context: Agents were running out of context when skill + KB context was very large

**2025-12-25:** Implementation complete
- Created tokens.go with estimation functions
- Added pre-spawn validation to main.go
- Fixed test failure (warning message now includes component name)

**2025-12-25:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Pre-spawn token estimation with warning at 100k and error at 150k tokens
