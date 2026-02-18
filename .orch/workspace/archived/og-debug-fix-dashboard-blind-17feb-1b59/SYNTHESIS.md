# Session Synthesis

**Agent:** og-debug-fix-dashboard-blind-17feb-1b59
**Issue:** orch-go-1022
**Outcome:** success

---

## Plain-Language Summary

Three compounding bugs made tmux-spawned agents invisible in the dashboard. The workspace scan's duplicate check only matched on workspace name (not beads_id), so tmux agents with emoji-prefixed IDs were duplicated as "completed" entries. Beads enrichment was skipped for workspace-scanned entries, so phase was always null. And the graph API never populated `active_agent`, preventing the work-graph tree from showing which agent works on which issue. All three are now fixed.

## TLDR

Fixed three bugs in serve_agents.go and serve_beads.go that made tmux agents invisible in the dashboard: added beads_id to workspace scan dedup check, enabled beads enrichment for recent workspace entries, and added active_agent field to graph nodes with agent data populated from OpenCode sessions + tmux windows + beads comments.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for verification steps.

Key outcomes:
- `go build ./cmd/orch/` compiles without errors
- `go vet ./cmd/orch/` passes
- `go test ./cmd/orch/` all tests pass (2.3s)
- No duplicate beads_ids in `/api/agents` response for tmux agents
- `/api/beads/graph` nodes have `active_agent` populated for in-progress issues

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve_agents.go` - Fixed workspace scan duplicate check to include beads_id matching; moved beads_id extraction before dedup; replaced enrichment skip with conditional enrichment for recent workspaces
- `cmd/orch/serve_beads.go` - Added ActiveAgentInfo struct and active_agent field to GraphNode; added buildActiveAgentMap() function; added imports for opencode, tmux, verify packages

### Commits
- (pending) fix: dashboard blind to tmux agents - three compounding bugs

---

## Evidence (What Was Observed)

- Probe documented the root cause chain: workspace scan claims agent first → tmux dedup skips → beads enrichment never runs → phase=null, status=completed, window=null
- Old binary confirmed 9 duplicate beads_ids in `/api/agents` response
- Old binary confirmed all in-progress graph nodes have `active_agent=NONE`
- `verify.Comment` is an alias for `beads.Comment` - direct type compatibility confirmed
- `extractBeadsIDFromWorkspace`, `extractBeadsIDFromTitle`, `extractDateFromWorkspaceName` all available in package scope

### Tests Run
```bash
go build ./cmd/orch/  # OK
go vet ./cmd/orch/    # OK
go test ./cmd/orch/   # 2.314s, all passing
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Workspace scan's alreadyIn check only compared workspace directory name to agent ID, missing agents whose IDs have different formats (tmux emoji prefix, [beads-id] suffix)
- Beads enrichment performance guard must limit to recent workspaces (beadsFetchThreshold) to avoid 600+ historical workspace fetch
- The probe's claim about scan ordering was incorrect - tmux scan already runs before workspace scan in the code, but the duplicate check was the actual bug

### Decisions Made
- Used `wsBeadsID` variable name to avoid shadowing issues with existing `beadsID` usage patterns
- Performance guard: only enrich workspace entries within `beadsFetchThreshold` (date parsed from workspace name) to avoid CPU spikes
- `buildActiveAgentMap()` queries both OpenCode sessions and tmux windows for comprehensive agent coverage

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (3 bug fixes)
- [x] Tests passing
- [x] Ready for `orch complete orch-go-1022`

### Follow-up (nice-to-have, not blocking)
- WIP store's `setRunningAgents()` is still stubbed (wip.ts) - this affects ALL agents' work-graph tree display, not just tmux. Tracked separately.

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-fix-dashboard-blind-17feb-1b59/`
**Beads:** `bd show orch-go-1022`
