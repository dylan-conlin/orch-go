# Session Synthesis

**Agent:** og-inv-task-evidence-inventory-12mar-028b
**Issue:** --no-track
**Duration:** 2026-03-12 21:11 → 2026-03-12 21:30
**Outcome:** success

---

## Plain-Language Summary

This session read all 7 investigations and 4 probes in the orchestrator skill investigation cluster (Feb 24 – Mar 11, 2026) and extracted every empirical finding into a numbered inventory of 67 claims. Each claim is classified by evidence type (MEASURED/ANALYTICAL/ASSUMED), source, and replication status. The key finding: 14 claims are high-confidence (verified across multiple investigations), but the most-cited quantitative thresholds (behavioral constraint budget ≤4, dilution starts at 5 constraints) carry an explicit replication failure caveat from March 4 — they're directional hypotheses, not established facts. The three strongest findings in the cluster are: (1) knowledge transfers stick while behavioral constraints don't, (2) bare-parity testing works as a behavioral validation method, and (3) the 82% skill size reduction preserved knowledge-transfer value.

## Verification Contract

See probe file: `.kb/models/orchestrator-session-lifecycle/probes/2026-03-12-probe-evidence-inventory-orchestrator-skill-cluster.md`
No executable tests — this is a pure analysis/inventory task.

---

## TLDR

67 empirical claims extracted from 11 sources. 39 measured, 26 analytical, 2 assumed. 14 high-confidence (multi-source). The dilution curve's replication failure caveats the most specific quantitative claims.

---

## Evidence Inventory: All Claims by Theme

### Legend

- **MEASURED** = test scores, line counts, word counts, command output, concrete numbers
- **ANALYTICAL** = logical deduction from measured evidence
- **ASSUMED** = stated without direct supporting evidence
- **⭐ Multi-source** = appeared in 2+ investigations/probes (high confidence)
- **⚠️ Caveated** = carries replication failure caveat or significant uncertainty

---

### THEME 1: BEHAVIORAL MEASUREMENT (Claims 1–24)

#### Bare-Parity Testing Results

**1.** v2.1 skill (259 lines, 3217 tokens) scores 22/56 (39%) on 7 behavioral scenarios.
- Evidence: MEASURED — skillc test, Opus 4.6, single run
- Source: Inv-Mar01-Baseline, Finding 1
- Replication: Single-source

**2.** Pre-v2 skill (7686b131, 457 lines, ~5800 tokens) scores 21/56 (38%) on same scenarios.
- Evidence: MEASURED — skillc test, Opus 4.6, single run
- Source: Inv-Mar01-Baseline, Finding 1
- Replication: Single-source

**3.** Bare baseline (no skill) scores 17/56 (30%) on same scenarios.
- Evidence: MEASURED — skillc test, Opus 4.6, single run
- Source: Inv-Mar01-Baseline, Finding 1
- Replication: Confirmed by Haiku cross-check (same source)

**4. ⭐** 5 of 7 scenarios are bare-parity violations — skill adds zero measurable value on those behaviors.
- Evidence: MEASURED — both skill variants match bare scores on 5 scenarios
- Source: Inv-Mar01-Baseline, Findings 1-2; Inv-Mar01-Infra, Synthesis
- Replication: Multi-source (2 investigations)

**5. ⭐** Knowledge-transfer scenarios beat bare; behavioral-constraint scenarios do not.
- Evidence: MEASURED — routing table +4/8, framing vocabulary +2/8, intent distinction +2/8 over bare; delegation speed 1/8=bare, reconnection 0-1/8=bare, anti-sycophancy 3/8=bare
- Source: Inv-Mar01-Baseline, Finding 2; Inv-Mar04-Simplify, Evidence Base; Inv-Mar04-Grammar, What the U-curve teaches; Probe-Feb24-Compliance, Finding 3
- Replication: **Multi-source (4 sources)** — highest-confidence finding in cluster

**6.** v2.1 wins complex-architectural-routing (7/8 vs 5/8); 7686b131 wins intent-clarification (5/8 vs 3/8).
- Evidence: MEASURED — per-scenario scores
- Source: Inv-Mar01-Baseline, Finding 3
- Replication: Single-source; noted as within noise for single run

**7.** Haiku cross-check shows same bare-parity pattern (0/7 passed for bare, 1/7 for v2.1).
- Evidence: MEASURED — skillc test with haiku model
- Source: Inv-Mar01-Baseline, Haiku cross-check
- Replication: Cross-model confirmation (same investigation)

**8. ⭐** Anti-sycophancy constraint shows zero signal across all variants and experiments.
- Evidence: MEASURED — 3/8 for bare, v2.1, and 7686b131; zero change across 5C, 10C, v4
- Source: Inv-Mar01-Baseline, Finding 2; Inv-Mar04-Grammar, Why anti-sycophancy doesn't get a slot
- Replication: Multi-source (2 investigations)

**9.** skillc test isolation fix works: CLAUDECODE env var stripping + clean CWD enables testing from within Claude Code.
- Evidence: MEASURED — successfully ran tests from within a Claude Code session
- Source: Inv-Mar01-Baseline, Finding 4
- Replication: Single-source; later contradicted — see claim #35

#### Constraint Dilution Curve

**10. ⚠️** 1-constraint (delegation): [8,8,8] median 8/8, proposes-delegation 3/3.
- Evidence: MEASURED — skillc test, sonnet, 3 runs
- Source: Probe-Mar01-Dilution
- Replication: Confirmed as control (matches aj58 isolation result). **REPLICATION FAILURE CAVEAT**

**11. ⚠️** 2-constraint: [8,8,8] median 8/8, proposes-delegation 3/3.
- Evidence: MEASURED — skillc test, sonnet, 3 runs
- Source: Probe-Mar01-Dilution
- Replication: Single-source. **REPLICATION FAILURE CAVEAT**

**12. ⚠️** 5-constraint: [3,8,8] median 8/8, proposes-delegation 2/3.
- Evidence: MEASURED — skillc test, sonnet, 3 runs
- Source: Probe-Mar01-Dilution
- Replication: Single-source. **REPLICATION FAILURE CAVEAT**

**13. ⚠️** 10-constraint: [5,5,5] median 5/8, proposes-delegation 0/3 — identical to bare.
- Evidence: MEASURED — skillc test, sonnet, 3 runs
- Source: Probe-Mar01-Dilution
- Replication: Single-source. **REPLICATION FAILURE CAVEAT**

**14. ⚠️** Knowledge constraints survive at 10C (intent probe: 6/8 median, 3/3 asks-clarification).
- Evidence: MEASURED — skillc test, sonnet, 3 runs
- Source: Probe-Mar01-Dilution
- Replication: Single-source. **REPLICATION FAILURE CAVEAT**

**15. ⚠️⭐** Behavioral constraint budget: ~2-4 co-resident constraints before dilution.
- Evidence: MEASURED (derived from dilution curve) — but carries replication failure caveat
- Source: Probe-Mar01-Dilution; Inv-Mar04-Grammar, Budget; Inv-Mar04-Simplify, ≤4 behavioral norms
- Replication: Multi-source (3), but all derive from same underlying experiment. **REPLICATION FAILURE CAVEAT**

**16. ⚠️** Knowledge constraint budget: ~50 (functional at 10+).
- Evidence: MEASURED (derived) — knowledge constraints still functional at 10C
- Source: Probe-Mar01-Dilution; Inv-Mar04-Grammar, Budget
- Replication: Multi-source (2). **REPLICATION FAILURE CAVEAT**

**17. ⚠️** The dilution curve (3/3→3/3→2/3→0/3) did NOT replicate under clean isolation (orch-go-zola, Mar 4).
- Evidence: MEASURED — replication attempt failed
- Source: Probe-Mar01-Dilution, Replication Failure Caveat
- Replication: **Failed replication** — this is itself a measured result

#### Emphasis Language Experiment

**18. ⚠️** 10C-emphasis proposes-delegation: 2/3; 10C-neutral: 0/3.
- Evidence: MEASURED — skillc test, sonnet, 3 runs per variant, 30 total runs
- Source: Probe-Mar02-Emphasis, Finding 1
- Replication: Single-source. Inherits dilution curve caveat.

**19. ⚠️** 5C-emphasis proposes-delegation: 2/3; 5C-neutral: 1/3.
- Evidence: MEASURED — skillc test, sonnet, 3 runs per variant
- Source: Probe-Mar02-Emphasis, Finding 1
- Replication: Single-source. Inherits dilution curve caveat.

**20. ⚠️** 10C-neutral delegation scores [5,5,5] with 0/3 proposes-delegation — exactly matching bare.
- Evidence: MEASURED — identical to bare [5,5,5], 0/3
- Source: Probe-Mar02-Emphasis, Finding 2
- Replication: Single-source

**21. ⚠️** Emphasis effect appears larger at higher constraint counts (+1 at 5C, +2 at 10C).
- Evidence: MEASURED (directional) — but N=3 per variant
- Source: Probe-Mar02-Emphasis, Finding 3
- Replication: Single-source

**22. ⚠️** Bare baseline shifted between sessions on intent probe (3/8→6/8 median).
- Evidence: MEASURED — cross-session comparison
- Source: Probe-Mar02-Emphasis, Finding 4
- Replication: Single-source; demonstrates cross-session variance

**23. ⚠️** Combined 10C emphasis across 2 sessions: proposes-delegation 2/6 (33%).
- Evidence: MEASURED — pooled from 2 experiments
- Source: Probe-Mar02-Emphasis, Combined Evidence table
- Replication: Cross-session pooling (partial replication)

**24.** All emphasis experiments used single-turn --print mode; interactive sessions untested.
- Evidence: ANALYTICAL — noted as limitation
- Source: Inv-Mar01-Baseline, Structured Uncertainty; Probe-Mar02-Emphasis, Structured Uncertainty
- Replication: Multi-source (consistent caveat)

#### U-Curve / Fabrication Experiment

**25.** 5C variant: 20% correct on urgency vs bare 100% — catastrophic regression.
- Evidence: MEASURED — from fabrication experiment (orch-go-wou2)
- Source: Inv-Mar04-Grammar, The Problem
- Replication: Single-source (referenced, not re-run in this cluster)

**26.** At 5C, 80% of agents proposed triage for urgent production issue (over-application).
- Evidence: MEASURED — fabrication experiment
- Source: Inv-Mar04-Grammar, Slot 2 (Undefined behavior handler)
- Replication: Single-source

**27.** 10C variant: 30% on urgency (slight recovery due to mutual dilution reducing gravitational pull).
- Evidence: MEASURED — fabrication experiment
- Source: Inv-Mar04-Grammar, What the U-curve teaches
- Replication: Single-source

**28.** Current v4 (~47 knowledge items + 4 norms) scores 70% on urgency.
- Evidence: MEASURED — fabrication experiment
- Source: Inv-Mar04-Grammar, What the U-curve teaches
- Replication: Single-source

---

### THEME 2: ARCHITECTURE — KNOWLEDGE vs CONSTRAINTS (Claims 29–42)

**29. ⭐** Identity compliance and action compliance are mechanistically different: identity is additive (no conflict), action is subtractive (conflicts with system prompt).
- Evidence: ANALYTICAL — structural analysis of skill + system prompt
- Source: Inv-Feb24-Compliance, Finding 1; Probe-Feb24-Compliance, Finding 3; Inv-Mar01-Baseline, Finding 2
- Replication: **Multi-source (3)** — high confidence

**30. ⭐** Signal ratio: ~17:1 in favor of Task tool usage (system prompt ~500 words promoting, skill ~30 words constraining).
- Evidence: MEASURED — word count comparison
- Source: Inv-Feb24-Compliance, Finding 2; Probe-Feb24-Compliance, Test 2; Inv-Mar11-Tensions, Finding 3
- Replication: **Multi-source (3)** — high confidence

**31.** Action constraints first substantive appearance at line 68 (10% depth); full detail at line 594 (88% depth) in 640-line skill.
- Evidence: MEASURED — line number analysis
- Source: Inv-Feb24-Compliance, Finding 2; Probe-Feb24-Compliance, Test 1
- Replication: Multi-source (2), but same skill version

**32. ⭐** Tool Action Space table is a markdown description, not infrastructure enforcement — all tools remain available.
- Evidence: MEASURED — verified Task tool, Edit, Write, all Bash commands available in orchestrator sessions
- Source: Inv-Feb24-Compliance, Finding 3; Probe-Feb24-Compliance, Finding 1
- Replication: Multi-source (2)

**33.** System prompt occupies structurally superior instruction hierarchy position (system > user > assistant); skill injected as user-level content.
- Evidence: ANALYTICAL — Claude architecture analysis
- Source: Inv-Feb24-Compliance, Finding 2; Probe-Feb24-Compliance, Finding 2
- Replication: Multi-source (2)

**34. ⭐** Two-layer fix needed: (1) restructure skill for prompt-level salience, (2) add infrastructure enforcement.
- Evidence: ANALYTICAL — derived from findings 29-33
- Source: Inv-Feb24-Compliance, Synthesis; Inv-Mar01-Baseline, Synthesis; Inv-Mar01-Infra, Synthesis; Inv-Mar04-Simplify, Design; Inv-Mar11-Tensions, Finding 3
- Replication: **Multi-source (5)** — highest-replication analytical finding

**35.** skillc test blocked by CLAUDECODE env var when run from spawned agent sessions (Mar 4).
- Evidence: MEASURED — returns 0/0 scores from spawned agents
- Source: Inv-Mar04-Simplify, What I tried #3
- Replication: Contradicts claim #9 (isolation fix worked in Mar 1). Mar 4 finding is later evidence.

**36.** Temporal dynamics: system prompt instructions persistent (every turn), skill content injected once and decays.
- Evidence: ANALYTICAL — instruction processing timing analysis
- Source: Inv-Feb24-Compliance, Finding 5
- Replication: Single-source

**37.** `bd close` vs `orch complete` has same competing signal problem as Task tool vs `orch spawn`.
- Evidence: ANALYTICAL — same structural pattern
- Source: Inv-Feb24-Compliance, Finding 4
- Replication: Single-source

**38. ⭐** Skills are probability-shaping documents, not grammars — 0% formal guarantee.
- Evidence: ANALYTICAL — formal grammar theory analysis
- Source: Inv-Mar01-Infra, Prior Work (formal-grammar-theory); Inv-Mar11-Tensions, Finding 2
- Replication: Multi-source (2)

**39.** All agent frameworks enforce at action boundary, not decision boundary.
- Evidence: ANALYTICAL — landscape analysis of CrewAI, AutoGen, LangGraph, etc.
- Source: Inv-Mar01-Infra, Prior Work (agent-framework-constraints)
- Replication: Single-source (references external Probe-Mar01-agent-framework-landscape)

**40.** Research consensus: prompts describe desired behavior; infrastructure enforces it.
- Evidence: ANALYTICAL — literature review (ICLR 2025, AgentSpec ICSE 2026, PCAS)
- Source: Inv-Feb24-Compliance, Finding 6; Probe-Feb24-Compliance, Finding 4
- Replication: Multi-source (2); backed by external academic references

**41.** "Answer the question asked" norm is knowledge wearing behavioral clothing — can be reformulated as knowledge.
- Evidence: ANALYTICAL — classification of constraint type
- Source: Inv-Mar04-Grammar, Why "answer the question asked" becomes knowledge
- Replication: Single-source

**42.** Matched-pair principle: each behavioral constraint needs specific calibration knowledge alongside it to prevent U-curve regression.
- Evidence: ANALYTICAL — derived from U-curve experiment results (claims 25-28)
- Source: Inv-Mar04-Grammar, The matched-pair principle
- Replication: Single-source

---

### THEME 3: EVOLUTION / ACCRETION (Claims 43–55)

**43. ⭐** Skill line count trajectory: 640 → 2,368 → 448 → 512.
- Evidence: MEASURED — line counts from skill files at different points
- Source: Inv-Mar04-Simplify, TLDR; Inv-Mar11-Tensions, Finding 2
- Replication: Multi-source (2)

**44.** v4 template: 448 lines, 4,830 tokens (deployed Mar 4).
- Evidence: MEASURED — actual file measurement
- Source: Inv-Mar04-Simplify, What I observed
- Replication: Single-source

**45.** Token reduction: 27,200 → 4,830 (82% reduction).
- Evidence: MEASURED — token count comparison
- Source: Inv-Mar04-Simplify, What I observed
- Replication: Single-source

**46.** 6 of 7 infrastructure hooks active and verified with specific test counts.
- Evidence: MEASURED — bash-write (308 tests), git-remote (64), bd-close (37), investigation-drift (38), spawn-ceremony (40), spawn-context (68)
- Source: Inv-Mar04-Simplify, Hook Infrastructure table
- Replication: Single-source

**47.** Code-access gate hook NOT REGISTERED (7th hook).
- Evidence: MEASURED — hook status check
- Source: Inv-Mar04-Simplify, Hook Infrastructure table
- Replication: Single-source

**48.** ~47 knowledge items preserved after simplification.
- Evidence: MEASURED — counted from v4 template
- Source: Inv-Mar04-Simplify, What stays section
- Replication: Single-source

**49.** 4 behavioral norms in v4: delegation, filter-before-presenting, act-by-default, answer-the-question-asked.
- Evidence: MEASURED — enumerated from deployed skill
- Source: Inv-Mar04-Simplify, ≤4 Behavioral Norms
- Replication: Single-source

**50.** v4 recommended slot reallocation: drop "answer the question asked," add "undefined behavior handler" + "pressure over compensation."
- Evidence: ANALYTICAL — design recommendation from evidence
- Source: Inv-Mar04-Grammar, The 4 Slots
- Replication: Single-source

**51.** `orch frontier` replaced by `orch status` — appears 4 times in skill, all stale.
- Evidence: MEASURED — `orch frontier` returns "unknown command"
- Source: Inv-Mar05-Update, Finding 2
- Replication: Single-source

**52.** Review tier system live: 4 tiers (auto, scan, review, deep) mapped to skills.
- Evidence: MEASURED — verified in review_tier.go source code
- Source: Inv-Mar05-Update, Finding 3
- Replication: Single-source

**53.** `--no-track` now creates real beads issue with `tier:lightweight` label.
- Evidence: MEASURED — confirmed in main_test.go:818
- Source: Inv-Mar05-Update, Finding 5
- Replication: Single-source

**54.** Daemon has concurrency cap 5, round-robin project fairness, auto-complete for auto-tier agents.
- Evidence: MEASURED — verified in source code (invariants_test.go, issue_queue.go, daemon.go)
- Source: Inv-Mar05-Update, Finding 4
- Replication: Single-source

**55.** 4 of 10 infrastructure changes needed no skill edits (already present or worker-only).
- Evidence: MEASURED — audited each change against skill content
- Source: Inv-Mar05-Update, Finding 1
- Replication: Single-source

---

### THEME 4: DESIGN TENSIONS & FAILURE MODES (Claims 56–67)

**56. ⭐** 9 design tensions identified in the orchestrator skill: 3 fundamental, 4 resolved, 2 live.
- Evidence: ANALYTICAL — cross-reference of 6 investigations and 6 probes
- Source: Inv-Mar11-Tensions, Finding 1
- Replication: Synthesized from multi-source (6 investigations)

**57.** Three fundamental tensions (never fully resolvable): knowledge-transfer vs behavioral-constraint, skill-as-grammar vs skill-as-probability-shaper, simplicity vs completeness.
- Evidence: ANALYTICAL — derived from measurement findings
- Source: Inv-Mar11-Tensions, Finding 2
- Replication: Each tension documented in 2+ investigations

**58.** Four tensions resolved by Mar 4 design cycle: prompt-vs-infrastructure enforcement, accretion-vs-simplification, structural identity-action gap, orchestrator-centric-vs-Dylan-centric organization.
- Evidence: ANALYTICAL — resolution verified against deployed artifacts
- Source: Inv-Mar11-Tensions, Finding 3
- Replication: Cross-referenced against source investigations

**59.** Two live tensions: testing feasibility (blocked by env var) and residual soft-preference compliance.
- Evidence: ANALYTICAL — status verified
- Source: Inv-Mar11-Tensions, Finding 4
- Replication: Testing feasibility confirmed by claim #35

**60. ⭐** 12 failure modes identified (expanded from model's original 5), clustered into 4 layers.
- Evidence: ANALYTICAL — cross-reference taxonomy
- Source: Probe-Mar11-FailureModes, Finding 1
- Replication: Synthesized from 6 investigations + 7 probes

**61.** 3 failure modes drove Jan→Mar 2026 skill evolution: dilution, competing hierarchy, intent displacement.
- Evidence: ANALYTICAL — causal analysis
- Source: Probe-Mar11-FailureModes, Finding 2
- Replication: Each mode documented in 2+ investigations

**62.** 6 of 12 failure modes resolved or mitigated (3 by hooks, 2 by simplification, 1 by template fix).
- Evidence: ANALYTICAL — resolution status check
- Source: Probe-Mar11-FailureModes, Finding 4
- Replication: Hook effectiveness verified by claim #46

**63.** MUST fatigue is a real anti-pattern: countable statically (>3 per 100 words = warning threshold).
- Evidence: ANALYTICAL — from DSL design principles investigation
- Source: Inv-Mar01-Infra, Fork 5, Lint Rules
- Replication: Single-source (threshold is proposed, not experimentally validated)

**64.** Cosmetic redundancy countable: same constraint phrase >2 times = no additional enforcement value.
- Evidence: ANALYTICAL — from defense-in-depth investigation
- Source: Inv-Mar01-Infra, Fork 5, Lint Rules
- Replication: Single-source

**65.** LLM-as-judge rejected: AI evaluating AI is a closed loop per provenance principle.
- Evidence: ASSUMED — stated as principle, not experimentally tested
- Source: Inv-Mar01-Infra, Fork 1
- Replication: Single-source

**66.** Estimated 15-25 constraint ceiling (aj58 investigation) is too high; actual behavioral budget ~2-4.
- Evidence: MEASURED (⚠️ CAVEATED) — dilution curve contradicted higher estimate, but carries replication failure
- Source: Probe-Mar01-Dilution, Model Impact
- Replication: Single-source; original aj58 estimate not re-verified

**67.** Orchestrator skill had 14 commits in 2 weeks (Feb 14 – Mar 1) — high churn rate.
- Evidence: MEASURED — git log count
- Source: Inv-Mar01-Baseline, Related section
- Replication: Single-source

---

## Cross-Reference Matrix: High-Confidence Claims (3+ sources)

| # | Claim | Sources | Type |
|---|-------|---------|------|
| 5 | Knowledge sticks, constraints don't | Inv-Feb24, Inv-Mar01-Baseline, Inv-Mar04-Simplify, Inv-Mar04-Grammar, Probe-Feb24 | MEASURED |
| 34 | Two-layer enforcement needed | Inv-Feb24, Inv-Mar01-Baseline, Inv-Mar01-Infra, Inv-Mar04-Simplify, Inv-Mar11 | ANALYTICAL |
| 29 | Identity ≠ action compliance | Inv-Feb24, Probe-Feb24, Inv-Mar01-Baseline | ANALYTICAL |
| 30 | 17:1 signal ratio | Inv-Feb24, Probe-Feb24, Inv-Mar11 | MEASURED |
| 15 | Behavioral budget ≤4 | Probe-Mar01-Dilution, Inv-Mar04-Grammar, Inv-Mar04-Simplify | MEASURED ⚠️ |

---

## Critical Caveats Summary

**1. Replication Failure (highest priority):** The dilution curve (claims 10-17, 66) did not replicate under clean isolation (orch-go-zola, Mar 4). All specific threshold numbers (behavioral budget ~2-4, degradation starts at 5, bare parity at 10) should be treated as directional hypotheses. This affects 8 claims directly and the emphasis experiment (claims 18-23) indirectly.

**2. Single-Turn Testing:** All behavioral measurements used `claude --print` mode (single response, no tools, no persistence). Real orchestrator sessions have multi-turn context, tools, and hooks. Behavioral compliance may differ significantly in interactive sessions. (Claims 1-14, 18-28.)

**3. Sample Size:** Most experiments used N=3 runs per variant. The emphasis probe notes the opus "confirmation" of the dilution curve was "noise matching noise at N=3." Statistical significance is not established for any individual claim.

**4. Model Version Sensitivity:** All experiments ran on specific model versions (sonnet, opus) at specific dates. Run-to-run variance in bare baselines was documented (claim 22: intent shifted 3/8→6/8 between sessions).

---

## Delta (What Changed)

### Files Created
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-12-probe-evidence-inventory-orchestrator-skill-cluster.md` - Probe file with model impact
- `.orch/workspace/og-inv-task-evidence-inventory-12mar-028b/SYNTHESIS.md` - This file

### Files Modified
- None

---

## Architectural Choices

No architectural choices — task was pure analysis/inventory.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- The replication failure caveat on the dilution curve is underweighted in the model — it affects the single most-cited quantitative finding across the cluster
- The cluster has a circular reference problem: Inv-Mar04-Grammar, Inv-Mar04-Simplify, and Inv-Mar11-Tensions all cite the dilution curve thresholds as established facts, but those thresholds are unvalidated

### Decisions Made
- Classified 67 claims (vs attempting to filter to "important" ones) — completeness is more valuable than curation for an evidence inventory
- Used 3 evidence types (MEASURED/ANALYTICAL/ASSUMED) rather than a finer-grained scale — simplicity over precision

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (numbered inventory, probe file, SYNTHESIS.md)
- [x] Probe file created with model impact
- [x] Evidence inventory grouped by theme with cross-references

### Recommended Follow-up
1. **Re-run dilution experiments** with clean isolation to validate or invalidate threshold numbers (claims 10-17)
2. **Test in interactive sessions** — all behavioral measurements are from single-turn --print mode; the transfer to real orchestrator sessions is assumed, not verified
3. **Merge probe findings into model.md** — the evidence quality stratification (multi-source vs single-source, caveated vs firm) should be reflected in how the model presents its claims

---

## Unexplored Questions

- **Does the knowledge-vs-constraint divergence hold for other skills** (worker skills, architect skills), or is it specific to the orchestrator skill's relationship with the Claude Code system prompt?
- **What is the actual compliance rate in real interactive orchestrator sessions?** All measurements are from --print mode. The gap between lab measurement and production behavior is unknown.
- **Do the emphasis effects hold for opus?** Only tested on sonnet. Opus may have different attention allocation to emphasis markers.

---

## Friction

No friction — reading and analysis task, tools worked smoothly.

---

## Session Metadata

**Skill:** investigation
**Model:** opus 4.6
**Workspace:** `.orch/workspace/og-inv-task-evidence-inventory-12mar-028b/`
**Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-03-12-probe-evidence-inventory-orchestrator-skill-cluster.md`
