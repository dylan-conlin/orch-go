# Probe: Productive vs Unproductive Frustration — Can the System Distinguish?

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-27
**Status:** Complete
**claim:** Frustration boundaries should trigger session restart when compound signals fire (2+ of thrashing, stuck, circular, failure)
**verdict:** extends

---

## Question

The frustration boundary design (probe 2026-03-27, investigation orch-go-u5x4g) proposes compound signal detection → session boundary. The model treats ALL detected frustration as a restart trigger. But Dylan's lived experience and the resistance thread suggest some frustration is productive — "just-hard" work where understanding is forming but the process feels painful. Does the empirical record confirm that frustration correlates with unproductive sessions? Or does frustration sometimes signal productive struggle that a boundary would interrupt?

Specifically testing:
1. Does friction/frustration in session debriefs correlate with LOWER or HIGHER session productivity?
2. Are the two frustration types (bolted-on vs just-hard) distinguishable from observable signals?
3. Should the compound frustration detector have a "hard but progressing" exception?

---

## What I Tested

### Test 1: Friction-Productivity Correlation Across 18 Session Debriefs

Counted friction reports, learnings (What We Learned bullets), completed items, and spawns per session across all March 2026 debriefs.

```bash
# Counted per session:
# Friction reports (grep -c "Friction:"), Learnings (bold bullets in What We Learned),
# Completed items (grep -c "Completed:"), Spawned items (grep -c "Spawned:")
# Across .kb/sessions/2026-03-*-debrief.md
```

### Test 2: Friction Type Classification

Extracted friction categories per session (gap, ceremony, tooling, bug, none) and cross-referenced with session learning quality.

### Test 3: Abandoned/Blocked Sessions Pattern

Searched all debriefs for "Abandoned", "BLOCKED", "stuck", "stall" patterns — these represent the unproductive frustration mode where the system WAS fighting the agent.

### Test 4: Resistance Thread Evidence

Read the resistance thread (2026-03-27) and five-element product surface thread for Dylan's observed distinction between flow and grind modes, and the fix-count directionality signal.

---

## What I Observed

### Finding 1: High-Friction Sessions Are the MOST Productive

| Session | Friction Reports | Learnings | Spawns | Completions |
|---------|-----------------|-----------|--------|-------------|
| Mar 19  | 7               | 15        | 97     | 47          |
| Mar 11  | 3               | 16        | 98     | 5           |
| Mar 10  | 2               | 17        | 57     | 10          |
| Mar 12  | 2               | 17        | 78     | 9           |
| Mar 20  | 2               | 18        | 63     | 31          |
| **Mean (2+ friction)** | **3.2** | **16.6** | **78.6** | **20.4** |
| Mar 13  | 0               | 7         | 17     | 4           |
| Mar 18  | 0               | 4         | 6      | 0           |
| Mar 21  | 0               | 7         | 44     | 14          |
| Mar 22  | 0               | 9         | 19     | 13          |
| **Mean (0 friction)** | **0** | **6.8** | **21.5** | **7.8** |

Sessions with 2+ friction reports average **16.6 learnings** vs **6.8** for zero-friction sessions (2.4x). The correlation is positive — friction correlates with MORE productivity, not less.

### Finding 2: Two Frustration Types Are Observable in Debrief Data

**Type A: Bolted-on (system fighting you)**
- Governance hooks blocking valid work (Mar 10, 11, 22: "governance hook blocked the exact files this task requires")
- Tool quirks (Mar 9: "sed with tab escapes on macOS produced literal t")
- Process ceremony exceeding the fix (Mar 11: "governance file protection blocked fix")
- **Observable signal**: friction type is `gap` or `ceremony`, issue gets BLOCKED status, no learning produced by the blocked work itself

**Type B: Just-hard (productive struggle)**
- Mar 10: "Publication focus abandoned — lost trust in the system's self-assessment" → produced "Closed loop risk" and "Independent disconfirmation" learnings (two of the deepest insights)
- Mar 11: Heavy governance friction session → produced "Measurement as first-class harness layer" thread
- Mar 9: Knowledge accretion formalization struggle → produced the model itself
- **Observable signal**: friction type is `tooling` or `none` (the frustration isn't reported as friction — it's in the WORK itself), learnings have high abstraction level, threads/threads entries created

### Finding 3: Abandoned Sessions Show Pure Unproductive Frustration

From debriefs:
- Feb 28: Abandoned 10 agents — "Stuck at Exploration phase, never progressed"
- Mar 4: Abandoned orch-go-w7xe — "stuck on skillc test"
- Mar 11: Abandoned orch-go-orlcp (no reason given)

These share a pattern: **no phase transitions, no artifacts created, no learning externalized**. The agent is active but nothing is landing.

### Finding 4: Fix-Count Direction Is the Key Discriminator

From the resistance thread, Dylan already identified this:
- **Bolted-on**: fix count goes UP (each fix spawns more fixes, backlog grows)
- **Just-hard**: fix count goes DOWN (each fix makes remaining problems fewer/clearer)

The beads data already tracks this — issue chains (does fix A spawn fix B?) and area label concentration. A session where the same area label gets 4+ fixes in 3 weeks with each spawning follow-ups is bolted-on. A session where one hard fix stays solved is just-hard.

### Finding 5: Progress Artifacts Are the Best Real-Time Discriminator

For the compound frustration detector, the most reliable in-session signal is whether the agent is producing durable artifacts:
- **Phase transitions happening** (even slowly) → progressing
- **Knowledge artifacts being created** (investigations, probes, thread entries) → understanding forming
- **Commits landing** (non-zero commit count) → work is crystallizing
- **File reads deepening** (reading different sections of same file vs repeating same grep) → learning

If frustration compound signal fires AND the session has artifacts + phase transitions, the frustration is productive. If it fires AND no artifacts + stuck phase, the frustration is unproductive.

---

## Model Impact

- [ ] **Confirms** invariant: Framing is stronger than instructions — mid-session frustration that's producing understanding should NOT be interrupted by a boundary
- [x] **Extends** model with: Frustration boundary detection needs a **productivity qualifier** before triggering. Two-type taxonomy:

**Proposed extension to the frustration boundary design:**

1. **Unproductive frustration (bolted-on)**: Compound signal fires + no artifact creation + no phase transitions + fix count directionally increasing. → TRIGGER boundary (interactive: propose, headless: respawn)

2. **Productive frustration (just-hard)**: Compound signal fires + artifacts being created + phases advancing (even slowly) + fix count directionally decreasing. → SURFACE signal as "hard work in progress" but DO NOT trigger boundary

The productivity qualifier checks:
- `artifact_count > 0` in last 15 minutes (commits, investigation updates, thread entries, probe writes)
- `phase_transitions > 0` since frustration signals started
- `fix_direction` from beads issue chain analysis (are follow-up issues being created, or are existing ones being resolved?)

**Cost asymmetry justification:**
- For headless workers: false-positive boundary cost is LOW (one respawn, ~5 min). Productivity qualifier is nice-to-have.
- For interactive sessions: false-positive boundary cost is HIGH (interrupts Dylan's train of thought during a breakthrough). Productivity qualifier is REQUIRED.

**Session types table update:**

| Session Type | Boundary Trigger | Productivity Qualifier |
|--------------|------------------|----------------------|
| Worker (headless) | Compound signal (2+) | Optional (respawn is cheap) |
| Interactive (Dylan) | Compound signal (3+) + NO productivity qualifier | Required (interruption is costly) |

---

## Notes

The data strongly suggests the existing frustration boundary design should NOT treat all detected frustration as a restart trigger. The highest-productivity sessions in the system's history (Mar 10, 11, 12, 19, 20) were ALL high-friction sessions. Indiscriminate frustration boundaries would have interrupted the sessions that produced the deepest insights.

The key insight: **frustration is not the enemy — stagnation is**. The signal to act on isn't "is the user/agent frustrated?" but "is the user/agent frustrated AND not making progress?" The compound frustration detector plus a progress qualifier captures this distinction without requiring the system to understand the CONTENT of the frustration.

This connects to the five-element product surface: the "resistance" element should surface the TYPE of resistance (bolted-on vs just-hard), not just its presence. The fix-count direction signal could be the ambient indicator Dylan described wanting — visible in the product surface without having to ask.
