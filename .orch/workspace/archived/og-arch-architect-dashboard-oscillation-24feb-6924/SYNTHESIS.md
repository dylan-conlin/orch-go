# Session Synthesis

**Agent:** og-arch-architect-dashboard-oscillation-24feb-6924
**Issue:** orch-go-1184
**Duration:** 2026-02-24
**Outcome:** success

---

## Plain-Language Summary

The dashboard oscillation for claude-backend agents is caused by using tmux window existence as a liveness signal — which the two-lane agent discovery decision explicitly prohibits ("Tmux owns: Presentation layer. Does NOT own: Any state whatsoever"). The tmux check involves 3+ shell-outs per agent, which fail intermittently when the dashboard runs under overmind. A 10s TTL cache was added to paper over the unreliability, but it exceeds the 1-5s allowed range and just changes the oscillation period.

The fix is to **revert the tmux liveness approach (orch-go-1182 and orch-go-1183)** and replace it with **phase-based liveness**: use beads phase comments as a heartbeat signal for claude-backend agents. Phase data is already fetched by the query engine (zero cost), comes from beads (authoritative per two-lane decision), and doesn't involve any shell-outs. Worker-base skill already enforces phase reporting within the first 3 tool calls, making this a reliable signal.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace.

**Key outcomes:**
1. Investigation artifact produced at `.kb/investigations/2026-02-24-design-dashboard-oscillation-tmux-liveness-architectural-analysis.md`
2. Probe file produced at `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md`
3. Three decision forks navigated with substrate traces
4. Clear implementation plan with file targets and acceptance criteria

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-24-design-dashboard-oscillation-tmux-liveness-architectural-analysis.md` — Full architectural analysis with 3 forks navigated, recommendation, and implementation plan
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md` — Probe confirming Invariant 6 and 7 violations, extending model with phase-based liveness

### Files Modified
- None (architecture investigation only)

---

## Evidence (What Was Observed)

- `query_tracked.go:349-362` — tmux window check directly determines agent status ("active"/"dead"), violating two-lane decision's "Tmux does NOT own any state whatsoever"
- `query_tracked.go:26-33` — 10s TTL cache exceeds 1-5s allowed range from two-lane decision
- `pkg/tmux/tmux.go:864-890` — `FindWindowByWorkspaceNameAllSessions` makes 3+ shell-outs per call: list-sessions, has-session (×2), list-windows per session
- `query_tracked.go:304` — `phases` map already passed to `joinWithReasonCodes` — phase-based liveness requires zero new data fetching
- `pkg/orch/extraction.go:1525-1589` — `runSpawnClaude` never calls `AtomicSpawnPhase2`, leaving manifest without session_id
- `serve_agents_handlers.go:497-504` — `agentStatusToAPIResponse` maps "unknown" (from missing_session) to "dead", and "dead" (from tmux failure) also maps to "dead" via default case

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-24-design-dashboard-oscillation-tmux-liveness-architectural-analysis.md` — Full design investigation
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md` — Model probe

### Decisions Made
- Tmux liveness check is architecturally wrong — violates two-lane decision, must be reverted
- Phase-based liveness is the correct approach — uses authoritative beads data, zero cost, compliant

### Constraints Discovered
- Claude-backend agents have NO reliable process-level liveness signal independent of tmux
- The tmux command reliability is structurally limited by overmind socket confusion — not fixable
- The two-lane decision has a gap: it didn't anticipate agents without OpenCode sessions

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement phase-based liveness for claude-backend agents (revert orch-go-1182/1183)
**Skill:** feature-impl
**Context:**
```
Revert tmux liveness check (orch-go-1182) and tmux liveness cache (orch-go-1183) from
query_tracked.go. Replace with phase-based liveness: for spawn_mode=="claude" agents
without session_id, use beads phase comments and spawn_time to determine status.
See .kb/investigations/2026-02-24-design-dashboard-oscillation-tmux-liveness-architectural-analysis.md
for full implementation plan with code snippets.
```

---

## Unexplored Questions

- **Claude Code API:** If Claude Code ever exposes a session/liveness API, that would be the proper liveness source for claude-backend agents. Worth monitoring.
- **Phase reporting latency:** The 5-minute grace period assumes agents report Phase within ~3 minutes. If agents frequently take longer, the grace period may need tuning.
- **Two-lane decision update:** The decision should be amended to note that claude-backend agents use phase comments as liveness proxy. This fills the gap the decision didn't anticipate.

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-architect-dashboard-oscillation-24feb-6924/`
**Investigation:** `.kb/investigations/2026-02-24-design-dashboard-oscillation-tmux-liveness-architectural-analysis.md`
**Beads:** `bd show orch-go-1184`
