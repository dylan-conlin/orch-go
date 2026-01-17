# Session Handoff

**Orchestrator:** og-orch-understand-current-state-16jan-c1d0
**Focus:** understand current state of the daemon. is it working well from cross-project work? how does it determine when to use escape-hatch vs opencode, etc
**Duration:** 2026-01-16 09:40 → 10:05
**Outcome:** success

---

## TLDR

Investigated daemon's operational state. **Key findings:** (1) Cross-project NOT enabled - daemon only polls orch-go. (2) Backend selection works well - infrastructure detection auto-applies escape hatch. (3) Daemon healthy but starving (no `triage:ready` labels). **Bonus:** Discovered and spawned fix for dashboard follow mode bug where price-watch agents weren't visible (project name mismatch: `price-watch` vs `pw`).

---

## Spawns (Agents Managed)

### Completed
*None yet*

### Still Running
| Agent | Issue | Skill | Phase | ETA |
|-------|-------|-------|-------|-----|
| og-debug-dashboard-follow-mode-16jan-a731 | orch-go-mg741 | systematic-debugging | Starting | ~15m |

### Blocked/Failed
*N/A*

---

## Evidence (What Was Observed)

### Patterns Across Agents
*N/A - research session*

### Completions
*N/A*

### System Behavior
- **Daemon is running:** PID 26214, launchd-managed, KeepAlive=true
- **Not cross-project:** Daemon plist has `WorkingDirectory=/Users/dylanconlin/Documents/personal/orch-go` and no `--cross-project` flag
- **Queue starvation:** 52 open issues, 0 with `triage:ready` label → daemon spawns nothing
- **Completion blocked:** orch-go-2nruy blocked on verification failure (missing test evidence)
- **Backend selection works:** Infrastructure keyword detection auto-applies escape hatch (`--backend claude --tmux`) for orch-go/opencode/dashboard work

---

## Knowledge (What Was Learned)

### Backend Selection (Escape-Hatch vs OpenCode)

**Priority order in `spawn_cmd.go`:**
1. Explicit `--backend` flag (highest priority)
2. `--opus` flag forces claude backend
3. **Infrastructure work detection** - auto-applies escape hatch
4. Auto-selection based on `--model` (opus → claude, sonnet → opencode)
5. Config default (`spawn_mode` in project config)
6. Default to opencode

**Infrastructure detection keywords** (case-insensitive):
- opencode, orch-go, pkg/spawn, pkg/opencode, pkg/verify
- cmd/orch, spawn_cmd.go, serve.go, main.go
- dashboard, agent-card, agents.ts, daemon.ts
- skillc, SPAWN_CONTEXT, spawn system, orchestration infrastructure

**Flow for daemon-spawned work:**
```
Daemon → `orch work <beadsID>` → runWork() → runSpawnWithSkillInternal()
         ↓
         isInfrastructureWork(task, beadsID) checks keywords
         ↓
         If infrastructure → auto-applies --backend claude --tmux
         ↓
         Otherwise → default opencode backend
```

### Cross-Project Status

**Current:** NOT enabled. Daemon only polls orch-go.

**Why it appears cross-project:** `orch status` shows agents from price-watch, specs-platform - these were spawned manually, not by daemon. The daemon tracks all OpenCode sessions for capacity management regardless of origin.

**To enable cross-project:**
```bash
# In launchd plist, add to ProgramArguments:
<string>--cross-project</string>
```

### Constraints Discovered
- Daemon requires `triage:ready` label - no exceptions
- Infrastructure detection is keyword-based, not semantic - could miss edge cases
- Cross-project requires explicit `--cross-project` flag on daemon startup

### Decisions Made
*No decisions required - understanding session only*

### Externalized
*No need to externalize - this is documented in `.kb/guides/daemon.md` and `.kb/models/daemon-autonomous-operation.md`*

### Artifacts Created
*None - research session*

---

## Friction (What Was Harder Than It Should Be)

### Tooling Friction
- `orch status` shows "52 ready" which is misleading - these are open issues, not ready issues (would spawn if labeled)

### Context Friction
- No friction - guides and models were comprehensive

### Skill/Spawn Friction
*N/A - no spawns*

---

## Focus Progress

### Where We Started
- **Daemon status:** Running, 52 ready issues in queue
- **Agent state:** 83 active agents (all idle), 112 completed, many marked AT-RISK
- **Cross-project evidence:** orch status shows agents from orch-go, price-watch, specs-platform
- **Open questions:** How daemon selects backend (claude vs opencode), cross-project handling

### Where We Ended
- **Cross-project question answered:** NOT enabled - daemon only polls orch-go
- **Backend selection understood:** Infrastructure detection auto-applies escape hatch, works well
- **Root cause of inactivity:** No `triage:ready` labels - backlog needs triage
- **Daemon is healthy:** Running, polling every 60s, capacity management working

### Scope Changes
- None - stayed focused on understanding questions

---

## Next (What Should Happen)

**Recommendation:** shift-focus

### If Shift Focus
**New focus:** Triage the orch-go backlog to release work to daemon
**Why shift:** Understanding questions answered. Daemon is healthy but starving. 10 ready issues (`bd ready`) need `triage:ready` labels to flow through daemon.

**Immediate actions:**
1. Review the 10 ready issues from `bd ready`
2. For confident issues: `bd label <id> triage:ready`
3. For uncertain issues: Leave as is for orchestrator review

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should cross-project be enabled? Currently all Dylan's work flows through orch-go orchestration home.
- Should `orch status` distinguish between "open issues" and "triage:ready issues"?

**System improvement ideas:**
- Add `--cross-project` to launchd plist if Dylan wants daemon to poll other projects automatically
- Dashboard could show "triage:ready count" vs "total open count" to clarify spawn potential

---

## Session Metadata

**Agents spawned:** 1
**Agents completed:** 0
**Issues closed:** none
**Issues created:** orch-go-mg741 (dashboard follow mode fix)

**Workspace:** `.orch/workspace/og-orch-understand-current-state-16jan-c1d0/`
