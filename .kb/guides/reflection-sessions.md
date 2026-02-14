# Reflection Sessions

**Purpose:** Orchestrator playbook for running high-signal reflection sessions that reduce rediscovery and improve knowledge hygiene outcomes.

**Last verified:** 2026-02-07

**Primary audience:** Orchestrator sessions (not worker triage).

---

## Quick Start

Run a reflection session when any trigger fires, pick one primary lane, then produce 1-3 durable outcomes.

```bash
# Start from full signal set
kb reflect

# Or lane-focused runs (examples)
kb reflect --type synthesis
kb reflect --type stale
kb reflect --type open
kb reflect --type drift
kb reflect --type promote
```

Use `kb-reflect` skill mechanics for worker-level triage trees and disposition details. This guide defines orchestrator-level session protocol: when to run, what lane to choose, and how to measure results.

---

## 1) Trigger Conditions (When to Run)

Run a session when **any one** of these triggers is true:

| Trigger | Threshold | Why It Exists |
|---------|-----------|---------------|
| **Volume** | >=10 new investigations since last reflection | Prevents backlog buildup |
| **Recurrence** | >=3 investigations in 14 days for one coherent problem family | Catches repeat rediscovery early |
| **Milestone** | Major feature/epic completion OR repeated constraint surfacing | Converts delivery learnings into durable artifacts |
| **Time-floor** | 14 days since last reflection | Safety backstop when event triggers are quiet |

If multiple triggers fire, prioritize the one with highest Amnesia Tax impact (Section 4).

---

## 2) Lane Selection and Rotation

Each session uses a two-lane model:

- **Primary lane (70-80%):** exactly one lane for deep work
- **Maintenance lane (20-30%):** quick sweep for urgent stale/open items

### Available primary lanes

| Lane | Typical Command | Primary Outcome |
|------|------------------|-----------------|
| **synthesis** | `kb reflect --type synthesis` | Consolidate repeated investigations into guide/decision |
| **promote** | `kb reflect --type promote` | Promote quick entries into durable decisions/constraints |
| **stale** | `kb reflect --type stale` | Refresh/archive uncited decisions |
| **drift** | `kb reflect --type drift` | Resolve constraints diverging from actual practice |
| **open** | `kb reflect --type open` | Close pending investigation actions |

### Rotation rule

1. Score top candidates with ATS (Section 4).
2. Select the lane with highest-scoring candidate.
3. If top lanes are close, rotate away from previous session lane to avoid starvation.
4. Always reserve maintenance time for urgent stale/open items.

---

## 3) Session Checklist (Start, During, End)

### Start

1. Confirm which trigger fired (Volume, Recurrence, Milestone, or Time-floor).
2. Run `kb reflect` (full or lane-specific) and gather top 3-5 candidates.
3. Compute ATS for each top candidate.
4. Pick primary lane and define a session objective (one sentence).

### During

1. Process candidates in ATS order.
2. For each candidate choose one: **promote**, **defer**, or **discard**.
3. Record rationale for defer/discard to improve future scoring quality.
4. Keep maintenance sweep bounded (urgent stale/open only).

### End

1. Produce **1-3 durable outputs** (guide, decision, constraint update, or issue).
2. Capture a metrics snapshot (Section 5).
3. Record recommended next lane for the next reflection session.
4. Link resulting artifacts to the originating findings for traceability.

---

## 4) Amnesia Tax Score (ATS)

Use ATS to rank candidates by rediscovery cost, not raw count:

`ATS = Recurrence (0-3) + Delay Cost (0-3) + Blast Radius (0-2) + Preventability (0-2) - Noise Penalty (0-2)`

### Scoring dimensions

| Dimension | Score Range | Practical scoring guide |
|-----------|-------------|-------------------------|
| **Recurrence** | 0-3 | 0=single mention, 1=occasional, 2=repeated, 3=frequent cluster in recent window |
| **Delay Cost** | 0-3 | 0=small annoyance, 1=minor time tax, 2=recurring context loss, 3=major repeated rework |
| **Blast Radius** | 0-2 | 0=one session/agent, 1=multiple sessions in one area, 2=cross-area impact |
| **Preventability** | 0-2 | 0=hard to prevent, 1=partially preventable, 2=clear intervention likely to reduce repeats |
| **Noise Penalty** | 0-2 | 0=coherent family, 1=some ambiguity, 2=likely keyword collision/mixed topic noise |

### Practical examples

| Candidate | Example scoring | ATS | Decision hint |
|-----------|-----------------|-----|---------------|
| Repeated session handoff confusion across orchestrator runs | Recurrence 3, Delay 3, Radius 2, Preventability 2, Noise 0 | **10** | Primary lane now (high confidence, high leverage) |
| One stale decision with no current usage signal | Recurrence 0, Delay 1, Radius 0, Preventability 1, Noise 1 | **1** | Maintenance sweep or defer |
| Constraint drift appearing in multiple active workstreams with mixed evidence | Recurrence 2, Delay 2, Radius 2, Preventability 1, Noise 1 | **6** | Strong candidate; validate coherence before promote |

Rule of thumb: prioritize ATS 7+ first, ATS 4-6 second, ATS <=3 only when quick-win or maintenance-critical.

---

## 5) Success Metrics (30-Day Rolling Window)

Track quality over quantity.

| Metric | How to read it | Healthy direction |
|--------|----------------|-------------------|
| **Repeat Investigation Rate (RIR)** | Repeat investigations per top problem family | Downward trend |
| **Rediscovery Latency** | Median days between first and repeat investigation | Upward trend (repeats happen later) |
| **Promotion Yield Quality** | % of promoted artifacts cited within 30 days | Upward trend |
| **Reflection Throughput Quality** | Durable outputs per session (target 1-3) | Stable 1-3 high-value outputs; avoid output inflation |

Reflection is working when RIR decreases and citation/follow-through improve without creating a growing open backlog.

---

## Automation vs Orchestrator Judgment

**Automate in `kb reflect`:** candidate surfacing, lane grouping, ATS input fields, stale/open threshold detection, telemetry capture.

**Keep manual at orchestrator layer:** coherence validation, intervention choice (guide vs decision vs constraint vs issue), sequencing/trade-offs, and final promotion acceptance.

---

## References

- `.kb/decisions/2026-02-07-orchestrator-reflection-session-protocol.md`
- `.kb/guides/decision-index.md`
- `.kb/investigations/2026-02-07-inv-design-orchestrator-level-reflection-session.md`
- `~/.opencode/skill/worker/kb-reflect/SKILL.md`

---

## History

- **2026-02-07:** Created orchestrator reflection session playbook from accepted protocol decision.
