## Summary (D.E.K.N.)

**Delta:** Added follow-up recommendations prompting to `orch complete` - now surfaces SYNTHESIS.md recommendations before closing issues.

**Evidence:** Implementation in cmd/orch/main.go:runComplete(), all tests pass, code builds successfully.

**Knowledge:** The existing `verify.ParseSynthesis()` already extracts recommendations and next actions - just needed to wire it into the complete flow.

**Next:** Close - implementation complete and tested.

**Confidence:** High (90%) - unit tests for parsing exist; interactive prompting is straightforward.

---

# Investigation: Orch Complete Prompt Follow Up

**Question:** How to prompt orchestrator for follow-up issues when investigation recommendations exist?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: ParseSynthesis already extracts recommendations

**Evidence:** The `verify.ParseSynthesis()` function already parses SYNTHESIS.md and extracts:
- `Recommendation` field (close, spawn-follow-up, escalate, resume)
- `NextActions` array (follow-up items)
- `AreasToExplore` and `Uncertainties` for unexplored questions

**Source:** pkg/verify/check.go:139-187

**Significance:** No new parsing logic needed - just need to wire existing extraction into `runComplete()`.

---

### Finding 2: runComplete had no synthesis awareness

**Evidence:** The `runComplete()` function (cmd/orch/main.go:2468-2629) verified phase status and closed issues, but never read or surfaced SYNTHESIS.md content.

**Source:** cmd/orch/main.go:2468-2629

**Significance:** Missing opportunity to surface agent recommendations to orchestrator before closing.

---

### Finding 3: Implementation approach

**Evidence:** Added synthesis parsing after verification passes, before closing:
1. Parse SYNTHESIS.md if workspace exists
2. Check for non-close recommendations or next actions
3. Display recommendations and prompt for follow-up issue creation
4. Continue with normal close flow

**Source:** cmd/orch/main.go:2568-2618 (new code added)

**Significance:** Simple integration point that doesn't disrupt existing flow.

---

## Synthesis

**Key Insights:**

1. **Existing infrastructure sufficed** - The synthesis parsing and recommendation extraction already existed in the verify package.

2. **Minimal code change** - About 50 lines added to surface recommendations at the right moment.

3. **Interactive prompt pattern** - Follows existing patterns in the codebase (bufio.Reader for user input).

---

## Confidence Assessment

**Current Confidence:** High (90%)

**What's certain:**
- ✅ Code compiles and builds successfully
- ✅ All existing tests pass
- ✅ Synthesis parsing is well-tested in pkg/verify/check_test.go

**What's uncertain:**
- ⚠️ Interactive prompting is hard to unit test
- ⚠️ Edge cases with malformed SYNTHESIS.md files

**What would increase confidence to Very High:**
- Manual end-to-end testing with real agent completions
- Adding integration test with mock stdin

---

## References

**Files Examined:**
- pkg/verify/check.go - Synthesis parsing and verification logic
- cmd/orch/main.go - Complete command implementation
- .orch/templates/SYNTHESIS.md - Template structure

**Commands Run:**
```bash
go build ./cmd/orch/...
go test ./... -short
```

---

## Investigation History

**2025-12-25:** Investigation started
- Initial question: How to prompt for follow-up issues from investigation recommendations
- Context: SYNTHESIS.md contains recommendations that weren't being surfaced during completion

**2025-12-25:** Implementation complete
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Added follow-up prompting to `orch complete` using existing synthesis parsing

**2025-12-25:** Enhanced with interactive issue creation
- Added per-item prompting instead of batch y/n
- Now creates beads issues directly via beads.FallbackCreate
- Supports [y/N/q] for each actionable item
- Issues created with P2 priority and triage:review label
