# Probe: Issue Quality Baseline — Skill Inference "Success" Is False Confidence

**Model:** measurement-honesty
**Date:** 2026-03-21
**Status:** Complete
**claim:** MH-01, MH-02
**verdict:** confirms

---

## Question

The daemon skill inference pipeline reports 0 failures across 641 spawns and 682 unique inferences. Is this "100% success rate" honest signal, or false confidence (invariant 1)? Does treating 0 inference failures as "inference works" commit the absent-signal trap (invariant 2)?

---

## What I Tested

### 1. spawn.skill_inferred events in ~/.orch/events.jsonl

```bash
# 17,199 total events; 10,231 real (non-test); 682 unique real issues
grep -c "skill_inferred" ~/.orch/events.jsonl  # → 17199

# Inference method breakdown (682 unique real issues):
# skill:* label:        83 (12%)
# Title pattern:        84 (12%)
# Description heuristic: 40 (5%)
# Type-based fallback: 475 (69%)
```

### 2. Beads issue corpus analysis across 20 projects (4,036 issues)

```bash
# issue_type field: present in 100% of issues
# Description: present in 66% (median 425 chars)
# Labels: present in 15% (skill:* in only 2%)
# Self-containment Level 1 (title+type): 100%
# Self-containment Level 2 (title+type+desc>50c): 65%
# Self-containment Level 3 (title+type+desc>200c+labels): 8%
```

### 3. Daemon spawn correlation

```bash
# daemon.spawn events: 641
# Spawns with no skill: 0
# Spawned AND inferred: 638 (near-perfect)
# Spawned but NOT inferred: 0
# Inferred but NOT spawned: 44 (mostly test fixtures)
```

### 4. .harness/events.jsonl vs ~/.orch/events.jsonl

```bash
# .harness/events.jsonl: 61 events, ALL gate.fired (no skill_inferred)
# ~/.orch/events.jsonl: 176,038 events including 17,199 skill_inferred
# The two files serve different purposes; skill inference events only go to ~/.orch/
```

---

## What I Observed

### Finding 1: Inference "never fails" because it structurally cannot fail

The inference pipeline has 4 tiers: `skill:*` label → title pattern → description heuristic → type-based fallback. The type-based fallback covers all valid issue types (bug, feature, task, investigation, experiment, question). Since beads always populates `issue_type` (100% of 4,036 issues), the pipeline always reaches a deterministic mapping. The "success rate" is architecturally guaranteed to be 100%.

This is a textbook case of MH-01: **a metric that cannot go red provides no information.** The spawn.skill_inferred event records *that* a skill was inferred, not *whether* it was correct. No mechanism exists to measure inference accuracy.

### Finding 2: 69% of routing is type-based — the coarsest possible signal

The majority of inferences (475/682 = 69%) fall through all heuristics to the type-based fallback. This maps:
- `task` → `feature-impl` (50% of all issues are tasks)
- `feature` → `feature-impl`
- `bug` → `systematic-debugging`
- `investigation` → `investigation`

This means 69% of daemon-routed agents get their skill from a single field with ~6 possible values. Whether this produces the *right* skill is unmeasured. A `task` titled "Design: orchestrator as scoping agent" would be routed to `feature-impl` when it probably needs `architect`.

### Finding 3: The "self-containment" question was misframed

The spawn task asked: "what % of issues are self-contained enough for automated daemon routing?" Answer: **100%** at Level 1 (title + type). But this answers the wrong question. The right question is "what % of daemon routings produce the *correct* skill?" — and this has never been measured.

This is MH-02: **treating absent negative signal as positive signal.** Zero inference failures + zero accuracy measurements = false confidence that routing works correctly.

### Finding 4: Label and description channels are underutilized

- Only 2% of issues corpus-wide have `skill:*` labels (the highest-confidence routing signal)
- Only 15% have any labels at all
- 34% of issues have NO description (orch-go: 48% without description)
- The description heuristic (the richest signal source) only fires for 5% of inferences

The richer inference channels exist but are rarely exercised. Most of the "intelligence" in the inference pipeline is dead code in practice.

---

## Model Impact

- [x] **Confirms** invariant MH-01: The spawn.skill_inferred metric is structurally incapable of producing negative signal. It records that inference occurred, not whether the result was correct. Displaying "0 inference failures" as a metric would create false confidence.

- [x] **Confirms** invariant MH-02: Zero inference failures is being implicitly treated as "inference works." No mechanism measures whether `task` → `feature-impl` was the right routing for any given issue. The absent-signal trap is operating here — the pipeline cannot fail, so absence of failure is being interpreted as evidence of correctness.

- [x] **Extends** model with: **Structural impossibility of failure as a distinct false-confidence pattern.** The existing model examples (merge rate, success rate) show metrics that *happen* to always be green. This case shows a metric that *architecturally guarantees* a positive result via exhaustive fallback chains. The pipeline is designed so that inference never fails — making the "success rate" measurement formally vacuous.

---

## Notes

### Answer to the spawn task's key question

**What % of issues are self-contained enough for automated daemon routing without orchestrator spawn context?**

**100%.** Every issue has a type, and the type-based inference path always produces a skill. No quality gate or template system is needed for the daemon to *route* issues.

**But this is the wrong question.** The question should be: "What % of daemon routings produce the correct skill?" This is currently unmeasurable because:
1. No ground truth exists (no human labels saying "this issue should have used skill X")
2. No outcome correlation exists (does `feature-impl` routing lead to worse outcomes than `architect` routing for the same issue?)
3. The agent.completed events don't record which skill was used vs. which skill should have been used

### Redesign recommendation

Neither a quality gate nor a template system would improve routing accuracy. What would:
1. **Sample accuracy measurement**: Human-label 50 random daemon spawns with "correct skill" and compare to inferred skill
2. **Outcome correlation**: Track completion rate and rework rate per inference method (label vs title vs description vs type-fallback)
3. **Title pattern coverage**: The 12% title-match rate suggests many issues have informative titles that aren't being caught by the current keyword set
