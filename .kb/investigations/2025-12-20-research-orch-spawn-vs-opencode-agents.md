**TLDR:** The `orch spawn` pattern cannot be fully replaced by native OpenCode agents because it provides essential orchestration infrastructure (tmux windows, beads integration, workspace management) that native agents lack. While native agents are good for packaging instructions, `orch spawn` is necessary for observability, accountability, and background execution in Dylan's multi-agent workflow. High confidence (90%) based on analysis of `orch-go` source and OpenCode CLI capabilities.

---

# Investigation: orch spawn vs Native OpenCode Agents

**Question:** Can the `orch spawn` pattern be replaced by native OpenCode agents?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Orchestrator
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: tmux Visibility and Backgrounding
`orch spawn` provides a dedicated tmux window per agent, which is critical for Dylan's workflow. Native OpenCode agents run in the current terminal session and do not inherently provide background execution or dedicated terminal windows.

**Evidence:** `orch-go` source code (`cmd/orch/main.go`) shows explicit tmux window creation and command sending. `opencode run --agent` is a blocking operation in the current terminal.

**Source:** `cmd/orch/main.go:548`, `opencode run --help`

**Significance:** Without the `orch` wrapper, Dylan would lose the ability to monitor agents in real-time in his "Right Ghostty" window.

---

### Finding 2: State Tracking and Beads Integration
`orch spawn` is deeply integrated with beads for issue tracking and lifecycle management. Native agents are stateless across sessions and lack integration with external issue trackers.

**Evidence:** `orch-go` source code shows beads issue creation, status updates, and phase verification. OpenCode agents have no built-in beads support.

**Source:** `cmd/orch/main.go:496`, `pkg/spawn/context.go:86`

**Significance:** Beads integration is the "glue" that allows the orchestrator and daemon to coordinate work across multiple agents and sessions.

---

### Finding 3: Dynamic Context vs Static Instructions
`orch spawn` uses `SPAWN_CONTEXT.md` to provide dynamic, task-specific context (beads ID, project dir, specific task). Native agents use static "instructions" defined at creation time.

**Evidence:** `pkg/spawn/context.go` defines a template for `SPAWN_CONTEXT.md` that is populated with dynamic data at spawn time.

**Source:** `pkg/spawn/context.go:14`

**Significance:** Dynamic context is essential for agents to know their specific role, tracking ID, and deliverables for a given task.

---

## Synthesis

**Key Insights:**

1. **Orchestration vs Execution** - OpenCode agents are an *execution* feature (packaging instructions and tools). `orch spawn` is an *orchestration* feature (managing the environment, state, and lifecycle of agents).
2. **Observability is Non-Negotiable** - The tmux-based workflow is a core requirement for Dylan's environment. Native agents do not provide this out of the box.
3. **Beads is the Source of Truth** - The entire orchestration system (including the daemon) relies on beads for coordination. Native agents have no awareness of this layer.

**Answer to Investigation Question:**

No, the `orch spawn` pattern cannot be replaced by native OpenCode agents. While native agents provide a cleaner way to define agent roles, they lack the infrastructure for background execution, real-time observability (tmux), and persistent state tracking (beads) that `orch spawn` provides. `orch spawn` should continue to be used as the orchestration wrapper, potentially leveraging native agents as the underlying execution mechanism in the future.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
The analysis of the `orch-go` source code clearly shows the value-add of the wrapper. The OpenCode CLI help and behavior confirm that it lacks the specific orchestration features (tmux, beads) that Dylan relies on.

**What's certain:**
- ✅ `orch spawn` provides tmux windows; native agents do not.
- ✅ `orch spawn` integrates with beads; native agents do not.
- ✅ `orch spawn` provides dynamic context via files; native agents use static instructions.

**What's uncertain:**
- ⚠️ Future OpenCode updates might add backgrounding or better subagent visibility.
- ⚠️ The exact internal implementation of OpenCode agents (though their CLI behavior is clear).

---

## Implementation Recommendations

### Recommended Approach ⭐

**Keep the `orch spawn` wrapper.** Continue developing `orch-go` as the primary orchestration tool.

**Why this approach:**
- Preserves the high-observability tmux workflow.
- Maintains robust state tracking via beads.
- Allows for autonomous work management via the daemon.

**Trade-offs accepted:**
- Maintaining a separate Go binary (`orch-go`) alongside OpenCode.
- Slightly more complex setup than using native agents alone.

**Implementation sequence:**
1. Continue refactoring `orch-go` into its target structure.
2. (Optional) Explore using native agents *within* `orch spawn` to simplify prompt construction.

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Spawn implementation.
- `pkg/spawn/context.go` - `SPAWN_CONTEXT.md` generation.
- `pkg/registry/registry.go` - Agent tracking.

**Commands Run:**
```bash
# Check OpenCode agent commands
opencode agent --help
opencode agent list

# Check OpenCode run options
opencode run --help
```
