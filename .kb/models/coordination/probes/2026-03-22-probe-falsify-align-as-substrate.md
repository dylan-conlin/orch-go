# Probe: Falsify Align-as-Substrate

**Model:** coordination
**Date:** 2026-03-22
**Status:** Complete
**claim:** (implicit — "Align is the meta-primitive/substrate" from model.md line 75)
**verdict:** extends

---

## Question

The coordination model claims Align is "the meta-primitive" — a substrate on which Route, Sequence, and Throttle operate. Without Align, the other three primitives drift: gates measure wrong things (Throttle), routes go stale (Route), sequence steps become wrong (Sequence). Removing any other primitive leaves the rest intact.

Can we find cases where Align is broken but Route/Sequence/Throttle still function correctly? If so, the substrate claim is falsified and Align is a peer primitive.

---

## What I Tested

Classified 15 failure cases from four data sources into three categories:
- **Case 1:** Align broken, Route/Sequence/Throttle hold (falsifies substrate)
- **Case 2:** Route/Sequence/Throttle broken AND Align broken (supports substrate via co-occurrence)
- **Case 3:** Route/Sequence/Throttle broken, Align intact (expected — peers break independently)

Data sources examined:
- 14 MAST failure modes (Berkeley, 1642 traces)
- McEntire controlled experiment (28 tasks × 4 architectures)
- orch-go 80-trial experiment (messaging/context-share/no-coord/placement conditions)
- orch-go production failures (2 post-mortems, 10 agent failure investigations)

---

## What I Observed

### Case 1: Align Broken, Route/Sequence/Throttle Hold (5 cases)

| # | Source | What Happened | Primitives |
|---|--------|---------------|------------|
| 1 | MAST FM-1.1 | Agent disobeys task specification. Correctly routed, correctly sequenced, velocity controlled — but does the wrong thing. | Align: BROKEN. Route/Sequence/Throttle: mechanically hold |
| 2 | McEntire hierarchical (36% failures) | Human orchestrator routes tasks to correct agents, orders them, controls budget. Agents diverge from human intent on 36% of tasks. | Align: BROKEN (partial). Route/Sequence/Throttle: hold |
| 3 | launchd post-mortem | 186 investigations correctly routed and sequenced. Shared model of correctness ("launchd is right") was wrong. All investigations reinforced wrong premise. | Align: BROKEN. Route/Sequence: hold. Throttle: hold (each investigation controlled) |
| 4 | orch-go competing instructions | Orchestrator correctly routed to task, sequenced properly. System prompt (500 words promoting Task tool) overrode skill constraint (30 words restricting Task tool). Agent did wrong actions despite correct routing. | Align: BROKEN (conflicting signals). Route/Sequence: hold |
| 5 | orch-go stale knowledge cascade | Orchestrator correctly routed to cross-project debugging. Used stale knowledge ("OshCut uses scraper") when project had moved to HTTP API months earlier. Cascaded into wrong-direction investigation. | Align: BROKEN (stale model). Route/Sequence: hold |

**Observation:** In all 5 cases, Route/Sequence/Throttle operated mechanically — messages were delivered, steps were ordered, velocity was controlled. But the outputs were wrong because agents' model of correctness was wrong. The primitives *ran* but didn't *produce useful coordination*.

### Case 2: Route/Sequence/Throttle Broken AND Align Broken (4 cases)

| # | Source | What Happened | Primitives |
|---|--------|---------------|------------|
| 6 | System spiral post-mortem (Dec 27-Jan 2) | Agents modified their own ground truth (347 commits/6 days). Each fix changed what "correct" meant. Route/Sequence/Throttle all broke downstream. | All four broken. Align broke first → others cascaded. |
| 7 | McEntire pipeline (0% success) | $50 budget consumed on planning (Throttle), agents rejected 87% of submissions with zero basis (Align), contradictory governance 28s apart (Sequence), agents redid each other's work (Route). | All four broken simultaneously. |
| 8 | orch-go agent ignores architect design | Agent built pkg/harness in orch-go despite architect designing separate repo. Spawn context only injected investigation title, not conclusions (Align broken). Agent also routed work to wrong location (Route broken). | Align broken + Route broken (co-occurrence) |
| 9 | 80-trial messaging condition | Agents had state alignment (knew each other's plans) and task alignment (6/6 individual). But task instruction ("place after FormatDurationShort") conflicted with coordination instruction ("don't conflict"). Both agents chose identical insertion point. | System-level Align: BROKEN (conflicting instructions). Route: BROKEN (no structural separation). |

**Observation:** Cases 6 and 7 strongly support the substrate claim — Align broke first and cascaded into other failures. Cases 8 and 9 show co-occurrence without clear causal direction.

### Case 3: Route/Sequence/Throttle Broken, Align Intact (6 cases)

| # | Source | What Happened | Primitives |
|---|--------|---------------|------------|
| 10 | 80-trial no-coord condition | Agents individually perfect (160/160 scored 6/6). No routing mechanism. 100% merge conflict. | Align: INTACT (individual). Route: ABSENT. |
| 11 | MAST FM-1.3 (step repetition) | Agent understands task correctly but repeats steps. | Align: can be intact. Sequence: BROKEN. |
| 12 | MAST FM-3.1 (premature termination) | Agent knows what correct means but stops too early. | Align: can be intact. Throttle: BROKEN. |
| 13 | orch-go cross-project skill injection | Skill routing failed for non-orch-go projects (CLAUDE_CONTEXT collision). Agents may have understood tasks but couldn't receive correct skill. | Align: potentially intact. Route: BROKEN. |
| 14 | orch-go duplicate spawn race | TOCTOU race in manual spawn path. System understands correct behavior, but sequence/atomicity not enforced. | Align: INTACT. Sequence: BROKEN. |
| 15 | orch-go user-as-message-bus | Each orchestrator's internal model was correct. No cross-session communication channel existed. Human relayed 5+ messages between orchestrators. | Align: INTACT (per-orchestrator). Route: BROKEN (cross-session). |

**Observation:** These confirm Route/Sequence/Throttle can break independently while Align holds. Expected for any non-substrate relationship.

---

## Key Analytical Finding: Mechanical vs Functional Operation

The five Case 1 examples show that Route/Sequence/Throttle CAN operate when Align is broken — they deliver messages, order steps, and limit velocity. But the outputs are wrong.

This reveals an ambiguity in "substrate":

| Interpretation | Verdict |
|---------------|---------|
| **Substrate = can't mechanically run without** | FALSIFIED by Cases 1-5. Route/Sequence/Throttle run fine without Align. |
| **Substrate = can't produce value without** | SUPPORTED — in all Case 1 examples, the primitives ran but produced garbage. |

The strongest challenge to even the "produce value" interpretation: **McEntire hierarchical achieved 64% success with only partial Align.** This means Route/Sequence/Throttle CAN produce value with imperfect Align. The relationship is proportional, not binary.

### The Multiplier Model

Success degrades proportionally with Align quality, not catastrophically:

```
Coordination_value = Route_quality × Sequence_quality × Throttle_quality × Align_quality
```

- Align_quality = 0 → everything is 0 (substrate interpretation holds here)
- Align_quality = 0.6 → 64% success (McEntire hierarchical)
- Align_quality = 1.0 → 100% success possible (when other primitives also hold)

This is a **multiplier**, not a substrate. A substrate is binary (present/absent). Align has a continuous effect.

### The 80-trial Messaging Condition: Align's Internal Conflict

The messaging condition reveals Align itself can have internal contradictions:
- **Task Align:** "Place after FormatDurationShort" (both agents agree ✅)
- **Coordination Align:** "Don't conflict with the other agent" (both agents understand ✅)
- **Resolution:** Task instruction > coordination instruction (18/20 trials chose semantic correctness over coordination)

This supports the model's open question about decomposing Align into sub-primitives (task alignment vs coordination alignment vs system alignment).

---

## Model Impact

- [ ] **Confirms** invariant: Align is asymmetrically important (7/14 MAST modes, dominant in orch-go failures)
- [ ] **Contradicts** invariant: "substrate" is too strong — Route/Sequence/Throttle mechanically operate without Align
- [x] **Extends** model with: Align is a **multiplier** on the other three primitives' effectiveness, not a substrate. Route/Sequence/Throttle can mechanically operate without Align, but their coordination value degrades proportionally to Align quality. The term "prerequisite" or "validity condition" is more precise than "substrate."

### Recommended Model Updates

1. **Replace "meta-primitive" / "substrate" language** with "multiplier" or "validity condition":
   - Before: "Align is the meta-primitive. Without Align, the other three primitives drift."
   - After: "Align is the validity condition. Route/Sequence/Throttle operate mechanically without it, but their coordination value is proportional to Align quality."

2. **Add the mechanical/functional distinction** to the primitives section:
   - Mechanical operation (message delivery, step ordering, velocity limiting) is independent across all four primitives
   - Functional value (correct coordination outcomes) requires Align as a multiplier

3. **Note Align's internal decomposition evidence** (already in open questions, now with direct evidence):
   - Task alignment (agent understands its own task)
   - State alignment (agent knows system state)
   - Coordination alignment (agent adjusts behavior for multi-agent context)
   - The 80-trial messaging condition shows task alignment defeating coordination alignment (18/20 trials)

---

## Notes

**Evidence quality:** Case 1 examples are the strongest finding. Five independent cases from three data sources (MAST academic taxonomy, McEntire controlled experiment, orch-go production) all show the same pattern: Route/Sequence/Throttle mechanically holding while Align is broken.

**What would change this verdict:**
- Finding a case where broken Align causes Route/Sequence/Throttle to mechanically fail (not just produce wrong outputs) would support the substrate claim
- Finding a case where perfect Align + broken Route still produces correct outcomes would weaken the multiplier model

**Remaining gap:** The multiplier model predicts that Align_quality × 0 (other primitive absent) = 0 value. The 80-trial placement condition tests Route alone (Align was incidentally intact). We don't have a case testing Throttle=0 with Align=1 or Sequence=0 with Align=1 in isolation.
