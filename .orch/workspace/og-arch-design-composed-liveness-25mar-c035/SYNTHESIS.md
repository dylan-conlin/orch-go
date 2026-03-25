# Session Synthesis

**Agent:** og-arch-design-composed-liveness-25mar-c035
**Issue:** orch-go-k8sle
**Duration:** 2026-03-25
**Outcome:** success

---

## Plain-Language Summary

Claude Code agents have a liveness detection bug that goes in the OPPOSITE direction from what we thought. The prior investigation (orch-go-jhluq) said "IsProcessing is never set for Claude agents." That's wrong — it IS set, but from a static signal (historical phase comments), not a live one. This means dead Claude agents with stale phase comments look permanently "processing" and can never be flagged as unresponsive. The fix: wire `IsPaneActive()` (an existing tmux function that checks if a non-shell process is running in the pane) as a live override for Claude agents, exactly mirroring how OpenCode's session API overrides for OpenCode agents. Three files change: discovery.go (signal priority reorder + window ID enrichment), status_cmd.go (Claude override after OpenCode enrichment), serve_agents_handlers.go (same).

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

Key outcomes:
- Dead Claude agents correctly classified as "dead" (not masked by stale phase)
- Active Claude agents have live IsProcessing from IsPaneActive
- No regression in OpenCode agent detection

---

## TLDR

Designed composed liveness detection for Claude Code agents. The core insight: IsProcessing IS set for Claude agents already (from status→IsProcessing mapping), but from a static signal (phase comments) rather than a live signal. Dead agents are permanently masked as "processing." Fix: wire IsPaneActive() as a live override in 3 code paths.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-25-design-composed-liveness-detection-claude.md` — Design investigation with 5 findings, implementation recommendations
- `.orch/workspace/og-arch-design-composed-liveness-25mar-c035/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-design-composed-liveness-25mar-c035/BRIEF.md` — Comprehension artifact
- `.orch/workspace/og-arch-design-composed-liveness-25mar-c035/VERIFICATION_SPEC.yaml` — Verification contract

---

## Evidence (What Was Observed)

- `agentStatusToAgentInfo` (status_cmd.go:462) maps status="active" → IsProcessing=true for ALL agents including Claude
- OpenCode enrichment (status_cmd.go:234) OVERRIDES IsProcessing for OpenCode agents but nothing overrides for Claude
- Discovery's Claude backend path (discovery.go:404) gives phase_reported higher priority than tmux_window_alive, masking dead agents
- `IsPaneActive()` (pane.go:68) already exists, uses dual-signal detection (pane_current_command + child process check)
- `CheckTmuxWindowAlive` (discovery.go:67) discards the WindowInfo that IsPaneActive needs
- StallTracker (stall_tracker.go) provides the pattern for future pane content delta refinement

---

## Architectural Choices

### IsPaneActive over pane content delta
- **What I chose:** Use IsPaneActive() as the live IsProcessing signal
- **What I rejected:** Pane content delta (hash comparison between polls)
- **Why:** IsPaneActive is simpler, already tested, and answers our question ("is the agent process running?"). Pane content delta solves a narrower problem (agent alive but idle at prompt) that rarely occurs for autonomous agents.
- **Risk accepted:** IsPaneActive returns true for agents at an idle prompt. For autonomous orch-spawned agents this is rare, but if it becomes common, pane content delta can be added as a refinement layer.

### Live override pattern over discovery-level IsProcessing
- **What I chose:** Set IsProcessing in discovery for Claude, let consumers use it as override
- **What I rejected:** Adding IsProcessing to AgentStatus for ALL backends
- **Why:** Mirrors the existing OpenCode pattern (conversion sets IsProcessing from status, then live signal overrides). Minimizes change to the discovery→consumer boundary.
- **Risk accepted:** Two different mechanisms set IsProcessing for different backends. Adds complexity for future maintainers.

### Signal priority reorder: tmux before phase
- **What I chose:** Check tmux window alive before using phase for status
- **What I rejected:** Keeping phase_reported as highest-priority signal
- **Why:** Phase comments are historical artifacts. A dead agent with a phase comment should be classified as "dead", not "active." This is a Class 5 (Contradictory Authority Signals) defect.
- **Risk accepted:** Agents whose tmux windows are closed before completion are now immediately classified as "dead" instead of "active with phase." This is correct behavior but may surface more dead agents in status output.

---

## Knowledge (What Was Learned)

### Decisions Made
- IsPaneActive is sufficient for MVP; pane content delta deferred
- Signal priority in Claude backend: tmux alive first, phase second
- PID from session files not needed (IsPaneActive subsumes it)

### Constraints Discovered
- Discovery's AgentStatus has no IsProcessing field — consumers derive it from Status. Adding it creates a mixed model (discovery sets for Claude, consumers override for OpenCode).

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Implementation Decomposition

3 component issues needed:

1. **Component 1: Discovery signal refactor** — Refactor Claude backend routing in discovery.go. Add TmuxWindowID and IsProcessing to AgentStatus. Replace CheckTmuxWindowAlive bool with FindTmuxWindowForAgent. Reorder signal priority.

2. **Component 2: status_cmd.go Claude override** — Add Claude IsProcessing override loop after OpenCode enrichment. Use discovery's IsProcessing from IsPaneActive.

3. **Component 3: serve_agents_handlers.go Claude override** — Same pattern for dashboard API.

4. **Integration issue** — Verify end-to-end: active Claude agent shows IsProcessing=true, dead Claude agent shows dead status, no OpenCode regression.

---

## Unexplored Questions

- How common is the "agent at idle prompt" scenario for autonomous Claude agents? If common, pane content delta tracker needed.
- Should discovery set IsProcessing for ALL backends (unifying the model) in a future refactoring?
- Would a `ClaudeLivenessService` (parallel to OpenCode's session API) be worth building if Claude Code agent count grows significantly?

---

## Friction

Friction: ceremony: investigation template is 249 lines, mostly boilerplate sections that don't apply to architect sessions (Investigation History, External Documentation).

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-composed-liveness-25mar-c035/`
**Investigation:** `.kb/investigations/2026-03-25-design-composed-liveness-detection-claude.md`
**Beads:** `bd show orch-go-k8sle`
