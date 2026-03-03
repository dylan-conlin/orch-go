# Decision: Gate Over Remind Refinement - Gates Must Be Passable By The Gated

**Date:** 2026-01-04
**Status:** Accepted
**Context:** Repro verification and dependency check gates blocked completion without clear benefit

## Decision

Add a caveat to the Gate Over Remind principle: Gates must be passable by the gated party. A gate that requires human intervention is a checkpoint, not a gate.

## Context

Following the Gate Over Remind principle, we added verification gates to `orch complete`:

**Repro verification gate:**
- For bug-type issues, required orchestrator to verify the original reproduction no longer occurs
- Added `--skip-repro-check` and `--skip-repro-reason` flags for bypass
- Agent couldn't satisfy this - it required orchestrator manual verification

**Dependency check gate:**
- Blocked spawning when dependent issues were still open
- Agent couldn't resolve dependencies - they're external constraints
- Added `--force` bypass

**What happened:**
> "Jan 4, 2026: Disabled repro verification and dependency check gates - they blocked completion without clear benefit."

The guides document the smell:
> "If completion requires multiple flags to work, that's a smell. The gates may be adding friction without value."

## Rationale

**Why these gates failed:**

| Gate | Who could pass it | Result |
|------|-------------------|--------|
| Build verification | Agent (fix build errors) | Works - valid gate |
| Test evidence | Agent (run tests, report output) | Works - valid gate |
| Repro verification | Orchestrator only (manual check) | Failed - human checkpoint |
| Dependency check | Nobody (external constraint) | Failed - scheduling issue |

**The distinction:**

- **Valid gate:** Blocks progress until the gated party does something. Agent can satisfy it.
- **Human checkpoint:** Blocks progress until a human does something. Disguised as automation.
- **Scheduling constraint:** Blocks progress based on external state. Nobody can satisfy it by action.

**Why human checkpoints fail at scale:**

Gate Over Remind works because gates are *unavoidable* - you can't proceed without satisfying them. But if every completion requires human verification:
- Orchestrator becomes a bottleneck
- Gates get bypassed via `--force`
- Eventually gates get disabled entirely

The repro gate had two bypass flags (`--skip-repro-check`, `--skip-repro-reason`). When you need escape hatches for escape hatches, the gate is wrong.

## The Refined Test

Original test: "Is this a reminder that can be ignored, or a gate that blocks progress?"

Refined test:
1. Is this a reminder that can be ignored? → Make it a gate
2. Can the gated party pass it by their own work? → Valid gate
3. Does it require someone else to act? → Human checkpoint, not a gate
4. Does it depend on external state? → Scheduling constraint, not a gate

## Implications

**For new gates:**
- Ask: "Can the agent satisfy this by doing work?"
- If yes → valid gate
- If no → reconsider (maybe it's a pre-spawn check, or a dashboard warning, but not a completion gate)

**For existing gates:**
- Build verification: Agent can fix → keep
- Test evidence: Agent can run tests → keep
- Visual verification: Tricky - agent can provide evidence, but approval requires human. Current design (agent provides evidence, orchestrator approves with `--approve`) is acceptable because the gate is on *evidence*, not *approval*.

**What repro verification should have been:**
- Gate on agent: "Must include repro verification in SYNTHESIS.md"
- Not gate on orchestrator: "Orchestrator must manually verify"

The agent should prove they checked. The orchestrator trusts but verifies. The gate is on the agent's documentation, not the orchestrator's action.

## Evidence

- `orch-go/.kb/guides/completion-gates.md` - Documents all 11 gates, notes repro/dependency gates disabled
- `orch-go/.kb/guides/agent-lifecycle.md` - "Disabled repro verification and dependency check gates - they blocked completion without clear benefit"
- `orch-go/.kb/investigations/2026-01-04-inv-orch-complete-force-bypasses-repro.md` - Investigation into --force bypassing repro gate
- `orch-go/.kb/investigations/2026-01-03-inv-gate-orch-spawn-issue-beads.md` - Implementation of dependency gating
