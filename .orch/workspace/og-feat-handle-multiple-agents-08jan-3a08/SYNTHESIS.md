# Session Synthesis

**Agent:** og-feat-handle-multiple-agents-08jan-3a08
**Issue:** orch-go-ou4r8
**Duration:** 2026-01-08
**Outcome:** success

---

## TLDR

Added new "awaiting-cleanup" status to distinguish completed-but-orphaned agents from truly crashed/dead agents. This helps orchestrators prioritize attention: dead agents need investigation, awaiting-cleanup agents just need `orch complete`.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-08-inv-handle-multiple-agents-same-beads.md` - Investigation documenting the problem and solution

### Files Modified
- `cmd/orch/serve_agents.go` - Added "awaiting-cleanup" status to `determineAgentStatus()` function
- `cmd/orch/serve_agents_test.go` - Added test cases for new status
- `web/src/lib/stores/agents.ts` - Added awaiting-cleanup to AgentState type and derived store
- `web/src/lib/components/agent-card/agent-card.svelte` - Added amber styling and cleanup icon for new status
- `web/src/lib/components/agent-detail/agent-detail-panel.svelte` - Added status color for awaiting-cleanup
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Added Awaiting Cleanup section

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- Status determination uses Priority Cascade model (serve_agents.go:1038-1056)
- Dead status was set at line 446-447 based on 3-minute activity timeout
- Phase: Complete and SYNTHESIS.md checks were correctly upgrading status to "completed"
- The gap was: dead agents with completion signals showed as "dead" not distinguishable from crashed

### Tests Run
```bash
go test ./cmd/orch/... -run "TestDetermineAgentStatus" -v
# PASS: 16 test cases including new awaiting-cleanup scenarios
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-08-inv-handle-multiple-agents-same-beads.md` - Full investigation

### Decisions Made
- "awaiting-cleanup" is returned when (Phase: Complete OR SYNTHESIS.md exists) AND session is dead
- Agents with active/idle sessions that have completion signals return "completed" (not awaiting-cleanup)
- Beads issue closed always returns "completed" (highest priority)

### Constraints Discovered
- Cross-repo workspace detection relies on workspace cache scanning session directories
- If session's Directory is wrong (OpenCode --attach bug), workspace may not be found
- Phase: Complete via beads comments is more reliable than SYNTHESIS.md detection for cross-repo agents

### Externalized via `kb quick`
- (recommend) `kb quick decide "awaiting-cleanup distinguishes completed-orphan from crashed" --reason "Helps orchestrator prioritize: awaiting-cleanup needs orch complete, dead needs investigation"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file completed
- [x] Ready for `orch complete orch-go-ou4r8`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Auto-cleanup on respawn (closing old session when spawning on same beads ID)
- Dashboard grouping by beads ID (showing related agents together)
- Daemon auto-complete (if agent has SYNTHESIS.md + idle, could auto-complete)

**Areas worth exploring further:**
- Cross-repo workspace detection reliability
- Why some agents don't report Phase: Complete

**What remains unclear:**
- Whether "awaiting-cleanup" should appear in Active section or only Needs Attention

---

## Session Metadata

**Skill:** feature-impl
**Model:** Claude
**Workspace:** `.orch/workspace/og-feat-handle-multiple-agents-08jan-3a08/`
**Investigation:** `.kb/investigations/2026-01-08-inv-handle-multiple-agents-same-beads.md`
**Beads:** `bd show orch-go-ou4r8`
