# Decision: ATC-Not-Conductor Orchestrator Reframe

<!-- ABOUT DECISIONS
    Decisions are commitments that constrain future work.
    Key metadata fields:
      - Date: When the decision was made
      - Status: Proposed | Accepted | Superseded
      - Enforcement: How strictly the decision is enforced
          convention = social agreement, teams follow voluntarily
          context-only = injected as context, not gated
          gate = mechanically enforced (blocks work that violates)
      - Deciders: Who made the decision
      - Extends: Prior decision this builds on (decisions are layered)
-->

**Date:** 2026-02-28
**Status:** Accepted
**Enforcement:** convention
**Deciders:** Dylan
**Extends:** Prior strategic orchestrator model decision

## Decision

The orchestrator's mental model is **Air Traffic Controller (ATC)**, not **orchestra conductor**. The orchestrator manages airspace (system state, sequencing, deconfliction, safety) while agents fly their own routes. The orchestrator does not direct agent execution.

## Context

<!-- Context explains WHY the decision was needed.
     Name the pain that motivated it — abstract principles
     without concrete problems don't justify commitments. -->

The word "orchestrator" naturally evokes a conductor metaphor — someone who stands at the center, directing every musician's timing, dynamics, and expression. This framing caused persistent drift:

1. **Orchestrators micro-managing agent work** — reviewing implementation details, reading code files, suggesting specific approaches instead of delegating
2. **Agents waiting for direction** — treating the orchestrator as a command source rather than exercising autonomous judgment
3. **Bottleneck formation** — the orchestrator becoming the throughput constraint because everything routes through them

The system already evolved away from this in practice. But the conductor metaphor persists in how people think about the role, and thinking shapes behavior.

## The Reframe

### Conductor (Wrong Model)

| Aspect | Conductor Behavior |
|--------|-------------------|
| Relationship to performers | Directs every phrase, every dynamic |
| Information flow | Center-out (conductor to musicians) |
| Performers' autonomy | Follow the conductor's interpretation |
| If conductor stops | Music stops |
| Quality signal | How well performers follow direction |

### ATC (Right Model)

| Aspect | ATC Behavior |
|--------|-------------|
| Relationship to pilots | Manages airspace, not flight controls |
| Information flow | Bidirectional (clearances and position reports) |
| Pilots' autonomy | Fly their own aircraft, make tactical decisions |
| If ATC stops | Planes keep flying (degraded but functional) |
| Quality signal | No collisions, efficient sequencing, safe landings |

### What ATC Does

| ATC Function | Orchestrator Equivalent |
|-------------|------------------------|
| **Sequencing** — who lands/takes off when | Triage priority, spawn ordering |
| **Separation** — prevent collisions | Deconflict overlapping work, prevent edit conflicts |
| **Clearance** — authorize transitions | Gate completion reviews, approve scope changes |
| **Weather advisories** — surface conditions | Inject context (knowledge base, hotspot warnings, prior decisions) |
| **Emergency handling** — prioritize distress | Escalation handling, blocked agent triage |
| **Handoff** — transfer between zones | Session boundaries, cross-project coordination |

### What ATC Does NOT Do

| Not ATC's Job | Not Orchestrator's Job |
|---------------|----------------------|
| Fly the plane | Write code or investigate |
| Choose the flight path | Decide implementation approach |
| Operate the instruments | Select tools or patterns |
| Land the aircraft | Complete the agent's deliverables |

## Why This Matters

### 1. Autonomy is the Default

In ATC, pilots have authority over their aircraft. They request clearance for major transitions but make continuous tactical decisions without asking. Similarly, agents have authority over implementation details and only escalate for strategic, irreversible, or cross-boundary decisions.

### 2. The System Survives Orchestrator Absence

If ATC goes offline, planes don't fall out of the sky — they follow procedures, maintain separation, and land safely (degraded mode). Similarly, agents should complete their work even if the orchestrator session ends.

### 3. Information Flow is Bidirectional

A conductor broadcasts interpretation outward. ATC receives position reports AND issues clearances. The orchestrator receives phase reports AND provides context. Neither direction dominates.

### 4. The Orchestrator's Value is Systemic, Not Directive

ATC's value isn't making individual flights better — it's making the system safe and efficient. The orchestrator's value isn't making individual agents better — it's maintaining system coherence.

## What This Changes

<!-- Decisions should make concrete what changes and what stays the same.
     Abstract principles that don't change behavior aren't decisions. -->

### Language

- "Direct agents" becomes "Clear agents for work"
- "Assign tasks" becomes "Sequence and deconflict"
- "Review agent output" becomes "Verify safe completion"
- "Tell the agent how to..." becomes "Surface context, agent decides approach"

### Behavior

- Orchestrator provides **context**, not **instructions** (weather advisory, not flight controls)
- Agent reports **position**, not **requests for direction** (phase comments, not "what should I do?")
- Completion review checks **safety** (did it land correctly?), not **technique** (did it fly the way I would have?)

### Self-Check

The orchestrator should ask: "Am I controlling the flight stick, or managing the airspace?" If the former, delegate.

## Origin

<!-- Origin captures where the decision came from.
     Was it a post-mortem finding? A pattern observed over weeks?
     An external influence? This helps future readers understand
     the decision's evidential weight. -->

Emerged from observing persistent drift where the "orchestrator" label caused conductor-like behavior: micro-management, bottleneck formation, and agents waiting for direction instead of exercising autonomous judgment. The system's architecture already implements ATC patterns (daemon coordination, agent authority, phase reporting), but the mental model hadn't been explicitly named.
