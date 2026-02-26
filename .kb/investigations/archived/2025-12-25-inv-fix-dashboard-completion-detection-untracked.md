## Summary (D.E.K.N.)

**Delta:** Untracked agents now show as "completed" when their workspace has SYNTHESIS.md, fixing the forever-active bug.

**Evidence:** Tests pass for checkWorkspaceSynthesis function; build succeeds; all existing tests pass.

**Knowledge:** Untracked agents have fake beads IDs (e.g., `orch-go-untracked-12345`) which won't match real beads issues, so Phase: Complete check fails silently - workspace-based detection is the fallback.

**Next:** Close - implementation complete with tests.

**Confidence:** High (90%) - implementation follows established pattern, all tests pass.

---

# Investigation: Fix Dashboard Completion Detection Untracked

**Question:** How to detect completion for untracked agents (--no-track) in the dashboard when there's no beads issue to check Phase: Complete?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Untracked agents use fake beads IDs

**Evidence:** When `--no-track` is used, beads ID is generated as `{project}-untracked-{timestamp}` (see main.go:1521).

**Source:** cmd/orch/main.go:1519-1521

**Significance:** These fake IDs won't match any real beads issues, so the Phase: Complete lookup at serve.go:517-526 returns nothing.

---

### Finding 2: Workspace SYNTHESIS.md is authoritative for completion

**Evidence:** The existing code already uses SYNTHESIS.md to mark workspaces as completed in the workspace scanning loop (serve.go:409-411).

**Source:** cmd/orch/serve.go:408-411

**Significance:** SYNTHESIS.md is already the established signal for full-tier agent completion. Adding this check to active agents follows the same pattern.

---

### Finding 3: Fix requires workspace lookup for active agents

**Evidence:** Active agents are added to the list before the beads batch fetch. The fix needs to find their workspace path using `findWorkspaceByBeadsID` after the beads phase check fails.

**Source:** cmd/orch/serve.go:287-338 (active sessions loop), cmd/orch/serve.go:516-526 (beads phase check)

**Significance:** The fix location is after the beads phase check, as a fallback for agents not yet marked completed.

---

## Synthesis

**Key Insights:**

1. **Untracked agents fall through all completion checks** - They have beads IDs that don't exist in beads, so Phase: Complete is never found.

2. **SYNTHESIS.md is already the standard completion signal** - The workspace scanning loop uses it; we just need to extend this to active sessions.

3. **Minimal change required** - Just add a fallback check after the beads phase check to look for SYNTHESIS.md in the workspace.

**Answer to Investigation Question:**

The fix is to add a workspace-based completion check after the beads Phase: Complete check. For agents not yet marked completed, look up their workspace using `findWorkspaceByBeadsID` and check if SYNTHESIS.md exists. If it does, mark the agent as "completed".

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Implementation follows established pattern (SYNTHESIS.md check already used for workspace scanning), all tests pass, build succeeds.

**What's certain:**

- ✅ The implementation correctly checks for SYNTHESIS.md
- ✅ Tests verify the checkWorkspaceSynthesis function works correctly
- ✅ All existing tests continue to pass

**What's uncertain:**

- ⚠️ Performance impact of workspace lookup for each active agent not yet tested under load
- ⚠️ Edge case: agent with SYNTHESIS.md but not truly complete (unlikely but possible)

**What would increase confidence to Very High:**

- End-to-end test with actual untracked agent showing as completed
- Performance benchmarks with many active agents

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add workspace SYNTHESIS.md check as fallback** - After beads phase check fails to mark agent as completed, check workspace for SYNTHESIS.md.

**Why this approach:**
- Uses existing SYNTHESIS.md pattern already established in codebase
- Minimal code change (8 lines)
- Handles both tracked and untracked agents uniformly

**Trade-offs accepted:**
- Extra filesystem check for each non-completed agent
- Acceptable because workspace lookup is already done elsewhere and is fast

**Implementation sequence:**
1. Add `checkWorkspaceSynthesis` helper function
2. Add fallback check after beads phase check
3. Add unit tests for the helper function

### Alternative Approaches Considered

**Option B: Check for "untracked" in beads ID and skip beads lookup**
- **Pros:** Faster, no filesystem check
- **Cons:** Doesn't work for tracked agents that somehow don't have Phase: Complete
- **When to use instead:** Never - this is less robust

**Rationale for recommendation:** Using SYNTHESIS.md is more robust and handles all edge cases.

---

## References

**Files Examined:**
- cmd/orch/serve.go - handleAgents function, agent status derivation
- cmd/orch/main.go - findWorkspaceByBeadsID, untracked ID generation

**Commands Run:**
```bash
# Run tests
go test -v ./cmd/orch/ -run TestCheckWorkspaceSynthesis

# Full test suite
go test ./...
```

---

## Investigation History

**2025-12-25 12:00:** Investigation started
- Initial question: How to detect completion for untracked agents?
- Context: Bug report that untracked agents show as "active" forever

**2025-12-25 12:15:** Found root cause
- Untracked agents have fake beads IDs that don't match real issues

**2025-12-25 12:30:** Implementation complete
- Added checkWorkspaceSynthesis function
- Added fallback check after beads phase check
- All tests pass

**2025-12-25 12:45:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Added workspace SYNTHESIS.md check as fallback for untracked agent completion detection
