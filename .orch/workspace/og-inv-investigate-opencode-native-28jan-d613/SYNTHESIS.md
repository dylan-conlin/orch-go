# Session Synthesis

**Agent:** og-inv-investigate-opencode-native-28jan-d613
**Issue:** orch-go-20995
**Duration:** 2026-01-28 (session start) → 2026-01-28 (session complete)
**Outcome:** success

---

## TLDR

Investigated whether OpenCode has native agent/spawn abstractions and how orch-go could integrate more deeply. Found that OpenCode has comprehensive support (agent modes, task tool, parentID hierarchy, metadata.role) - orch-go currently uses pragmatic ORCH_WORKER headers. Documented three integration levels (status quo, incremental, deep) with trade-offs.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-28-inv-investigate-opencode-native-agent-spawn.md` - Full investigation documenting OpenCode's native agent model

### Files Modified
- None (investigation only)

### Commits
- `62966160` - inv: initial checkpoint - investigating OpenCode native agent abstractions
- `5336d8b1` - inv: complete OpenCode native agent/spawn investigation

---

## Evidence (What Was Observed)

**OpenCode Source Code Analysis:**
- `Agent.Info` schema has `mode: "subagent" | "primary" | "all"` (agent/agent.ts:27)
- Built-in subagents: "general" and "explore" with specific permissions/prompts
- `Session.Info` schema has `parentID: Identifier.schema("session").optional()` (session/index.ts:49)
- Task tool creates sessions with `parentID: ctx.sessionID` (tool/task.ts:69)
- `Session.fork()` and `Session.children()` functions for session hierarchy
- Session metadata has `role: "orchestrator" | "meta-orchestrator" | "worker"` (session/index.ts:82)
- x-opencode-env-ORCH_WORKER header read at session.ts:207-211, sets metadata.role="worker"
- ACP layer has forkSession, resumeSession, listSessions (acp/agent.ts)

**orch-go Integration Points:**
- Sets ORCH_WORKER=1 env var in spawn commands (cmd/orch/spawn_cmd.go:836)
- OpenCode client converts to x-opencode-env-ORCH_WORKER header (pkg/opencode/client.go:596)
- Verified running sessions have metadata.role="worker" via curl /session

### Tests Run
```bash
# Verified OpenCode is running and sessions have metadata
curl -s http://127.0.0.1:4096/session | head -5
# SUCCESS: Sessions returned with metadata.role="worker" for orch-spawned agents

# Searched for parentID/fork usage in OpenCode source
grep -r "parentID\|fork" ~/Documents/personal/opencode/packages/opencode/src --include="*.ts" -n
# FOUND: 61 matches showing comprehensive session hierarchy support

# Searched for ORCH_WORKER in orch-go source
grep -r "ORCH_WORKER\|x-opencode-env" /Users/dylanconlin/Documents/personal/orch-go --include="*.go" -n
# FOUND: 43 matches showing consistent ORCH_WORKER usage across spawn modes
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-28-inv-investigate-opencode-native-agent-spawn.md` - Documents OpenCode's native agent model and three integration options for orch-go

### Decisions Made
- No implementation decision made - investigation presents three architectural options (status quo, incremental, deep integration) for orchestrator to decide
- Flagged for promotion to decision record (architectural trade-off worth preserving)

### Constraints Discovered
- OpenCode sessions are project-scoped (directory field) - implications for cross-project parentID hierarchy unclear, needs testing
- Task tool designed for in-session delegation - may not align with orch-go's external CLI spawning model
- Agent mode system (subagent/primary) designed for OpenCode's task tool paradigm

### Externalized via `kb quick`
- `kb quick decide "OpenCode native agent model vs orch-go external orchestration"` - Documents that OpenCode has task tool + parentID for in-process delegation while orch-go uses CLI spawn + registry for cross-project orchestration
- `kb quick constrain "OpenCode sessions are project-scoped (directory field)"` - Notes session directory field and unclear cross-project parentID behavior

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created, findings documented)
- [x] Tests passing (N/A - investigation only, no code changes)
- [x] Investigation file has `**Phase:** Complete` (status updated)
- [x] Ready for `orch complete orch-go-20995`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does Session.children() perform at scale? Is it O(N) scan or indexed lookup?
- Can parentID reference sessions in different project directories? (cross-project hierarchy)
- What happens to child sessions when parent is deleted? (orphaned? cascade?)
- Does OpenCode UI have tree view for session hierarchy?
- Would ACP integration provide benefits over direct HTTP API for orch-go?
- Could orch-go define custom agents globally (~/.config/opencode/) or only per-project?

**Areas worth exploring further:**
- Benchmarking Session.children() vs orch-go's registry lookups
- Testing cross-project parentID behavior with actual spawns
- Evaluating whether task tool paradigm aligns with orch-go's orchestration model
- ACP client vs HTTP client trade-offs for orch-go integration

**What remains unclear:**
- Whether Option B (incremental parentID) provides meaningful value beyond UI visibility
- Performance implications of session hierarchy at scale (100+ workers)
- OpenCode's roadmap for agent/task features (could change native model)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4.5
**Workspace:** `.orch/workspace/og-inv-investigate-opencode-native-28jan-d613/`
**Investigation:** `.kb/investigations/2026-01-28-inv-investigate-opencode-native-agent-spawn.md`
**Beads:** `bd show orch-go-20995`
