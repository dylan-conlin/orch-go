<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch frontier` reports agents for closed issues as "stuck" because it only checks OpenCode session age, not beads issue status.

**Evidence:** Issue `orch-go-20997` (status: closed) was reported as stuck because an OpenCode session with that ID in its title still existed and was > 2h old.

**Knowledge:** Agent discovery from OpenCode sessions must validate beads status before reporting. Sessions can persist after issues close.

**Next:** Fix implemented - filter agents by checking beads status before displaying. Ready for review.

**Promote to Decision:** recommend-no - Tactical fix, not architectural pattern

---

# Investigation: Orch Frontier Reports Stuck Agents

**Question:** Why does `orch frontier` report stuck agents for closed issues, and how should this be fixed?

**Started:** 2026-01-29
**Updated:** 2026-01-29
**Owner:** Claude (architect agent)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Agent discovery doesn't validate beads status

**Evidence:** The `getActiveAndStuckAgents()` function in `cmd/orch/frontier.go` discovers agents from:
1. tmux windows (by window name)
2. OpenCode sessions (by session title)

Neither discovery path checks whether the associated beads issue is open or closed.

**Source:** `cmd/orch/frontier.go:108-193`

**Significance:** OpenCode sessions can persist after an agent completes work and the issue is closed. When the session age exceeds 2 hours, it gets categorized as "stuck" even though the work is done.

---

### Finding 2: Reproduction confirmed with `orch-go-20997`

**Evidence:** 
- `orch frontier` output showed: `STUCK (> 2h) (1) orch-go-20997 [20h 41m]`
- `bd show orch-go-20997 --json` confirmed: `"status": "closed"`
- The issue was closed but the OpenCode session with that beads ID still existed

**Source:** Command output from `orch frontier` and `bd show`

**Significance:** This confirms the bug: closed issues incorrectly appear as stuck agents.

---

### Finding 3: Beads CLI supports batch status lookup

**Evidence:** `bd show <id1> <id2> ... --json` can retrieve multiple issues in one call, returning an array of issue objects including status.

**Source:** `bd show orch-go-20997 orch-go-21014 --json` returns both issues with their statuses

**Significance:** We can efficiently filter agents by checking all discovered beads IDs in a single CLI call, minimizing performance impact.

---

## Synthesis

**Key Insights:**

1. **Session lifetime ≠ issue lifetime** - OpenCode sessions persist independently of beads issue status. An agent can complete work, close the issue, and the session remains.

2. **Stuck detection needs issue context** - The 2-hour threshold for "stuck" is only meaningful for open issues. A 20-hour-old session for a closed issue isn't stuck - it's just a leftover session.

3. **Efficient filtering is possible** - Using batch `bd show` calls allows status checking without N+1 subprocess overhead.

**Answer to Investigation Question:**

`orch frontier` reported stuck agents for closed issues because `getActiveAndStuckAgents()` only considered session age, not issue status. The fix adds a filtering step that:
1. Collects all discovered beads IDs
2. Batch-checks their status via `bd show ... --json`
3. Excludes agents whose issues are closed from the output

---

## Structured Uncertainty

**What's tested:**

- ✅ Fix filters out closed issues (verified: ran `orch frontier` after fix, `orch-go-20997` no longer appears)
- ✅ Agents without beads IDs are preserved (verified: unit test `TestFilterOpenIssueAgents_PreservesNonBeadsAgents`)
- ✅ All agent fields are preserved through filtering (verified: unit test `TestFilterOpenIssueAgents_PreservesAllFields`)

**What's untested:**

- ⚠️ Performance with large numbers of agents (batch call should be efficient but not benchmarked)
- ⚠️ Behavior when `bd` command is unavailable (fails open - shows all agents)

**What would change this:**

- If OpenCode sessions were automatically cleaned on issue close, this filter would be unnecessary
- If session age tracking improved (e.g., per-issue not per-session), the 2h threshold could be issue-aware

---

## Implementation Recommendations

**Purpose:** Document the implemented fix.

### Recommended Approach ⭐

**Filter agents by beads status after discovery** - Check if discovered agents' beads issues are closed and exclude them from output.

**Why this approach:**
- Minimal change to existing code flow
- Efficient batch lookup via single `bd show` call
- Fails open (if bd unavailable, shows all agents - better for visibility)

**Trade-offs accepted:**
- Additional subprocess call per `orch frontier` invocation
- Small latency increase (~50ms per call)

**Implementation sequence:**
1. Add `filterOpenIssueAgents()` function to filter agent slices
2. Add `getClosedIssueIDs()` helper for batch status lookup
3. Call filter in `runFrontier()` after discovery, before display

### Alternative Approaches Considered

**Option B: Clean up sessions on issue close**
- **Pros:** Removes root cause, sessions wouldn't persist
- **Cons:** Requires daemon integration, issue close detection, more complex
- **When to use instead:** Long-term solution if stale session buildup becomes a bigger problem

**Option C: Filter at display time only**
- **Pros:** Simplest change
- **Cons:** Would still show counts including closed issues before filtering display
- **When to use instead:** If performance of status lookup becomes an issue

**Rationale for recommendation:** Option A balances correctness with implementation simplicity. The batch lookup is efficient enough that latency is negligible.

---

## References

**Files Examined:**
- `cmd/orch/frontier.go` - Main frontier command and agent discovery
- `pkg/frontier/frontier.go` - Frontier state calculation
- `pkg/beads/cli_client.go` - Beads CLI client patterns

**Commands Run:**
```bash
# Verify closed issue
bd show orch-go-20997 --json

# Test fix
./orch frontier

# Run tests
go test ./cmd/orch/ -run TestFilter -v
```

**Related Artifacts:**
- **Constraint:** `orch status can show phantom agents (tmux windows where OpenCode exited)` - Related visibility issue

---

## Investigation History

**[2026-01-29 12:28]:** Investigation started
- Initial question: Why does `orch frontier` report stuck agents for closed issues?
- Context: User observed `orch-go-20997` showing as stuck despite being closed

**[2026-01-29 12:35]:** Root cause identified
- OpenCode sessions persist after issue close
- `getActiveAndStuckAgents()` doesn't validate beads status

**[2026-01-29 12:45]:** Fix implemented
- Added `filterOpenIssueAgents()` and `getClosedIssueIDs()` functions
- Verified fix works: closed issues no longer appear in STUCK section

**[2026-01-29 12:50]:** Investigation completed
- Status: Complete
- Key outcome: Bug fixed by filtering agents whose beads issues are closed
