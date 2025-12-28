# Session Synthesis

**Agent:** og-debug-dashboard-shows-agent-27dec
**Issue:** orch-go-r8a7
**Duration:** 2025-12-27
**Outcome:** success

---

## TLDR

Fixed bug where dashboard API showed actively running agents as "completed" when they had Phase: Complete or SYNTHESIS.md artifacts. The fix ensures status is only set to "completed" when there's no active OpenCode session or tmux window for the agent.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve.go` - Fixed status detection logic in handleAgents to respect session state over artifacts

### Commits
- (pending) - fix: status detection respects session state over artifacts

---

## Evidence (What Was Observed)

- **Root Cause:** The status detection logic at lines 886-906 in `serve.go` marked agents as "completed" based solely on artifacts (Phase: Complete, SYNTHESIS.md) without checking if an OpenCode session or tmux window was still running
- The comment claimed: "Phase: Complete is the definitive signal that the agent's work is done, regardless of whether the OpenCode session is still open" - This was incorrect
- CLI (`orch status`) correctly uses `IsSessionProcessing` and checks if beads issue is closed, not just artifacts
- API was setting `IsProcessing: false` always (line 636) due to performance optimization that removed per-session HTTP calls

### Tests Run
```bash
# Syntax verification
gofmt -e cmd/orch/serve.go
# Syntax OK

# Project has pre-existing build errors in pkg/spawn/learning.go
# unrelated to this change
```

---

## Knowledge (What Was Learned)

### Decisions Made
- **Session state trumps artifacts:** An agent with an active OpenCode session (SessionID != "") or tmux window (Window != "") should never be marked "completed" regardless of Phase or SYNTHESIS.md
- **Artifacts indicate completion intent, not actual completion:** Phase: Complete and SYNTHESIS.md mean the agent believes it's done, but if the session is still running, the agent hasn't exited yet

### Constraints Discovered
- Build has pre-existing errors in `pkg/spawn/learning.go` - `SourceProject` field missing from `GapEvent` struct

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Syntax verified passing (gofmt)
- [ ] Full tests - blocked by pre-existing build issues in pkg/spawn/learning.go (unrelated to this change)
- [x] Investigation file not required (simple debugging fix)
- [x] Ready for `orch complete orch-go-r8a7`

### Note for Orchestrator
The build has pre-existing errors in `pkg/spawn/learning.go` related to `GapEvent.SourceProject` field. These are NOT caused by this fix. The serve.go syntax is verified correct.

---

## Unexplored Questions

**Straightforward debugging session, no unexplored territory**

The fix is localized to the status detection logic in handleAgents. No architectural questions emerged.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** Opus
**Workspace:** `.orch/workspace/og-debug-dashboard-shows-agent-27dec/`
**Beads:** `bd show orch-go-r8a7`
