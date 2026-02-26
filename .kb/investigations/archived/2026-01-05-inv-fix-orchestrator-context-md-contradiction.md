## Summary (D.E.K.N.)

**Delta:** ORCHESTRATOR_CONTEXT.md template incorrectly told spawned orchestrators to run `orch session end` when they should WAIT for the level above.

**Evidence:** Decision 2026-01-04-orchestrator-session-lifecycle.md states: "Orchestrators and meta-orchestrators don't try to end their own sessions (no orch session end). They write SESSION_HANDOFF.md when goal reached, then wait."

**Knowledge:** Spawned orchestrators have a different lifecycle than interactive orchestrators - they wait for completion from above, they don't self-terminate.

**Next:** Template fixed, ready for verification and commit.

---

# Investigation: Fix Orchestrator Context Md Contradiction

**Question:** How should spawned orchestrators complete their sessions according to the decision?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Decision establishes hierarchical completion model

**Evidence:** Decision 2026-01-04-orchestrator-session-lifecycle.md states:
- "Orchestrators and meta-orchestrators don't try to end their own sessions (no `orch session end`)"
- "They write SESSION_HANDOFF.md when goal reached, then wait"
- "Session end happens when the level above runs `orch complete`"

**Source:** .kb/decisions/2026-01-04-orchestrator-session-lifecycle.md:60-62

**Significance:** This establishes the pattern: spawned entities don't self-terminate, they signal readiness and wait for the level above.

---

### Finding 2: Template contained contradictory instructions

**Evidence:** OrchestratorContextTemplate in pkg/spawn/orchestrator_context.go:
- Line 34: "You use `orch session end` when complete (not `/exit`)"
- Line 78: "Run: `orch session end`"
- Line 82: "Do NOT use `/exit` - that's for worker agents. Orchestrators use `orch session end`."

**Source:** pkg/spawn/orchestrator_context.go:30-34, 66-83

**Significance:** These instructions directly contradicted the decision, telling spawned orchestrators to run a command they shouldn't run.

---

### Finding 3: Fix aligns template with decision

**Evidence:** Updated template now:
- Tells spawned orchestrators to WAIT after writing SESSION_HANDOFF.md
- Explains the level above will run `orch complete`
- Prohibits both `/exit` AND `orch session end`

**Source:** pkg/spawn/orchestrator_context.go (after edit)

**Significance:** Template now matches the decision's hierarchical completion model.

---

## Synthesis

**Key Insights:**

1. **Spawned vs Interactive distinction** - Interactive orchestrators use `orch session start/end`. Spawned orchestrators are completed by the level above via `orch complete`.

2. **Hierarchical completion** - Each level is completed by the level above: worker → orchestrator, orchestrator → meta-orchestrator, meta-orchestrator → Dylan.

3. **SESSION_HANDOFF.md as signal** - The handoff document signals readiness for completion without triggering self-termination.

**Answer to Investigation Question:**

Spawned orchestrators should:
1. Write SESSION_HANDOFF.md when goal is accomplished
2. WAIT - do not exit or run any session-ending command
3. The level above (meta-orchestrator or Dylan) runs `orch complete` to close the session

---

## References

**Files Examined:**
- .kb/decisions/2026-01-04-orchestrator-session-lifecycle.md - Decision establishing the pattern
- pkg/spawn/orchestrator_context.go - Template with contradiction

**Related Artifacts:**
- **Decision:** .kb/decisions/2026-01-04-orchestrator-session-lifecycle.md
