# Session Synthesis

**Agent:** og-arch-investigate-phantom-agent-18feb-ed10
**Issue:** orch-go-1038
**Duration:** 2026-02-18 13:00:00 → 2026-02-18 13:24:41
**Outcome:** partial

---

## TLDR

`/session/status` is not empty on this server and is not used by `orch status`, so phantom accumulation is driven by status logic and lifecycle gaps. The primary overcount is `orch status` marking agents as phantom without checking beads open/closed, which counts completed agents as phantoms; session metrics persist because `~/.orch/session.json` never expires without `orch session end`.

---

## Delta (What Changed)

### Files Created
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-18-session-status-empty-phantoms.md` - Probe documenting /session/status behavior.
- `.orch/workspace/og-arch-investigate-phantom-agent-18feb-ed10/SYNTHESIS.md` - Session synthesis.

### Files Modified
- None

### Commits
- None

---

## Evidence (What Was Observed)

- `/session/status` returned a non-empty map: `curl -s -S http://127.0.0.1:4096/session/status` => `{"ses_38d5f01f6ffe2o0zxalpXLoSAh":{"type":"busy"}}`.
- `orch status` computes `IsPhantom` without checking beads open/closed (`cmd/orch/status_cmd.go`), then counts phantom before completed, so completed agents with no session/window inflate phantom counts (`cmd/orch/status_cmd.go`).
- Canonical phantom logic already exists in `pkg/state/reconcile.go` and requires beads open (`LivenessResult.IsPhantom`).
- Session metrics duration uses `now.Sub(sess.StartedAt)` and has no auto-expiry; if `orch session end` is never run, `TimeInSession` grows unbounded (`pkg/session/session.go`, `cmd/orch/status_cmd.go`).

### Tests Run
```bash
# No tests were run (investigation only)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-18-session-status-empty-phantoms.md` - Confirms /session/status is not empty; phantom count is not due to this endpoint.

### Decisions Made
- None

### Constraints Discovered
- `orch status` phantom counts can be inflated by design, independent of OpenCode `/session/status` behavior.
- Orchestrator session metrics require explicit `orch session end` to reset.

---

## Next (What Should Happen)

**Recommendation:** escalate

### If Escalate
**Question:** Which fix should be prioritized to reduce phantom accumulation and misleading status metrics?
**Options:**
1. Align `orch status` phantom logic with `state.LivenessResult.IsPhantom` and ensure completed agents are not counted as phantom.
2. Add a reconciliation command or daemon pass to detect open beads issues with no live session/window and prompt for `orch abandon` or `orch complete --force`.
3. Add auto-expiration for orchestrator sessions (e.g., if duration exceeds DefaultOrchestratorThresholds max and no spawns for N hours, mark inactive).

**Recommendation:** Option 1 first (low risk, immediate correctness), then Option 2 for lifecycle hygiene, then Option 3 as a UX improvement for session metrics.

---

## Unexplored Questions

- Should `orch status` treat `api-stalled` differently (e.g., if OpenCode server is down, avoid labeling as phantom)?
- Do we want automatic closure/abandonment of beads issues after a long idle window, or only surface candidates?

---

## Session Metadata

**Skill:** architect
**Model:** gpt-5.2-codex
**Workspace:** `.orch/workspace/og-arch-investigate-phantom-agent-18feb-ed10/`
**Investigation:** `.kb/models/agent-lifecycle-state-model/probes/2026-02-18-session-status-empty-phantoms.md`
**Beads:** `bd show orch-go-1038`
