# Probe: --no-track Invisible Agent Operational Cost

**Date:** 2026-03-03
**Status:** Complete
**Model:** spawn-architecture
**Probing claim:** The spawn architecture's --no-track flag creates a class of agents that are operationally invisible

---

## Question

Does the spawn architecture's `--no-track` flag create agents that are invisible to operational tooling (status, complete, clean, frontier), and if so, what is the blast radius?

## What I Tested

Traced the --no-track code path through the full agent lifecycle:
1. **Spawn:** `determineBeadsID()` in `pkg/orch/spawn_beads.go:70-76` — generates synthetic `{project}-untracked-{timestamp}` ID
2. **Status:** `orch status` tracked lane queries beads `orch:agent` label — untracked agents not labeled
3. **Sessions:** `orch sessions` queries OpenCode API — only works for headless (OpenCode) spawns, not Claude CLI (default backend)
4. **Complete:** `cmd/orch/complete_*.go` — requires `isUntrackedBeadsID()` guard to avoid beads API failures
5. **Clean:** `cmd/orch/clean_cmd.go` — orphan GC queries beads `orch:agent` label, misses untracked; tmux cleanup requires beads issue to be closed
6. **Daemon:** `pkg/daemon/active_count.go:18` — explicitly excludes untracked agents from count

## What I Observed

**Complete invisibility for Claude CLI --no-track agents:**

| Tool | Tracked agents | --no-track (OpenCode) | --no-track (Claude CLI) |
|------|---------------|----------------------|------------------------|
| `orch status` | Visible | NOT visible | NOT visible |
| `orch sessions` | N/A (different lane) | Visible | **NOT visible** |
| `orch complete` | Works | Works (with guard) | Works (with guard) |
| `orch clean --orphans` | Works | **NOT detected** | **NOT detected** |
| `orch clean --sessions` | Works | Server TTL handles | **NOT cleaned** |
| Daemon active count | Counted | **Excluded** | **Excluded** |

**Blast radius:** Since Claude CLI became the default backend (Feb 19, 2026), any `--no-track` agent is invisible in BOTH lanes of the two-lane architecture. The `orch sessions` lane only works for OpenCode-backed agents.

**Special-case code accumulated:** 5 distinct `isUntrackedBeadsID()` guards exist across the codebase to prevent crashes when beads operations encounter synthetic IDs. Each is a symptom of the same root cause: synthetic IDs that look like beads IDs but aren't.

## Model Impact

**Extends model:** The spawn-architecture model should document that `--no-track` creates a third, invisible class of agents that falls outside both lanes of the two-lane architecture. This was partially acknowledged in the Two-Lane ADR ("--no-track agents are invisible") but the full blast radius — especially after Claude CLI became the default backend — was not modeled.

**Specific claims affected:**
- The two-lane split assumes untracked agents are visible in `orch sessions`. This is only true for OpenCode-backed agents, not Claude CLI agents.
- The cleanup pipeline assumes all orphans are discoverable via beads `orch:agent` label. Untracked agents bypass this entirely.

**Recommended model update:** Add invariant: "Every spawned agent MUST have a beads issue for lifecycle management. The `--no-track` flag violates this invariant and should be replaced with lightweight tracking."
