# Decision: ATC-Not-Conductor Orchestrator Reframe

**Date:** 2026-02-28
**Status:** Accepted
**Deciders:** Dylan
**Extends:** `.kb/decisions/2026-01-07-strategic-orchestrator-model.md`

## Decision

The orchestrator's mental model is **Air Traffic Controller (ATC)**, not **orchestra conductor**. The orchestrator manages airspace (system state, sequencing, deconfliction, safety) while agents fly their own routes. The orchestrator does not direct agent execution.

## Context

The word "orchestrator" naturally evokes a conductor metaphor — someone who stands at the center, directing every musician's timing, dynamics, and expression. This framing has caused persistent drift in how the system operates:

1. **Orchestrators micro-managing agent work** — reviewing implementation details, reading code files, suggesting specific approaches instead of delegating and reconnecting
2. **Agents waiting for direction** — treating the orchestrator as a command source rather than exercising autonomous judgment within their authority
3. **Bottleneck formation** — the orchestrator becoming the throughput constraint because everything routes through them for "direction"

The system already evolved away from this in practice. The Strategic Orchestrator Model (Jan 7, 2026) moved the orchestrator's job from "what should we spawn next?" to "what do we need to understand?" The daemon handles coordination. The decision authority guide gives agents autonomous scope. But the conductor metaphor persists in how people think about the role, and thinking shapes behavior.

## The Reframe

### Conductor (Wrong Model)

| Aspect | Conductor Behavior |
|--------|-------------------|
| Relationship to performers | Directs every phrase, every dynamic |
| Information flow | Center-out (conductor → musicians) |
| Performers' autonomy | Follow the conductor's interpretation |
| If conductor stops | Music stops |
| Quality signal | How well performers follow direction |

### ATC (Right Model)

| Aspect | ATC Behavior |
|--------|-------------|
| Relationship to pilots | Manages airspace, not flight controls |
| Information flow | Bidirectional (clearances ↔ position reports) |
| Pilots' autonomy | Fly their own aircraft, make tactical decisions |
| If ATC stops | Planes keep flying (degraded but functional) |
| Quality signal | No collisions, efficient sequencing, safe landings |

### What ATC Does

| ATC Function | Orchestrator Equivalent |
|-------------|------------------------|
| **Sequencing** — who lands/takes off when | Triage priority, spawn ordering |
| **Separation** — prevent collisions | Deconflict overlapping work, prevent edit conflicts |
| **Clearance** — authorize transitions | Gate completion reviews, approve scope changes |
| **Weather advisories** — surface conditions | Inject context (kb context, hotspot warnings, prior decisions) |
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

In ATC, pilots have authority over their aircraft. They request clearance for major transitions (takeoff, landing, altitude changes) but make continuous tactical decisions without asking. Similarly, agents have authority over implementation details (see decision-authority.md) and only escalate for strategic, irreversible, or cross-boundary decisions.

### 2. The System Survives Orchestrator Absence

If ATC goes offline, planes don't fall out of the sky — they follow procedures, maintain separation, and land safely (degraded mode). Similarly, agents should complete their work even if the orchestrator session ends. The daemon provides basic coordination. Agents have SPAWN_CONTEXT.md with full task context. The system degrades gracefully.

### 3. Information Flow is Bidirectional

A conductor broadcasts interpretation outward. ATC receives position reports AND issues clearances. The orchestrator receives phase reports (bd comment) AND provides context (spawn context, kb context). Neither direction dominates.

### 4. The Orchestrator's Value is Systemic, Not Directive

ATC's value isn't making individual flights better — it's making the system safe and efficient. The orchestrator's value isn't making individual agents better — it's maintaining system coherence: no conflicting work, no lost context, no gaps in understanding, smooth reconnection to Dylan's priorities.

## Relationship to Existing Decisions

| Decision | How ATC Reframe Extends It |
|----------|---------------------------|
| Strategic Orchestrator Model (2026-01-07) | "Comprehension, not coordination" maps to ATC's situational awareness. ATC's primary skill is maintaining a mental picture of the airspace. |
| Observation Infrastructure Principle (2026-01-08) | ATC depends on radar (observation). If radar is wrong, ATC can't function. Same principle: if the system can't observe it, it can't manage it. |
| Separate Observation from Intervention (2026-01-14) | ATC observes (radar) separately from intervening (clearances). Same architectural pattern. |
| Absolute Delegation Rule | ATC never flies the plane. Orchestrator never writes code. Same boundary. |
| Orchestrator Skill Orientation Redesign (2026-02-16) | ORIENT → DELEGATE → RECONNECT maps to ATC's: maintain picture → issue clearances → hand off safely. |

## What This Changes

### Language

- "Direct agents" → "Clear agents for work"
- "Assign tasks" → "Sequence and deconflict"
- "Review agent output" → "Verify safe completion"
- "Tell the agent how to..." → "Surface context, agent decides approach"

### Behavior

- Orchestrator provides **context**, not **instructions** (weather advisory, not flight controls)
- Agent reports **position**, not **requests for direction** (phase comments, not "what should I do?")
- Completion review checks **safety** (did it land correctly?), not **technique** (did it fly the way I would have?)

### Self-Check

The orchestrator should ask: "Am I controlling the flight stick, or managing the airspace?" If the former, delegate.

## What This Rejects

- Orchestrator reviewing implementation details of agent code
- Orchestrator suggesting specific approaches to agents mid-flight
- Agents asking the orchestrator "how should I implement this?"
- The idea that agent quality depends on orchestrator direction

## What This Embraces

- Agents as autonomous professionals who own their deliverables
- Orchestrator as system manager who owns coherence, safety, and orientation
- Bidirectional information flow (context down, status up)
- Graceful degradation when orchestrator is absent

## Origin

Emerged from observing persistent drift where the "orchestrator" label caused conductor-like behavior: micro-management, bottleneck formation, and agents waiting for direction instead of exercising autonomous judgment. The system's architecture already implements ATC patterns (daemon coordination, agent authority, phase reporting), but the mental model hadn't been explicitly named.
