# Session Synthesis

**Agent:** og-work-redesign-knowledge-tree-16feb-498d
**Issue:** orch-go-r3zy
**Duration:** 2026-02-16T09:00:00 → 2026-02-16T10:30:00
**Outcome:** success

---

## Plain-Language Summary

Designed a 6-task epic to fix the work-graph dashboard's visibility problems. The critical finding is that `/api/beads/graph` - the endpoint the frontend depends on for issue dependency data - doesn't exist in the backend, making the work-graph page non-functional for dependency visualization. The attention store (per-issue verification signals) is also a stub. The epic addresses all 6 identified problems: scattered dep chains, buried in-progress items, invisible verification gates, missing per-issue verification badges, and unclear sort logic. Tasks are ordered by dependency: backend endpoint first, then frontend wiring, then UI improvements.

## Verification Contract

See: `VERIFICATION_SPEC.yaml` in this workspace directory.

Key outcomes:
- Epic orch-go-nbem created with 6 child tasks
- Dependencies set: Tasks 3,6 blocked by Task 1 (graph endpoint); Task 5 blocked by Task 2 (attention store)
- Problem 3 (decision/issue hierarchy inversion) explicitly deferred - needs separate investigation

---

## TLDR

Designed and decomposed the work-graph dashboard redesign into an epic (orch-go-nbem) with 6 implementation tasks addressing all 6 identified problems. Critical discovery: the `/api/beads/graph` endpoint is missing from the backend, and the attention store is a stub - both must be built before any UI improvements can work.

---

## Delta (What Changed)

### Issues Created
- `orch-go-nbem` - Epic: Redesign work-graph dashboard
- `orch-go-pcyb` - Build /api/beads/graph endpoint (P1, no blockers)
- `orch-go-nylj` - Wire attention store to /api/attention (P1, no blockers)
- `orch-go-pipi` - Add status-first partitioning for in_progress (P1, blocked by pcyb)
- `orch-go-sk88` - Surface verification tracker in daemon bar (P1, no blockers)
- `orch-go-nibk` - Show verification badges on issues (P2, blocked by nylj)
- `orch-go-25b6` - Add dep-chain grouping mode (P2, blocked by pcyb)

### Dependencies
```
orch-go-pcyb (graph endpoint)
  ├── blocks → orch-go-pipi (status partition)
  └── blocks → orch-go-25b6 (dep-chain grouping)

orch-go-nylj (attention store)
  └── blocks → orch-go-nibk (verification badges)

orch-go-sk88 (verification gate UI) - independent
```

---

## Evidence (What Was Observed)

### Critical Gap: Missing /api/beads/graph
- Frontend: `web/src/lib/stores/work-graph.ts:122` calls `GET /api/beads/graph`
- Backend: No handler registered in `cmd/orch/serve.go` - only `/api/beads` (stats) and `/api/beads/ready`
- Impact: Work-graph page cannot show dependency data or edges
- Data available: `bd list --json` returns issues with `dependencies[]` array containing `{issue_id, depends_on_id, type}` - sufficient to build the graph endpoint

### Attention Store is a Stub
- File: `web/src/lib/stores/attention.ts`
- Status: `fetch()` is a no-op that logs `console.warn('attention store: fetch not implemented')`
- Backend exists: `/api/attention` handler in `serve_attention.go` returns rich signals (verify, stuck, unblocked, etc.)
- The store exports `CompletedIssue[]` but the page expects `{completedIssues, signals}` shape

### Verification Tracker Not in API Response
- Daemon status file (`~/.orch/daemon-status.json`) has `verification` field with `is_paused`, `completions_since_verification`, `threshold`
- But `handleDaemon()` in `serve_system.go` doesn't include verification data in `DaemonAPIResponse`
- Frontend `DaemonStatus` interface has no verification fields
- Result: Verification gate state invisible in dashboard

### Current Sort/Group Logic
- Root sort: priority (0→4) then ID (alphabetical)
- Children sort: ID only
- Group modes: priority, area, effort (label-based)
- No status-aware sorting - in_progress treated same as open

### Data Shape from bd list --json
```json
{
  "id": "orch-go-xxxx",
  "title": "...",
  "status": "open|in_progress|closed|blocked",
  "priority": 0-4,
  "issue_type": "task|bug|feature|epic|question",
  "labels": ["triage:ready", "area:dashboard"],
  "dependencies": [{
    "issue_id": "orch-go-xxxx",
    "depends_on_id": "orch-go-yyyy",
    "type": "blocks"
  }],
  "dependency_count": N,
  "dependent_count": N
}
```

---

## Knowledge (What Was Learned)

### Design Forks Navigated

**Fork 1: Graph endpoint implementation**
- Options: CLI subprocess vs RPC client
- Decision: CLI with RPC fallback (existing pattern in serve_beads.go)
- Substrate: Beads Integration Architecture model specifies "RPC-first with CLI fallback"

**Fork 2: In-progress surfacing approach**
- Options: A) Dedicated section at top, B) Sort modifier within groups
- Decision: Option A - dedicated section, matching "Ready to Complete" pattern
- Reasoning: Explicit visual section is more discoverable than subtle sort changes

**Fork 3: Verification gate display**
- Options: A) New banner + per-issue badges, B) Inline in daemon bar + attention badges
- Decision: Option B - verification is daemon state (daemon bar), per-issue is attention (existing badge system)
- Substrate: Dashboard guide says "Priority Cascade Model" for agent status

**Fork 4: Dep-chain grouping**
- Options: A) New grouping mode, B) Visual connectors, C) Inline related section
- Decision: Option A - consistent with existing group modes
- Reasoning: Most discoverable, fits existing GroupByDropdown pattern

**Fork 5: Decision/issue hierarchy**
- Decision: Deferred - needs separate investigation
- Reasoning: Inverting hierarchy requires understanding how decisions relate to issues across the full system, not just the dashboard. The current parent-child model is used by beads, spawn context, and completion verification.

### Constraints Discovered
- `bd list --json` returns dependency data inline (no separate graph query needed)
- Attention store shape mismatch: store returns `CompletedIssue[]` but page expects `{completedIssues, signals}`
- Daemon status file has verification data but API response doesn't include it

### Externalized via `kb`
- No new kb entries needed - design captured in epic/issues

---

## Next (What Should Happen)

**Recommendation:** close

### Parallel Execution Plan

Three tasks can start immediately (no blockers):
1. **orch-go-pcyb** - Build /api/beads/graph endpoint (unblocks pipi + 25b6)
2. **orch-go-nylj** - Wire attention store (unblocks nibk)
3. **orch-go-sk88** - Surface verification tracker in daemon bar

After those complete:
4. **orch-go-pipi** - Status-first partitioning (needs graph endpoint)
5. **orch-go-nibk** - Verification badges (needs attention store)
6. **orch-go-25b6** - Dep-chain grouping (needs graph endpoint)

### If Close
- [x] All deliverables complete (epic + 6 child tasks with deps)
- [x] Design forks documented
- [x] Ready for `orch complete orch-go-r3zy`

---

## Unexplored Questions

- **Decision/issue hierarchy inversion**: How should the system handle the relationship between decisions and the issues they produce? Currently decisions can only be children of issues, but often decisions produce issues. This may need a new edge type in beads.
- **SSE cycling on knowledge-tree page**: Probe 2026-02-15 identified root cause (full tree replacement wipes expansion state) but fix isn't applied to work-graph yet. May interact with the graph data updates.
- **Graph endpoint performance**: With 30+ open issues, fetching all with deps via CLI could be slow. May need RPC path or batch Show() calls. Monitor after implementation.

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-work-redesign-knowledge-tree-16feb-498d/`
**Beads:** `bd show orch-go-r3zy`
