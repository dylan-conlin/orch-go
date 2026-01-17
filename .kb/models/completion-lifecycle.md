# Model: Agent Completion Lifecycle

**Domain:** Orchestration / Lifecycle / Verification
**Last Updated:** 2026-01-17
**Synthesized From:** 27 investigations into completion loops, SSE detection, and dashboard reconciliation.

---

## Summary (30 seconds)

The agent completion lifecycle is the transition from **Active Work** to **Knowledge Persistence**. It is the primary gate for the **Verification Bottleneck**. A healthy lifecycle ensures that agent findings are externalized (D.E.K.N.), workspaces are archived, and OpenCode sessions are purged to prevent "Registry Noise." The system uses a **Phase-based status model** where `Phase: Complete` in Beads is the only authoritative signal for success.

---

## Core Mechanism

### The Completion Chain
Completion is not a single event but a chain of state transitions across four layers:

1.  **Work (Agent)**: Agent writes artifacts (investigations, code) and self-verifies.
2.  **Signal (Beads)**: Agent comments `Phase: Complete`. This is the *authoritative signal*.
3.  **Verification (Orchestrator)**: `orch complete` runs automated gates (synthesis, build, tests, visual).
4.  **Persistence (System)**: Beads issue closes, Registry updates to `completed`, OpenCode session is deleted, Workspace is archived.

### The Authoritative Signal
**Constraint:** `Session idle ≠ Agent complete`.
Agents legitimately go idle during thinking or tool execution. `busy→idle` transitions in SSE/OpenCode are used only for dashboard animation, never for lifecycle logic. **`Phase: Complete` is the only truth.**

---

## The Recovery Path

### Dead Agent Recovery
"Dead" agents occur when a lifecycle chain breaks—usually during the synthesis phase where context is most exhausted.

| Failure Mode | Symptom | Recovery Action |
| :--- | :--- | :--- |
| **Synthesis Crash** | `Phase: Completing` + No Session | Use `orch complete --skip-synthesis --skip-phase-complete` if artifact exists in `.kb/`. |
| **Registry Drift** | Agent shows 'running' but issue is closed | Run `orch doctor --fix` to reconcile registry with beads. |
| **Zombie Session** | Agent is done but window/session remains | Run `orch abandon` or `orch complete` with workspace name. |

---

## Constraints

- **The Verification Bottleneck**: Spawning is automated (Daemon), but completion is manual. To maintain system health, orchestrators must dedicate "Hygiene Blocks" to process completions.
- **Visual Gating**: Any change to `web/` files *must* include screenshot evidence or manual smoke-test confirmation before completion.
- **Escape Hatch Discipline**: Infrastructure work (like OpenCode or Beads fixes) should use `--backend claude` to ensure the completion agent survives a service restart.

---

## Why This Matters

### Knowledge Persistence
Without a rigid completion lifecycle, the system suffers from **Understanding Lag**. Completed tasks remain in the "Active" frame, cluttering the dashboard and preventing the synthesis of findings into the Knowledge Base.

### Resource Management
Stale OpenCode sessions and tmux windows consume system memory and "Registry Slot" capacity, eventually blocking the Daemon from spawning new work.

---

## Evolution

- **Dec 2025: The Chaos Period**: 70% of agents completed without synthesis. Established the `SYNTHESIS.md` gate.
- **Jan 2026: The Great Extraction**: Refactored monolithic completion logic into `cmd/orch/complete_cmd.go` and `pkg/verify`.
- **Jan 17, 2026: Dead Agent Recovery**: Defined manual recovery paths for agents crashing during final synthesis.

---

## Integration Points

- **Principles**: Enforces **Verification Bottleneck** and **Session Amnesia**.
- **Guides**: Complements `.kb/guides/completion.md`.
- **Infrastructure**: Informs `orch status` and `orch hotspot` logic.
