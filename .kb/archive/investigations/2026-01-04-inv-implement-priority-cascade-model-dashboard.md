## Summary (D.E.K.N.)

**Delta:** Implemented Priority Cascade model for dashboard agent status - single determineAgentStatus() function replaces 10+ scattered status conditions.

**Evidence:** All 13 tests pass covering priority order (beads closed > Phase Complete > SYNTHESIS.md > session activity). Build succeeds.

**Knowledge:** The TTL cache eliminates the need for the line 609 optimization that skipped idle agents - we can safely fetch beads data for all agents without CPU spikes.

**Next:** Close - implementation complete with tests.

---

# Investigation: Implement Priority Cascade Model Dashboard

**Question:** Implement the Priority Cascade model for dashboard agent status as designed in .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Created determineAgentStatus() function with clear priority order

**Evidence:** New function at cmd/orch/serve_agents.go:1247-1275 implements priority cascade:
1. Beads issue closed → "completed"
2. Phase: Complete reported → "completed"
3. SYNTHESIS.md exists → "completed"
4. Session activity → sessionStatus ("active" or "idle")

**Source:** cmd/orch/serve_agents.go:1247-1275

**Significance:** Single source of truth for status determination, making the logic deterministic and debuggable.

---

### Finding 2: Removed line 609 optimization that caused idle agent bugs

**Evidence:** Changed from `if status == "active" && agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID]` to just `if agent.BeadsID != "" && !seenBeadsIDs[agent.BeadsID]`. This ensures ALL agents with beads ID are fetched, not just active ones.

**Source:** cmd/orch/serve_agents.go:605-618

**Significance:** Fixes the root cause of idle agents with Phase: Complete showing as "idle" instead of "completed". The TTL cache (5-30 second TTLs) already prevents CPU spikes.

---

### Finding 3: Consolidated status logic using the new function

**Evidence:** Refactored the agent status determination loop to:
1. Gather completion signals (issueClosed, phaseComplete, workspacePath)
2. Call determineAgentStatus() once
3. Remove the duplicate SYNTHESIS.md check that was outside the beadsIDsToFetch block

**Source:** cmd/orch/serve_agents.go:801-875

**Significance:** Eliminates duplicate code paths and ensures consistent status determination for all agents regardless of session activity.

---

## Synthesis

**Key Insights:**

1. **Priority Cascade simplifies reasoning** - Instead of 10+ scattered conditions, there's now one function with clear priority order that's easy to understand and test.

2. **The optimization was causing more bugs than CPU it saved** - The SESSION_HANDOFF.md correctly noted that fetching beads data for idle agents isn't expensive with the TTL cache in place.

3. **Workspace name fallback enables untracked agent completion** - For agents spawned with --no-track, the fallback lookup by workspace name from session title ensures SYNTHESIS.md is still found.

**Answer to Investigation Question:**

The Priority Cascade model has been successfully implemented. The dashboard agent status logic is now:
- Deterministic - Each agent gets ONE status based on highest-priority match
- Correct - Completion signals always override activity signals
- Debuggable - Single function with clear priority order
- Consistent with prior decision - Phase: Complete is authoritative over session time

---

## Structured Uncertainty

**What's tested:**

- ✅ Priority cascade logic (13 unit tests covering all combinations)
- ✅ Empty/non-existent workspace handling (2 additional tests)
- ✅ Build succeeds with changes
- ✅ All existing tests still pass (36.9s test suite)

**What's untested:**

- ⚠️ Live dashboard behavior (running server uses old binary)
- ⚠️ CPU impact of removing optimization (though TTL cache should handle it)
- ⚠️ Test agent og-inv-test-liveness-gate-04jan visibility (would require server restart)

**What would change this:**

- If CPU usage spikes significantly after deployment (would need to add more aggressive caching)
- If there are edge cases not covered by the 4-priority model

---

## References

**Files Modified:**
- cmd/orch/serve_agents.go - Core implementation
- cmd/orch/serve_agents_test.go - New tests for determineAgentStatus()

**Design Reference:**
- .kb/investigations/2026-01-04-design-dashboard-agent-status-model.md - Original design

**Commands Run:**
```bash
# Run all tests
go test -v ./cmd/orch/...

# Build verification
go build -o /dev/null ./cmd/orch/...
```

---

## Investigation History

**2026-01-04 11:26:** Implementation started
- Read design investigation for Priority Cascade model
- Created todo list with 6 tasks

**2026-01-04 11:30:** Tests written (TDD RED phase)
- 13 test cases for determineAgentStatus()
- Tests fail because function doesn't exist

**2026-01-04 11:32:** Implementation (TDD GREEN phase)
- Created determineAgentStatus() function
- All tests pass

**2026-01-04 11:35:** Refactoring
- Removed line 609 optimization
- Consolidated status logic in main loop
- Removed duplicate SYNTHESIS.md check

**2026-01-04 11:40:** Investigation completed
- Status: Complete
- Key outcome: Priority Cascade model implemented with full test coverage
