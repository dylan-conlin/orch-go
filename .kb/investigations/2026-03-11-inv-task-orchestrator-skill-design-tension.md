## Summary (D.E.K.N.)

**Delta:** Mapped 9 design tensions in the orchestrator skill; 3 are fundamental (never fully resolvable), 4 are resolved by the Mar 4 simplification + hooks, 2 remain live.

**Evidence:** Cross-referenced 6 investigations (Jan-Mar 2026), 6 probes, the Feb 28 skill snapshot (2,368 lines), and current deployed skill (512 lines). Tensions traced to specific evidence in behavioral testing, signal ratio analysis, and constraint dilution experiments.

**Knowledge:** The fundamental tensions (knowledge-transfer vs behavioral-constraint, skill-as-grammar vs skill-as-probability-shaper, simplicity vs completeness) cannot be eliminated — they define the design space. The Mar 4 simplification resolved the accretion cycle and prompt-vs-infrastructure enforcement by moving behavioral constraints to hooks. The two live tensions (testing feasibility vs measurement need, identity compliance vs action compliance residual) need ongoing work.

**Next:** Feed into orchestrator-skill model synthesis (subproblem 3 of 3). No direct implementation needed.

**Authority:** architectural - Crosses skill design, infrastructure hooks, and testing systems.

---

# Investigation: Orchestrator Skill Design Tension Mapping

**Question:** What design tensions exist within the orchestrator skill's design, and which are resolved vs live?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** Worker agent (subproblem 2 of 3)
**Phase:** Complete
**Next Step:** None — feeds into model synthesis
**Status:** Complete
**Model:** orchestrator-skill

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-01-18-inv-update-orchestrator-skill-add-frustration.md | extends | Yes | None |
| 2026-02-24-design-orchestrator-skill-behavioral-compliance.md | foundational | Yes | None |
| 2026-03-01-design-infrastructure-systematic-orchestrator-skill.md | foundational | Yes | None |
| 2026-03-04-design-simplify-orchestrator-skill.md | foundational | Yes | None |
| 2026-03-05-inv-design-orchestrator-skill-update-incorporating.md | extends | Yes | None |
| evidence/2026-02-28-orchestrator-intent-spiral/orchestrator-skill-snapshot.md | foundational | Yes | None — represents the 2,368-line high-water mark |

---

## Findings

### Finding 1: Nine distinct design tensions identified across the investigation corpus

**Evidence:** Systematic analysis of 6 investigations and 6 related probes surfaces 9 tensions, each with evidence from multiple sources. See full tension map in SYNTHESIS.md.

**Source:** All 6 investigations, probes 2026-02-16, 2026-02-24, 2026-03-01 (constraint dilution, framework landscape, emphasis language, orientation redesign)

**Significance:** The tensions cluster into three categories: fundamental (3), resolved (4), live (2). This categorization enables the model to distinguish tensions that require ongoing management from those that have stable resolutions.

---

### Finding 2: Three tensions are fundamental — they define the design space

**Evidence:**

1. **Knowledge-transfer vs behavioral-constraint:** The Mar 1 testing infrastructure investigation proved that knowledge items (routing tables, vocabulary) provide measurable lift over bare while behavioral constraints (delegation, anti-sycophancy) show bare parity at 5+ co-resident constraints. The constraint dilution probe quantified the budget: ~4 behavioral constraints before dilution, ~50 for knowledge. This tension is fundamental because the skill must do both jobs but they have incompatible scaling properties.

2. **Skill-as-grammar vs skill-as-probability-shaper:** The Mar 1 formal grammar theory investigation proved skills provide 0% formal guarantee. They are probability shapers, not grammars. But every skill revision implicitly treats them as grammars (writing "NEVER do X" as if it were enforceable). This tension is fundamental because the mismatch between how skills work and how humans intuitively write them can never be fully eliminated.

3. **Simplicity vs completeness:** The line count trajectory (640→2,368→448→512) shows the system oscillates. New features/protocols need to be in the skill for coverage (pull toward completeness), but every addition dilutes existing constraints (pull toward simplicity). The constraint dilution work proves this is a hard tradeoff, not a solvable problem.

**Source:** Mar 1 testing infrastructure design, Mar 1 constraint dilution probe, Mar 4 simplification investigation, Feb 28 skill snapshot (2,368 lines)

**Significance:** Fundamental tensions require management strategies (budgets, infrastructure offloading, measurement), not resolution attempts. Treating them as problems to solve leads to the oscillation pattern.

---

### Finding 3: Four tensions have stable resolutions from the Mar 4 design cycle

**Evidence:**

1. **Prompt-layer vs infrastructure-layer enforcement:** Resolved by the Feb 24 investigation's two-layer recommendation, implemented as 6 infrastructure hooks (bash-write gate, git-remote gate, bd-close gate, investigation-drift nudge, spawn-ceremony nudge, spawn-context validation). The Mar 4 simplification removed ~350 lines of prohibition text that hooks now enforce. Resolution: infrastructure enforces, prompt transfers knowledge.

2. **Accretion vs simplification:** Resolved by establishing the constraint budget (≤4 behavioral norms) and the principle "knowledge framing, not prohibition framing." The 2,368→448 reduction operationalized this. Resolution: explicit budget + infrastructure offloading prevents unbounded growth.

3. **Identity compliance vs action compliance (structural):** The Feb 24 investigation diagnosed the root cause (17:1 signal disadvantage, additive vs subtractive framing). The Mar 4 hooks resolve the structural problem — action constraints are now enforced by infrastructure, not prompt instructions. The skill's action-identity fusion at top (Section 1) addresses the prompt-level component. Resolution: two-layer approach.

4. **Orchestrator-centric vs Dylan-centric organization:** The Feb 16 probe found the skill organized around "who the orchestrator IS" not "what Dylan needs." The Mar 4 redesign reoriented around three jobs (COMPREHEND, TRIAGE, SYNTHESIZE) and Dylan interface as a first-class section. Resolution: reorganized around Dylan's four moments (spawn, during, completion, session boundary).

**Source:** Feb 24 behavioral compliance investigation, Mar 4 simplification investigation, Feb 16 orientation redesign probe, current deployed skill (512 lines)

**Significance:** These resolutions are stable but fragile — they depend on maintaining the constraint budget and hook infrastructure. The simplicity vs completeness fundamental tension creates pressure to regress.

---

### Finding 4: Two tensions remain live and need ongoing work

**Evidence:**

1. **Testing feasibility vs measurement need:** The Mar 1 testing infrastructure design created a complete framework (linter, behavioral scenarios, variant comparison). But the Mar 4 investigation found `skillc test` blocked by CLAUDECODE env var in spawned sessions. The constraint dilution and emphasis probes both carry "Replication Failure Caveat" annotations (added Mar 4). The testing infrastructure exists in design but is not reliably executable from agent sessions. This blocks the measurement feedback loop that validates skill changes.

2. **Identity compliance vs action compliance (residual):** Hooks resolve the structural enforcement gap. But the Feb 24 investigation noted that some action constraints can't be hooked — e.g., choosing `orch spawn` over `bd create -l triage:ready` (both are legitimate CLI commands, just different spawn paths). The skill's prompt-level guidance for these soft preferences is still subject to the signal ratio problem. Emphasis language provides partial mitigation (Mar 2 probe: +1-2 positions on dilution curve) but is unreliable.

**Source:** Mar 1 testing infrastructure design, Mar 4 simplification (blocked section), Mar 2 emphasis language probe, Feb 24 behavioral compliance investigation

**Significance:** The testing gap means the system can't validate that skill changes improve behavior. The residual action compliance gap means soft preferences remain probabilistic. Both are workable but unsolved.

---

## Synthesis

**Key Insights:**

1. **Fundamental tensions require management, not resolution.** The three fundamental tensions (knowledge vs behavioral, grammar vs probability-shaper, simplicity vs completeness) define the possibility space. Every skill revision navigates them. The Mar 4 design cycle's best contribution was making these tensions explicit and establishing management strategies (constraint budget, infrastructure offloading, bare-parity testing).

2. **The accretion cycle was the most expensive tension.** The 640→2,368→448 trajectory consumed significant design effort across 5+ investigations. Its resolution (constraint budget + hooks) is the highest-value outcome from this investigation corpus.

3. **Measurement is the linchpin.** The testing feasibility tension gatekeeps all others — without reliable bare-parity measurement, you can't validate whether the fundamental tensions are being managed well. The Mar 1 infrastructure design is sound but the execution environment problem (CLAUDECODE env var) blocks it.

**Answer to Investigation Question:**

Nine design tensions exist. Three are fundamental (will never be fully resolved): knowledge-transfer vs behavioral-constraint, skill-as-grammar vs skill-as-probability-shaper, simplicity vs completeness. Four are resolved by the Mar 4 simplification cycle: prompt vs infrastructure enforcement, accretion vs simplification, structural identity-action gap, and orchestrator-centric vs Dylan-centric organization. Two remain live: testing feasibility (blocked by env var) and residual soft-preference compliance.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 6 investigations and 6 probes read and cross-referenced (verified: content analysis performed on full text)
- ✅ Current deployed skill (512 lines) compared against Feb 28 snapshot (2,368 lines) for resolution verification
- ✅ Hook infrastructure confirmed as operational (6/7 hooks, per Mar 4 investigation)

**What's untested:**

- ⚠️ Whether the 4 "resolved" tensions stay resolved under future accretion pressure (the simplicity-completeness fundamental tension creates regression risk)
- ⚠️ Whether the constraint dilution thresholds (≤4 behavioral, ≤50 knowledge) hold across model versions (replication failure caveat from Mar 4)
- ⚠️ Whether the current 512-line skill is at, above, or below the optimal size for the constraint budget

**What would change this:**

- If `skillc test` env var issue is resolved, the testing tension becomes resolved
- If constraint dilution thresholds replicate cleanly, the fundamental tension management strategies gain stronger evidence
- If the skill grows past 600 lines without behavioral measurement, accretion tension may reopen

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-18-inv-update-orchestrator-skill-add-frustration.md` — Frustration trigger addition
- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` — Identity vs action compliance root cause
- `.kb/investigations/2026-03-01-design-infrastructure-systematic-orchestrator-skill.md` — Testing infrastructure design
- `.kb/investigations/2026-03-04-design-simplify-orchestrator-skill.md` — 2,368→448 simplification
- `.kb/investigations/2026-03-05-inv-design-orchestrator-skill-update-incorporating.md` — 72-commit delta update
- `.kb/investigations/evidence/2026-02-28-orchestrator-intent-spiral/orchestrator-skill-snapshot.md` — 2,368-line high-water mark
- `skills/src/meta/orchestrator/SKILL.md` — Current deployed skill (512 lines)
- 6 probes in `.kb/models/orchestrator-session-lifecycle/probes/` — Constraint dilution, emphasis language, orientation redesign, framework landscape, behavioral compliance, CLI staleness

**Related Artifacts:**
- **Model (target):** `.kb/models/orchestrator-skill/` — This investigation feeds model synthesis
- **Model (parent):** `.kb/models/orchestrator-session-lifecycle/` — Contains most related probes

---

## Investigation History

**2026-03-11 17:25:** Investigation started
- Initial question: What design tensions exist in the orchestrator skill's design?
- Context: Subproblem 2 of 3 for orchestrator-skill model synthesis

**2026-03-11 17:45:** Analysis complete
- Read all 6 investigations, 6 probes, both skill versions
- Identified 9 tensions, categorized as fundamental (3), resolved (4), live (2)

**2026-03-11 17:50:** Investigation completed
- Status: Complete
- Key outcome: 9 tensions mapped with evidence chains, resolution status, and categorization
