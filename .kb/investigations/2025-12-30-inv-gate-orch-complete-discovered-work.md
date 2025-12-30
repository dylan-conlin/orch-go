<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented gating on orch complete to require disposition of all discovered work items from SYNTHESIS.md before closing issue.

**Evidence:** Tests pass for CollectDiscoveredWork, PromptDiscoveredWorkDisposition, and integration into runComplete; verified skip-all requires reason.

**Knowledge:** Prior implementation allowed "q to quit" without filing issues for discovered work; now completion is blocked until all items are dispositioned (file/skip/skip-all).

**Next:** Close - implementation complete with TDD approach.

---

# Investigation: Gate Orch Complete Discovered Work

**Question:** How to prevent discovered work from investigations being silently dropped at completion time?

**Started:** 2025-12-30
**Updated:** 2025-12-30
**Owner:** Agent (og-feat-gate-orch-complete-30dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Prior implementation did not gate completion

**Evidence:** Lines 3693-3768 in cmd/orch/main.go allowed "q to quit" prompt loop which continued to close the issue even without all items dispositioned.

**Source:** `cmd/orch/main.go:3745-3748` - `if response == "q" || response == "quit" { fmt.Println("Skipping remaining items."); break }`

**Significance:** This allowed recommendations from 2-day-old investigations to never become beads issues, defeating the purpose of the synthesis section.

---

### Finding 2: Synthesis already captures discovered work sections

**Evidence:** The `Synthesis` struct in `pkg/verify/check.go` already parses:
- `NextActions` from "## Next Actions" and "### Follow-up Work"
- `AreasToExplore` from "**Areas worth exploring further:**"
- `Uncertainties` from "**What remains unclear:**"

**Source:** `pkg/verify/check.go:155-177` (Synthesis struct), `pkg/verify/check.go:274-304` (extractNextActions)

**Significance:** No new parsing needed - just need to use existing fields and gate on disposition.

---

### Finding 3: --force flag should skip the gate

**Evidence:** Existing code uses `completeForce` flag to skip phase verification. The discovered work gate should also respect this flag to allow automation and edge cases.

**Source:** `cmd/orch/main.go:3600` - `if !completeForce { ... }`

**Significance:** Consistent with existing force semantics - --force bypasses all checks.

---

## Synthesis

**Key Insights:**

1. **Gating must be blocking, not prompting** - The previous "prompt and continue" pattern allowed lazy dismissal; new pattern returns error and blocks completion unless all items are handled.

2. **skip-all requires documented reason** - Allows legitimate bulk skipping (e.g., "items already tracked in epic-123") while preventing lazy "I don't want to deal with this" dismissal.

3. **Testability via io.Reader/io.Writer** - By taking input/output as parameters, the disposition logic is fully unit testable without mocking stdin.

**Answer to Investigation Question:**

Gating is achieved by:
1. Collecting all discovered work items via `CollectDiscoveredWork()`
2. Prompting for each with `PromptDiscoveredWorkDisposition()`
3. Returning error (blocking completion) if not all items are dispositioned
4. Actually filing beads issues for items marked 'y'
5. Requiring documented reason for skip-all

---

## Structured Uncertainty

**What's tested:**

- ✅ CollectDiscoveredWork extracts items from NextActions, AreasToExplore, Uncertainties
- ✅ PromptDiscoveredWorkDisposition handles y/n/s responses correctly
- ✅ skip-all requires non-empty reason
- ✅ EOF before completion returns error with AllDispositioned=false
- ✅ Integration compiles and existing tests pass

**What's untested:**

- ⚠️ Real beads issue creation during orch complete (requires beads RPC integration test)
- ⚠️ Behavior when beads.FallbackCreate fails mid-disposition

**What would change this:**

- If users report skip-all reason requirement is too burdensome → could allow empty reason
- If certain section sources should be excluded from gating → add filter

---

## Implementation Recommendations

**Recommended Approach ⭐**

**Gating with required disposition** - Block completion until all discovered work items are dispositioned (y/n/s) and skip-all requires documented reason.

**Why this approach:**
- Prevents recommendations from silently disappearing
- skip-all reason prevents lazy dismissal while allowing legitimate bulk skip
- Consistent with --force bypass semantics

**Trade-offs accepted:**
- More interactive prompts during completion
- Users must explicitly disposition each item (can use skip-all to batch)

**Implementation sequence:**
1. Create `CollectDiscoveredWork()` to gather items from synthesis
2. Create `PromptDiscoveredWorkDisposition()` with testable io.Reader/io.Writer
3. Integrate into `runComplete()` with blocking error on incomplete disposition

---

## References

**Files Created:**
- `pkg/verify/discovered_work.go` - Core types and functions
- `pkg/verify/discovered_work_test.go` - Comprehensive tests

**Files Modified:**
- `cmd/orch/main.go` - Updated runComplete to use new gating logic

**Commands Run:**
```bash
go test ./pkg/verify/... -run "TestCollect|TestPrompt|TestDiscovered" -v
# All tests pass

go build ./cmd/orch/...
# Compiles successfully
```

---

## Investigation History

**2025-12-30:** Investigation started
- Initial question: How to gate orch complete on discovered work disposition?
- Context: Design conversation identified that recommendations from investigations were getting lost

**2025-12-30:** Implementation complete
- Created pkg/verify/discovered_work.go with TDD approach
- All tests passing, integrated into runComplete
- Status: Complete
