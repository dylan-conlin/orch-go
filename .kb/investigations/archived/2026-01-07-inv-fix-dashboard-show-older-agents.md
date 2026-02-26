<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Fixed dashboard to show agents older than 2h instead of hiding them; added `is_stale` boolean field to mark these agents visually.

**Evidence:** Go code compiles, frontend builds, related tests pass. Changed `continue` to `isStale := true` assignment at serve_agents.go:328-335.

**Knowledge:** The 2h beads fetch threshold was performance optimization (avoid 400+ RPC calls). Solution preserves optimization by skipping beads fetch for stale agents while still displaying them.

**Next:** Deploy and verify in browser. Stale agents should appear in Archive section with 📦 indicator.

**Promote to Decision:** recommend-no - Bug fix implementation, not architectural decision.

---

# Investigation: Fix Dashboard Show Older Agents

**Question:** How should the dashboard handle agents older than 2h that currently get excluded via `continue`?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Agent spawned via orch
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current behavior excludes old agents entirely

**Evidence:** In `serve_agents.go:328-335`, sessions older than `beadsFetchThreshold` (2 hours) were skipped with `continue`, causing them to not appear in the API response at all.

**Source:** `cmd/orch/serve_agents.go:328-335`
```go
// MAJOR OPTIMIZATION: Skip sessions older than beadsFetchThreshold entirely.
if timeSinceUpdate > beadsFetchThreshold {
    continue
}
```

**Significance:** This was a performance optimization to reduce beads RPC calls from 400+ to ~20-50, but it had the side effect of hiding historical agents from the dashboard.

---

### Finding 2: Frontend already has progressive disclosure infrastructure

**Evidence:** The frontend (`web/src/lib/stores/agents.ts:309-324`) already categorizes agents into Active, Recent (< 24h), and Archive (> 24h) sections. Stale agents would naturally fall into the Archive section.

**Source:** `web/src/lib/stores/agents.ts:303-324`

**Significance:** No need to create new UI sections - stale agents will use existing Archive section.

---

### Finding 3: IsStale pattern already exists in review command

**Evidence:** The `review.go` file already uses an `IsStale` boolean field for workspace completion tracking with a 24h threshold.

**Source:** `cmd/orch/review.go:125`, `review.go:439-445`

**Significance:** Adding `IsStale` to API response follows existing patterns in the codebase.

---

## Synthesis

**Key Insights:**

1. **Performance vs Visibility trade-off** - The 2h beadsFetchThreshold exists for performance. We can preserve this optimization by skipping beads fetch for stale agents while still including them in the response.

2. **Visual differentiation is important** - Users need to know stale agents have incomplete data. A visual indicator (📦 with tooltip) clearly communicates that phase/task data may be outdated.

3. **Existing infrastructure handles it** - The Archive section in progressive disclosure already handles old agents. No new UI sections needed.

**Answer to Investigation Question:**

Dashboard should include agents older than 2h in the API response with `is_stale: true`, skip beads fetch for them (preserving performance), and display a visual indicator (📦 badge with tooltip) so users know the data may be incomplete.

---

## Structured Uncertainty

**What's tested:**

- ✅ Go code compiles (verified: `go build ./cmd/orch` succeeded)
- ✅ Frontend builds (verified: `bun run build` succeeded)
- ✅ Related tests pass (verified: ran `go test ./cmd/orch/... -run "Stale|Agent"`)

**What's untested:**

- ⚠️ Browser visual verification (server needs restart to pick up new binary)
- ⚠️ Performance impact of including stale agents (likely minimal since beads fetch is skipped)

**What would change this:**

- Finding would be wrong if stale agents cause API response size issues (unlikely given JSON overhead is small)
- Implementation would need revision if visual indicator causes confusion vs clarification

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add `is_stale` boolean field** - Include stale agents in API response with `is_stale: true`, skip beads fetch, and add visual indicator in UI.

**Why this approach:**
- Minimal API change (single boolean field)
- Preserves performance optimization (beads fetch skipped for stale)
- Clear visual differentiation for users

**Trade-offs accepted:**
- Stale agents won't have accurate phase/task data (acceptable since they're old)
- An extra field in API response (minimal overhead)

**Implementation sequence:**
1. Add `IsStale bool` to `AgentAPIResponse` struct
2. Change `continue` to `isStale := true` assignment
3. Skip adding stale agents to `beadsIDsToFetch`
4. Add `is_stale?: boolean` to frontend Agent interface
5. Add visual indicator in agent-card component

---

## References

**Files Examined:**
- `cmd/orch/serve_agents.go` - Main API handler, contains the 2h exclusion logic
- `web/src/lib/stores/agents.ts` - Agent interface and progressive disclosure stores
- `web/src/lib/components/agent-card/agent-card.svelte` - Agent card UI component

**Commands Run:**
```bash
# Verify Go compilation
go build ./cmd/orch

# Verify frontend builds
cd web && bun run build

# Run related tests
go test ./cmd/orch/... -run "Stale|Agent" -v
```

**Related Artifacts:**
- **Decision:** None (tactical fix)
- **Investigation:** `.kb/investigations/2026-01-04-design-dashboard-agent-status-model.md` - Dashboard status model design

---

## Investigation History

**2026-01-07 Phase: Planning:** Investigation started
- Initial question: How to show older agents without hiding them?
- Context: Bug report - agents older than 2h excluded via `continue`

**2026-01-07 Phase: Implementing:** Solution designed and implemented
- Added `is_stale` boolean field approach
- Modified backend, frontend types, and UI components

**2026-01-07 Phase: Complete:** Investigation completed
- Status: Complete
- Key outcome: Fixed by including stale agents with visual indicator
