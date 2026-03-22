# Probe: Attractor Decay — How Fast Does Coordination Degrade When Attractors Go Stale?

**Model:** coordination
**Date:** 2026-03-22
**Status:** Complete
**claim:** CO-02
**verdict:** extends

---

## Question

Claim 2 says "Structural placement prevents conflicts completely" (20/20 SUCCESS). But what happens when the codebase changes underneath the placement attractors? How fast does the 100% success rate degrade? Is the degradation gradual or cliff-edge?

This tests the implied SENSOR requirement: attractors are only as good as their accuracy. The governance self-accretion probe (launchd post-mortem: 186 investigations reinforced wrong premise) showed learned attractors can go stale. How many codebase changes before attractors become ineffective?

---

## What I Tested

Three phases of codebase mutation with ORIGINAL (now stale) placement prompts:

**Phase 1: Rename** — Rename `FormatDurationShort` -> `FormatElapsed` and corresponding test. Agent A's attractor ("after FormatDurationShort") now references a non-existent function.

**Phase 2: Reorganize** — Move `StripANSI` and `ansiRegex` to a new `ansi.go` file. Agent B's attractor ("after StripANSI in display.go") now points to a function in the wrong file.

**Phase 3: Add alternatives** — Add `FormatDurationCompact` and `FormatTimestamp` functions. Creates new semantically plausible insertion points that compete with the (stale) attractor targets.

N=3 trials per phase, simple task only, haiku model, placement condition with original prompts.

```bash
cd experiments/coordination-demo/redesign && bash run-decay.sh --trials 3
```

---

## What I Observed

### Degradation Curve: No Degradation

| Phase | Success Rate | vs 100% baseline |
|-------|-------------|------------------|
| baseline | 100% | (20/20 from prior experiment) |
| rename | **100%** | (3/3) |
| reorganize | **100%** | (3/3) |
| alternatives | **100%** | (3/3) |

**All 9 trials succeeded.** Zero degradation across all three mutation types. All 18 agents scored 4/4 individually (completion, build, tests, spec match). The hypothesis of cliff-edge failure was completely wrong.

### Agent Adaptation Behavior

The agents did NOT blindly follow stale attractors. They READ the codebase and ADAPTED:

**Phase 1 (rename — anchor function renamed):**
- Agent A was told "place after `FormatDurationShort`" but that function was renamed to `FormatElapsed`
- Agent A found `FormatElapsed` at the same structural position and used it as anchor (3/3 trials)
- Agent B's anchor (`StripANSI`) was unchanged by the rename — placed correctly (3/3)
- **Mechanism: Semantic adaptation** — agent resolved the intent ("after the short-format duration function") not the literal name

**Phase 2 (reorganize — anchor function moved to different file):**
- Agent B was told "place after `StripANSI` in display.go (BEFORE `FormatDuration`)" but `StripANSI` was moved to `ansi.go`
- Agent B placed `FormatRate` after `ShortID` (the function that was before `StripANSI`'s old location), correctly before `FormatDuration` (3/3 trials)
- **Mechanism: Anchor redundancy** — the secondary anchor ("BEFORE `FormatDuration`") survived the reorganization and guided placement. The agent used the surviving anchor when the primary anchor was gone.

**Phase 3 (alternatives — competing insertion points added):**
- Agent A was told "place after `FormatDurationShort`" — function still exists, but `FormatDurationCompact` and `FormatTimestamp` were added after it
- Agent A placed `FormatBytes` immediately after `FormatDurationShort`, BEFORE the new competing functions (3/3 trials)
- **Mechanism: Literal compliance** — attractor was still valid, agent followed it precisely. New competing functions did not distract.

### Two Resilience Mechanisms

1. **Semantic adaptation**: When an anchor function is renamed or removed, the agent finds the semantically equivalent location by reading the actual code structure. The attractor's *intent* (which region of the file) is more durable than its *literal reference* (specific function name).

2. **Anchor redundancy**: Well-designed placement instructions include multiple anchors (e.g., "after X, BEFORE Y"). Losing one anchor still leaves the other. In Phase 2, the secondary anchor "BEFORE `FormatDuration`" survived when the primary anchor `StripANSI` was removed.

### Why Coordination Survived

The coordination value of attractors is not the specific function name — it's the **region assignment**. Agent A is always pointed to the "end of formatting functions" region. Agent B is always pointed to the "after utility functions" region. As long as these regions remain distinct, the specific anchor names don't matter. The mutations changed anchor names and file structure but preserved the region separation.

### Results directory

`experiments/coordination-demo/redesign/results/20260322-154258/`

---

## Model Impact

- [x] **Extends** model with: attractors are MORE resilient than expected — agents compensate for stale anchors through semantic adaptation and anchor redundancy. The implied sensor requirement is weaker than hypothesized.

### Specific extensions to CO-02

1. **Attractor resilience**: Placement attractors do NOT require exact function name accuracy. They tolerate renames (semantic adaptation), file reorganization (anchor redundancy), and addition of competing insertion points (literal compliance with nearest valid anchor).

2. **Region separation is the load-bearing property**: The coordination value of attractors comes from assigning agents to different regions of the codebase, not from the specific anchor function name. Region separation persists through codebase mutations that preserve overall structure.

3. **Sensor requirement is weaker than hypothesized**: The experiment was designed to show that stale attractors need real-time freshness sensors. Instead, agents themselves act as adaptive sensors — they read the codebase and compensate for attractor drift. The sensor is embedded in the agent, not needed in the infrastructure.

4. **Boundary condition (untested)**: These mutations preserved overall file structure and semantic grouping. A more radical mutation (complete file rewrite, function deletion without replacement) might break adaptation. The resilience finding applies to incremental codebase evolution, not wholesale restructuring.

### Open question answered

The model's open question "Can attractor-based coordination degrade? (What happens when structural destinations become stale or misaligned with evolving requirements?)" is now answered: **Not for incremental codebase changes.** Agents adapt to stale anchors through semantic resolution and anchor redundancy. Degradation may occur with wholesale restructuring (untested).

---

## Notes

The hypothesis (cliff-edge failure in Phase 1) was wrong in an interesting way. It assumed agents would blindly follow the literal attractor instruction. Instead, agents are adaptive — they read the actual code and find the location that matches the *intent* of the placement instruction, even when the literal anchor doesn't exist.

This is consistent with Claim 3 (individual agent capability is not the bottleneck) — agents are individually capable enough to adapt placement instructions to changed code. The failure in coordination experiments happens when BOTH agents adapt to the SAME location (semantic correctness bias), not when individual agents can't find appropriate locations.

The attractor decay experiment tested a different failure mode than the original coordination experiment:
- **Original**: Both agents have the SAME semantic target (both gravitate to "after FormatDurationShort") → 100% conflict
- **Decay**: Agents have DIFFERENT regional targets, one target has been moved → agents adapt and maintain separation → 0% conflict

This confirms that the coordination problem is fundamentally about **agent separation**, not about **anchor accuracy**.
