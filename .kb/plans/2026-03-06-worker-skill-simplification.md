## Summary (D.E.K.N.)

**Delta:** Apply the orchestrator skill simplification playbook (2,368→422 lines, 82% reduction, measurably better) to all worker skills. The same structural problem — behavioral weight crowding out knowledge and stance — exists in every worker skill.

**Evidence:** Skill-content-transfer model (3-type taxonomy validated across 162+ trials), worker skill audit (all 5 skills exceed ≤4 behavioral threshold), orchestrator simplification results (+5 knowledge lift, 0%→83% stance lift on S09), human calibration (rho=0.637 validates measurement), investigation stance experiment (54 trials, zero lift — action directives don't transfer).

**Knowledge:** Worker skills have the same interleaving problem: behavioral mandates embedded within knowledge sections rather than isolated. Self-review checklists are the single largest behavioral block in every skill (10-15 items each). **Critical refinement from Phase 0:** Not all stances transfer equally. Attention primers (change how agents see) transfer in --print mode. Action directives (tell agents what to do) do not. Each skill's stance must be classified and reframed if needed. The playbook is: strip behavioral to hooks, keep knowledge + attention-primer stances in the skill document, reframe action-directive stances.

**Next:** Extract self-review to hooks (cross-cutting), then simplify each skill — with stance classification/reframing as part of each simplification.

---

# Plan: Worker Skill Simplification

**Date:** 2026-03-06
**Status:** Phase 0-1 complete, Phase 2 unblocked (6 simplifications ready), Phase 3 in progress
**Owner:** Dylan

**Extracted-From:** Skill-content-transfer model, worker skill audit (2026-03-06), comprehension measurement program (Phases 1-2)
**Thread:** "Stance as attention priming — what agents don't see"

---

## Objective

Simplify all worker skills using the validated playbook: strip behavioral weight to hooks, preserve knowledge and stance, target ≤500 lines and ≤4 behavioral norms per skill. Success = each simplified skill scores equal or better than its predecessor on contrastive scenarios, with behavioral enforcement maintained by hooks.

---

## Substrate Consulted

- **Models:** Skill-content-transfer (3-type taxonomy, dilution thresholds, stance as cross-source reasoning primer)
- **Investigations:** Worker skill section taxonomy audit (2026-03-06), comprehension calibration (2026-03-05), stance generalization (2026-03-06), human calibration (2026-03-06)
- **Decisions:** Stance items non-removable (kb-3f85c9), hook-enforced behaviors must not appear in skill text (Invariant 4)
- **Constraints:** ≤500 lines / 5,000 tokens (Invariant 1), ≤4 behavioral norms (Invariant 2), auto-scorer validated only for cross-source scenarios (Invariant 7)
- **Evidence:** 108+ trials across 5 scenarios, 24 human-calibrated responses

---

## Decision Points

### Decision 1: Self-review extraction scope

**Context:** Every skill has a 10-15 item self-review checklist — pure behavioral weight. Moving to a hook drops behavioral counts ~50%. But hooks are binary (pass/fail), while self-review in skill text encourages reflection.

**Options:**
- **A: Full extraction to hook** — All self-review items become completion-time hook checks. Pros: maximum behavioral weight reduction. Cons: agents lose the "reflection" prompt; hook must cover all check types.
- **B: Partial extraction** — Top 4 most-violated items become hook gates, remainder stays as advisory text (not MUST/NEVER). Pros: keeps reflection, reduces behavioral count. Cons: residual behavioral weight.
- **C: Replace with stance** — Remove self-review checklist, add a stance line like "verify your own work as if reviewing someone else's." Pros: lightest touch, tests stance-as-self-review. Cons: unproven, may not catch specific issues.

**Recommendation:** A first. The hook provides deterministic enforcement; stance can be tested separately. If agents show quality regression post-extraction, fall back to B.

**Status:** Open

### Decision 2: Feature-impl stance content

**Context:** Feature-impl has 1 stance line (Harm Assessment). It's the most-spawned worker skill running with essentially no epistemic orientation. What stance should it have?

**Options:**
- **A: "Build what's needed, not what's asked"** — Orients toward understanding intent behind requirements.
- **B: "Every change has consumers you haven't met"** — Cross-source reasoning primer (mirrors orchestrator stance, proven mechanism).
- **C: "Test the assumption before building on it"** — Scientific method orientation (mirrors investigation stance).
- **D: Discover through contrastive experiment** — Write 2-3 candidate stances, test each with a contrastive scenario, pick the one that discriminates.

**Recommendation:** D. We have the infrastructure. Don't guess — measure.

**Status:** Open, blocked by Phase 0

### Decision 3: Density vs count threshold

**Context:** The model says dilution begins at 5+ co-resident behavioral constraints. But experiment skill has 18 mandates in 294 lines while systematic-debugging has 10 in 802 lines. Is dilution about absolute count or constraint density (ratio of mandates to total content)?

**Options:**
- **A: Count-based (current model)** — ≤4 absolute, regardless of skill length. Simple, conservative.
- **B: Density-based** — Ratio threshold (e.g., ≤1 mandate per 75 lines). Allows longer skills more constraints.
- **C: Both** — Must satisfy both count ≤4 AND density ≤1:75.

**Recommendation:** Defer to research (orch-go-6rb93). Use count-based for now — it's conservative and proven. If density research shows the threshold is ratio-dependent, update the model.

**Status:** Open, deferred to research

---

## Phases

### Phase 0: Validation Gate

**Goal:** Confirm that worker stances transfer the same way the orchestrator stance does.

**Deliverables:**
- Contrastive scenario for investigation skill ("test before concluding")
- N=6 bare vs with-stance results
- Go/no-go decision for the simplification program

**Exit criteria:**
- With-stance shows measurable lift over bare on at least one indicator (N=6)
- OR: no lift detected, triggering re-evaluation of the entire program

**Beads:** orch-go-uofaw (closed)

**Result:** Gate passed with refinement. Investigation stance ("test before concluding") shows ZERO lift across 54 trials. The critical finding: the investigation stance is an **action directive** (tells agents what to do), not an **attention primer** (changes how agents see). Action directives have no leverage in --print mode. This doesn't block simplification — behavioral weight reduction is justified independently — but it refines the stance preservation strategy. Each skill's stance must be classified as attention primer vs action directive, and action directives should be reframed.

**Stance classification from gate experiment:**

| Skill | Current Stance | Type | Transfer Expected? |
|-------|---------------|------|-------------------|
| orchestrator | "Look for implicit assumptions" | Attention primer | Yes (validated, +4-7 lift) |
| investigation | "Test before concluding" | Action directive | No (validated, 0 lift) |
| systematic-debugging | "Understand before fixing" | Attention primer | Likely (reframes perception) |
| architect | "Decide what should exist" | Attention primer | Likely (reframes perception) |
| experiment | "This is science, not exploration" | Attention primer | Likely (reframes perception) |
| feature-impl | (none) | N/A | Needs stance injection |

**Probe:** `.kb/models/skill-content-transfer/probes/2026-03-06-probe-investigation-stance-transfer.md`

### Phase 1: Infrastructure — Self-Review Hook

**Goal:** Extract self-review checklists from all skills into a deterministic completion-time hook, reducing behavioral weight ~50% across the board.

**Deliverables:**
- Completion-time hook that runs self-review checks
- Self-review sections removed from all skill documents
- Verification: behavioral mandate count drops in each skill

**Exit criteria:**
- Hook passes on well-formed completions, fails on missing deliverables
- No quality regression on existing contrastive scenarios (run baseline before/after)

**Depends on:** Phase 0 (need to know the playbook works)

**Beads:** orch-go-wq7kc

### Phase 2: Skill Simplification (6 skills, parallel)

**Goal:** Apply the playbook to each worker skill: strip remaining behavioral weight to hooks, preserve knowledge + stance, target ≤500 lines / ≤4 norms.

**Simplification order** (by tractability, not priority — all can run parallel):

| Skill | Current | Target | Key Intervention | Beads |
|-------|---------|--------|-----------------|-------|
| investigation | 266 lines, 6+ behavioral, 4 stance | ≤266, ≤4 behavioral | Strip self-review + leave-it-better mandate | orch-go-56axh |
| experiment | 294 lines, 8+ behavioral, 4 stance | ≤294, ≤4 behavioral | Consolidate 11-item Boundaries block to 3-4 norms | orch-go-4snnl |
| feature-impl | 599 lines, 8+ behavioral, 1 stance | ≤500, ≤4 behavioral, +stance | Add stance (Decision 2), strip self-review + phase gates | orch-go-w2q1e |
| architect | 673 lines, ~4 behavioral, 5 stance | ≤500, ≤4 behavioral | Trim template verbosity, extract to reference docs | orch-go-pd5rv |
| systematic-debugging | 802 lines, 5-6 behavioral, 6 stance | ≤500, ≤4 behavioral | Extract technique details to reference docs (progressive disclosure) | orch-go-quq5v |
| codebase-audit | 1,490 lines, TBD behavioral, TBD stance | ≤500, ≤4 behavioral | Major extraction — needs audit first, then restructure | orch-go-sr5ub |

**Exit criteria per skill:**
- ≤500 lines, ≤4 behavioral norms
- Stance explicitly named
- Contrastive scenario written and baselined
- No regression on knowledge transfer (existing scenarios)

**Depends on:** Phase 0 + Phase 1

### Phase 3: Research — Density vs Count (parallel)

**Goal:** Determine whether behavioral dilution is count-based, density-based, or both.

**Deliverables:**
- Experiment using experiment skill (18 mandates / 294 lines) as test case
- Compare dilution at same count but different densities
- Model update if threshold is density-dependent

**Depends on:** Phase 0 (needs stance validation data)

**Beads:** orch-go-6rb93

---

## Readiness Assessment

| Decision Point | Substrate Available | Navigable? |
|----------------|---------------------|------------|
| Self-review extraction scope | Audit data, hook infrastructure model | Yes — after Phase 0 |
| Feature-impl stance content | Contrastive scenario infrastructure | Yes — after Phase 0 |
| Density vs count threshold | Experiment skill as natural test case | Yes — after Phase 0 |

**Overall readiness:** Blocked on Phase 0 (stance validation experiment, running now). All downstream work is designed and dependency-wired. Once the gate clears, the critical path is: self-review hook → parallel simplification of all 6 skills.

---

## Structured Uncertainty

**What's tested:**
- Orchestrator simplification produces measurable improvement (108+ trials)
- Three-type taxonomy is universal across skills (audit confirms)
- Self-review is the largest behavioral block in every skill (audit confirms)
- Measurement system works for cross-source reasoning scenarios (rho=0.637)

**What's untested:**
- Whether worker stances transfer (Phase 0, running)
- Whether self-review extraction to hooks causes quality regression
- Whether feature-impl benefits from stance (it may be purely procedural by nature)
- Whether codebase-audit (1,490 lines) can be reduced to ≤500 without losing essential knowledge
- Whether density matters independently of count for dilution

**What would change this plan:**
- If worker stance doesn't transfer → re-evaluate entire program; simplification may still help via behavioral reduction but the stance preservation rationale weakens
- If self-review hook causes regression → fall back to partial extraction (Decision 1, option B)
- If codebase-audit can't fit in 500 lines → may need a "heavy skill" tier with different invariants
- If density research shows the threshold is ratio-based → update all target behavioral counts per-skill

---

## Success Criteria

- [ ] Worker stance transfer validated (Phase 0)
- [ ] Self-review hook deployed, behavioral counts drop ~50% (Phase 1)
- [ ] All 6 skills at ≤500 lines, ≤4 behavioral norms (Phase 2)
- [ ] Every skill has an explicitly named stance (Phase 2)
- [ ] Contrastive scenario exists for each skill's stance (Phase 2)
- [ ] No regression on knowledge transfer across any skill (Phase 2)
- [ ] Density vs count threshold resolved (Phase 3)
