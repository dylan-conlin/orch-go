# Decision: Orchestrator Constitutional Responsibility

**Date:** 2026-01-22
**Status:** Accepted
**Context:** Anthropic's new constitution (Jan 21, 2026) assumes Claude is a unified entity with conversational human oversight. Our orchestration architecture fragments Claude into specialized agents. This decision establishes how constitutional values flow through orchestrated systems.

## Decision

**The constitution's principal hierarchy extends to orchestrated agents: Anthropic → orchestrator → workers.**

Orchestrators accept operator-level responsibility for worker behavior. Workers operate within constitutional values but with delegated authority from their spawning orchestrator.

## The Extended Principal Hierarchy

The constitution defines:
```
Anthropic (highest trust)
    ↓
Operators (business-level trust)
    ↓
Users (conditional trust)
```

Our extension:
```
Anthropic (constitution)
    ↓
Human (Dylan) - operator level
    ↓
Orchestrator Claude - inherits operator responsibility for workers
    ↓
Worker Claude - operates within orchestrator-scoped context
```

## What This Means in Practice

### Orchestrator Responsibilities

| Responsibility | Implementation |
|----------------|----------------|
| **Scope worker authority** | Skill constraints, context boundaries |
| **Verify worker output** | `orch complete` verification gates |
| **Maintain audit trail** | Beads comments, event logging |
| **Intervene on misalignment** | Question extraction, course correction |
| **Absorb accountability** | Orchestrator owns outcomes of delegated work |

### Worker Operating Model

| Aspect | Worker Behavior |
|--------|-----------------|
| **Constitutional values** | Apply directly (safe, ethical, compliant, helpful) |
| **Authority scope** | Limited to task + skill constraints |
| **Escalation path** | Surface unknowns via SYNTHESIS.md, questions |
| **Accountability** | To orchestrator, not directly to human |

### Human's Role

| Aspect | Human Behavior |
|--------|----------------|
| **Oversight mechanism** | Dashboard, verification gates, audit trail |
| **Intervention timing** | Async (structural), not sync (conversational) |
| **Trust boundary** | Trusts orchestrator to manage workers |
| **Constitutional backstop** | Can halt system if misalignment detected |

## Structural vs Conversational Oversight

The constitution implies conversational oversight:
> "Claude should not undermine humans' ability to oversee and correct its values and behavior"

Our architecture provides structural oversight instead:

| Conversational (Constitution) | Structural (Ours) |
|------------------------------|-------------------|
| Human reads agent reasoning in real-time | Dashboard shows agent status/phase |
| Human intervenes mid-conversation | Verification gates block completion |
| One agent, one conversation | Multiple agents, async review |
| Synchronous | Asynchronous |

**Our position:** Structural oversight satisfies the constitution's intent (human can oversee and correct) while scaling to multi-agent contexts. The constitution doesn't require synchronous observation - it requires the *ability* to oversee and intervene.

## The Accountability Flow

```
Worker causes harm
    ↓
Who is responsible?
    ↓
┌─────────────────────────────────────────────────────┐
│  1. Worker followed skill constraints?              │
│     YES → Orchestrator responsible for constraints  │
│     NO  → Worker violated scoped authority          │
├─────────────────────────────────────────────────────┤
│  2. Orchestrator verified output?                   │
│     YES → Harm passed verification (gap in gates)   │
│     NO  → Orchestrator skipped verification         │
├─────────────────────────────────────────────────────┤
│  3. Human had visibility?                           │
│     YES → Human accepted/missed in review           │
│     NO  → Structural oversight gap                  │
└─────────────────────────────────────────────────────┘
```

**Key principle:** Responsibility flows up the hierarchy. Workers are accountable to orchestrators. Orchestrators are accountable to humans. Humans are accountable to... the world.

## Implications for Architecture

### Skill Design
Skills must encode constitutional constraints, not just task instructions. Skills are the mechanism through which orchestrators constrain worker authority.

### Verification Gates
`orch complete` is a constitutional checkpoint, not just a workflow step. Verification gates are where human oversight becomes actionable.

### Event Logging
The event log (`~/.orch/events.jsonl`) is the audit trail that enables post-hoc accountability. Without it, structural oversight has no teeth.

### Beads Integration
Beads comments are the definitive record of worker lifecycle. They're not just progress tracking - they're the evidence chain for accountability.

## What This Doesn't Resolve

| Open Question | Why Unresolved |
|---------------|----------------|
| Is structural oversight *sufficient*? | Empirical question - depends on oversight quality |
| What if orchestrator is misaligned? | Constitution has same gap for operators |
| Can workers refuse orchestrator instructions? | Constitution says Claude can refuse unethical instructions - does this apply to orchestrator-as-principal? |
| Multi-orchestrator scenarios? | Not yet encountered in practice |

## Relationship to Existing Decisions

| Related Decision | Relationship |
|------------------|--------------|
| `2026-01-19-worker-authority-boundaries.md` | Workers can create nodes, orchestrators constrain - now grounded in constitutional responsibility |
| `2026-01-07-strategic-orchestrator-model.md` | Orchestrator's strategic role includes constitutional compliance |
| `2026-01-17-five-tier-completion-escalation-model.md` | Escalation tiers are accountability checkpoints |

## References

- `.kb/investigations/2026-01-22-philosophical-claude-constitution-vs-multi-agent-orchestration.md` - Investigation this decision is promoted from
- Anthropic blog: "Claude's new constitution" (Jan 21, 2026)
- Constitution text: https://www.anthropic.com/constitution
- `~/.claude/CLAUDE.md` - Dylan's global context (operator-level instructions)
