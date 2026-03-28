# Session Synthesis

**Agent:** og-arch-design-research-cycle-28mar-d47b
**Issue:** orch-go-47ppm
**Duration:** 2026-03-28 → 2026-03-28
**Outcome:** success

---

## TLDR

Designed the research cycle for automated hypothesis testing against model claims. The key finding: it's not a tight autoresearch-style inner loop — it's a cross-session knowledge cycle (orient→research→probe→merge→orient) where the automation is in visibility and context assembly, not execution speed. Three-component design: `orch research` command, research skill, orient claim-status integration.

---

## Plain-Language Summary

The orch-go knowledge system has models with testable claims (like "named gaps compose better than conclusions") and probes that test those claims with experiments. The problem: nobody can easily see which claims have been tested and which haven't, and setting up the context for a research agent takes 10+ minutes of manual assembly every time.

The design solves this with three pieces: a new `orch research` command that reads model claims tables and shows their status, a research skill that gives agents a structured protocol for running experiments, and an orient integration that surfaces "you have 4 untested claims in named-incompleteness" at every session start. The cycle is driven by human attention (the orchestrator reads orient and decides what to test), not by automation — because automated research spawning would produce garbage probes to satisfy the system rather than genuine experiments.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-28-inv-design-research-cycle-autoresearch-style.md` — Full design investigation with 6 findings, 6 forks navigated, implementation recommendations
- `.orch/workspace/og-arch-design-research-cycle-28mar-d47b/VERIFICATION_SPEC.yaml` — Verification specification
- `.orch/workspace/og-arch-design-research-cycle-28mar-d47b/SYNTHESIS.md` — This file
- `.orch/workspace/og-arch-design-research-cycle-28mar-d47b/BRIEF.md` — Comprehension brief

### Issues Created
- `orch-go-jhz2s` — Implement orch research command with claims parser and spawn mode
- `orch-go-2o0my` — Create research skill for structured probe protocol
- `orch-go-bhois` — Wire research surface (claim status) into orch orient
- `orch-go-9ac1e` — Integration: end-to-end research cycle
- `orch-go-ugaml` — Question: claims table format standardization
- `orch-go-7n9we` — Question: daemon integration boundary

---

## Evidence (What Was Observed)

- 5 models examined for claims table format: named-incompleteness (6 claims), compositional-accretion (6), knowledge-accretion (14+), defect-class-taxonomy (7), completion-verification (4+). Format varies significantly.
- 4 probe files in named-incompleteness show the probe artifact format works well for hypothesis testing
- --loop controller is ~250 lines composing 3 existing primitives — confirms thin composition layers work
- --explore flag precedent shows how spawn flags transform agent behavior (swap skill to exploration-orchestrator)
- The 373-paper bibliometrics probe (manual workflow) produced a significant result — execution isn't the bottleneck
- NI Failure Mode 4 (compliance-driven probes) directly argues against daemon-automated research

---

## Architectural Choices

### Cross-session cycle over tight inner loop
- **What I chose:** Research cycle operates across sessions via orient visibility
- **What I rejected:** Autoresearch-style tight loop with automated iteration
- **Why:** Knowledge experiments take hours, not minutes. The "loop" is the orchestrator returning to orient and seeing updated claim status. Tight iteration would produce compliance-driven probes (NI FM4).
- **Risk accepted:** Cycle depends on orchestrator attention — if orient is ignored, claims stay untested

### Manual trigger over daemon automation
- **What I chose:** `orch research <model> <claim>` triggered by orchestrator
- **What I rejected:** Daemon auto-spawning research for untested claims
- **Why:** Research quality requires judgment about which claims are ripe, which methods are informative, how to interpret ambiguity. Daemon can't exercise this judgment.
- **Risk accepted:** Research velocity limited by orchestrator attention. Mitigated by orient visibility.

### Verdict existence over statistical parsing
- **What I chose:** Binary eval: probe has clear verdict (confirms/disconfirms/extends) or not
- **What I rejected:** Parsing p-values, effect sizes, confidence intervals from probes
- **Why:** The agent provides the statistical judgment. The system only needs to verify judgment was exercised. Follows --loop precedent (exit code = simplest eval).
- **Risk accepted:** Low-quality verdicts pass the check. Mitigated by research skill constraints.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-28-inv-design-research-cycle-autoresearch-style.md` — Full architect investigation

### Decisions Made
- Decision 1: Three-component design (command + skill + orient) because it composes existing primitives without new subsystems
- Decision 2: Manual trigger V1 because NI Failure Mode 4 argues against automated research
- Decision 3: Claim table IS the hypothesis specification — no separate hypothesis bank

### Constraints Discovered
- Claims table format varies across models — standardization needed for reliable parsing
- Daemon-automated research risks compliance-driven probes per NI FM4
- Cross-session cycles depend on orient visibility being noticed and acted on

---

## Next (What Should Happen)

**Recommendation:** close — design is complete, implementation issues created

### If Close
- [x] All deliverables complete (investigation, SYNTHESIS, BRIEF, VERIFICATION_SPEC)
- [x] Implementation issues created (orch-go-jhz2s, orch-go-2o0my, orch-go-bhois, orch-go-9ac1e)
- [x] Blocking questions surfaced (orch-go-ugaml, orch-go-7n9we)
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-47ppm`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should probes enter the comprehension queue? Currently they merge into models but don't generate briefs. If they did, the comprehension cycle would naturally surface research results.
- Could claim status be exposed in the sketchybar widget? "3 untested claims" as persistent ambient information.
- What's the relationship between `orch compose` (cluster briefs) and research cycle (cluster claims)? Both aggregate across sessions.

**What remains unclear:**
- How many total testable claims exist across all models (only sampled 5)
- Whether the claims table format can be standardized without significant migration cost
- Whether the research skill needs to be a new skill or could be a mode of the investigation skill

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace for design completeness checks and manual verification requirements.

---

## Friction

- No friction — smooth session

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-research-cycle-28mar-d47b/`
**Investigation:** `.kb/investigations/2026-03-28-inv-design-research-cycle-autoresearch-style.md`
**Beads:** `bd show orch-go-47ppm`
