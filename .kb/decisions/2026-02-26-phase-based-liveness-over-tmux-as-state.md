---
stability: foundational
---
# Decision: Phase-Based Liveness Over Tmux-as-State for Claude-Backend Agents

**Date:** 2026-02-26
**Status:** Accepted
**Deciders:** Dylan (via orchestrator), probe evidence
**Context Issues:** orch-go-1185, orch-go-1186
**Extends:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` (Amendment: Claude-Backend Liveness Gap)

## Context

When `runSpawnClaude` was introduced (Feb 21, 2026), it created a third spawn path that bypasses OpenCode entirely — the Claude CLI runs directly in tmux, producing agents with no OpenCode session. The query engine's liveness check returned `missing_session`, and the dashboard showed these agents as `"dead"` even when actively working. As of Feb 24, 30% of claude-mode agents (32 of 105) hit this gap.

Two approaches were tried:

1. **Tmux liveness check** (orch-go-1182/1183): Used tmux window existence as a liveness signal. Failed architecturally.
2. **Phase-based liveness** (orch-go-1185): Used beads phase comments as a heartbeat. Succeeded.

This decision promotes the phase-based approach and explicitly rejects tmux-as-state.

## Decision

**For agents without OpenCode sessions, use beads phase comments as the liveness signal. Never use tmux window existence to determine agent state.**

### Liveness Rules

| Condition | Status | Reason Code |
|-----------|--------|-------------|
| Phase comment exists (not "Complete") | `active` | `phase_reported` |
| Phase: Complete | `completed` | `phase_complete` |
| Recently spawned (<5 min), no phase yet | `active` | `recently_spawned` |
| No phase comment, >5 min since spawn | `dead` | `no_phase_reported` |

### Implementation

`cmd/orch/query_tracked.go:363-385` — when `SpawnMode == "claude"` and `SessionID == ""`, the query engine checks phases (already fetched from beads) instead of OpenCode session status.

## Why Tmux-as-State Failed

The tmux liveness approach violated two invariants from the two-lane decision:

**1. Domain Boundaries (Invariant 6):** The two-lane decision states tmux owns "Presentation layer" and does NOT own "Any state whatsoever." The tmux check mapped `window exists → active` and `window missing → dead`, making tmux a state oracle.

**2. Cache TTL (Invariant 7):** The fix required a 10-second TTL cache to paper over tmux command unreliability. The two-lane decision allows 1-5 second TTLs. Needing to exceed the allowed range was itself the signal that the approach was wrong.

**3. Structural unreliability:** The dashboard server runs inside overmind (which uses its own tmux), requiring socket override logic that races with 3+ tmux shell-outs per agent check. This caused 10-second oscillation between "active" and "dead" — not a fixable bug but a structural property of running tmux commands from inside an overmind-managed process.

## Why Phase-Based Liveness Works

1. **Authoritative source:** Phase comments come from beads, the canonical source of truth per the two-lane decision
2. **Zero additional cost:** The `phases` map is already computed and passed to `joinWithReasonCodes()` — no new fetching, no new caches, no shell-outs
3. **Reliable heartbeat:** Worker-base skill enforces phase reporting within first 3 tool calls, providing a consistent signal
4. **Graceful degradation:** The 5-minute grace period covers the gap between spawn and first tool call; agents that never report a phase are correctly marked `dead`
5. **No new state layer:** This reads an additional field from data already fetched, not a new source of truth

## Consequences

### Positive
- Agents using Claude CLI backend appear correctly in dashboard (active while working, dead when silent)
- No tmux dependency in the query engine — tmux stays pure UI layer
- No cache needed — phase data comes from beads, already fetched
- Consistent with two-lane architecture invariants

### Negative
- Agents that crash before reporting their first phase (within 5 min) appear as `active` (false positive during grace period)
- Agents that stop reporting phase comments but keep running appear as `active` (phase is a lagging indicator — reports what happened, not what's happening now)
- Grace period (5 min) is a heuristic, not a contract

### What Was Reverted
- Tmux liveness check (orch-go-1182): `checkTmuxWindowLiveness()` removed
- 10-second cache (orch-go-1183): `tmuxLivenessCache` removed

## Evidence

### Probes
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-tmux-liveness-two-lane-violation.md` — Documents invariant violations
- `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-spawn-dashboard-visibility-gap.md` — Root cause analysis

### Code
- `cmd/orch/query_tracked.go:363-385` — Phase-based liveness implementation
- `cmd/orch/query_tracked_test.go:525-680` — Tests for all four liveness states

### Model Updates
- `.kb/models/agent-lifecycle-state-model/model.md` — Updated Feb 25 with phase-based liveness section and Invariant 9
- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Amendment section (Feb 24) and updated domain boundaries table
