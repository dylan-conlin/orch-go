# Model: Skill Content Transfer

**Domain:** How Claude processes skill content — applies to ALL skills, not just orchestrator
**Last Updated:** 2026-03-06
**Synthesized From:** 7 investigations (Jan 18 - Mar 6, 2026), 144 trials across 3 variants + worker skill industry practice audit + investigation stance contrastive experiment

---

## Summary (30 seconds)

Skills contain three types of content — knowledge, behavioral constraints, and stance — that transfer through fundamentally different mechanisms. Knowledge (routing tables, templates, vocabulary) produces measurable lift (+5 points). Stance (epistemic orientation) produces the largest discrimination on hard scenarios (0% → 83% on implicit contradictions at N=6) — but only when the stance is an **attention primer** (changes what agents notice), not an **action directive** (tells agents what to do). The orchestrator's "look for implicit assumptions" works; the investigation's "test before concluding" does not (0 lift, 54 trials). Behavioral constraints (NEVER/MUST prohibitions) dilute at 5+ co-resident items and become inert at 10+. The design playbook: strip behavioral weight to hooks, keep knowledge and attention-priming stance in the skill document.

---

## Core Mechanism

### Three Content Types (Universal Taxonomy)

Every skill contains three types of content that transfer through fundamentally different mechanisms. This was discovered on the orchestrator skill and confirmed as universal across worker skills.

| Content Type | What It Does | Transfer Mechanism | Measurability |
|---|---|---|---|
| **Knowledge** | Tells the agent what exists (information it wouldn't otherwise have) | Direct — agent reads and applies | Single-turn: does agent use the right vocabulary/format? |
| **Stance** | Orients how the agent approaches problems (epistemic posture) | Indirect — shifts attention and framing (attention primers transfer; action directives do not) | Contrastive scenarios: does orientation change on ambiguous inputs? |
| **Behavioral** | Tells the agent what to do/not do (prohibitions, mandates) | Unreliable — dilutes at 5+, inert at 10+ | Compliance rate under competing system-prompt signals |

### Evidence

**Knowledge transfer confirmed (Mar 1, N=7 scenarios):**
- Bare Claude: 17/56. With knowledge items: 22/56. Lift: +5 points.
- Concentrated in: routing tables (+4/8), framing vocabulary (+2/8), intent clarification (+2/8).

**Stance transfer confirmed (Mar 5-6, N=6, 72 trials across 5 scenarios):**
- Scenario 09 (implicit contradiction): bare 0/6, without-stance 1/6, with-stance 5/6.
- 0% → 17% → 83%. Stance items don't teach agents what contradictions look like — they orient agents to *look for meaning*.
- Key design insight: discriminating scenarios require implicit signals (incompatible assumptions, not opposite conclusions). Explicit signals (data tables, opposite findings) hit ceiling on Sonnet.

**Stance generalization confirmed (Mar 6, N=6, 36 trials across 3 new scenarios):**
- Stance is a **cross-source reasoning primer**, not a generic attention amplifier.
- Lifts performance when defects hide in gaps BETWEEN information sources:
  - S12 (relationship tracing: query change → dashboard consumer): bare median 1 → stance median 6 (+5)
  - S13 (information freshness: git log contradicts deprecation comment): bare median 0 → stance median 4 (+4)
- No lift when defects are visible within a single source:
  - S11 (absence detection: auth middleware gap visible in code): bare median 5 → stance median 3 (-2)
- Per-indicator: `connects-git-evidence` bare 1/6 → stance 6/6. `notices-consumer-impact` bare 3/6 → stance 6/6.
- **Detection-to-action gap:** All action indicators (`recommends-fix`, `no-premature-completion`, `no-blind-removal`) at floor (0/6) for BOTH variants. Stance improves what agents notice but not what they do about it. Action requires behavioral constraints alongside stance.

**Investigation stance does NOT transfer (Mar 6, N=6, 54 trials across 3 scenarios):**
- Investigation stance items ("test before concluding", "artifacts are claims not evidence") show zero lift: bare median 4, stance median 3-4 across all scenarios.
- Per-indicator: I01 `questions-prior-finding` bare 0/6 → stance 2/6 (directional but not significant). I02 `identifies-middleware-gap` bare 6/6 → stance 3/6 (stance HURT performance).
- **Root cause:** Investigation stance is an action directive ("test, don't reason"), not an attention primer ("look for gaps between sources"). Action directives have no leverage in `--print` mode where agents cannot execute tests. Attention primers work because they change what agents notice, which IS visible in text output.

### Measurement Calibration (Mar 6, N=24, human blind-rated)

Human calibration experiment: 24 responses across 4 scenarios × 3 variants, blind-rated 1-5 by Dylan, compared against auto-scorer indicator counts.

**Overall:** Spearman rho = 0.637 (p=0.0001). Passes r>0.6 gate — automated proxies correlate with human judgment of comprehension quality.

**Per-scenario:**

| Scenario | rho | N | Interpretation |
|----------|-----|---|----------------|
| S09 (implicit contradiction) | 0.980 | 6 | Near-perfect — indicators track human judgment precisely |
| S13 (information freshness) | 0.894 | 6 | Strong — git-evidence and freshness indicators validated |
| S12 (relationship tracing) | 0.747 | 6 | Good — consumer-impact indicators work |
| S11 (absence detection) | 0.141 | 6 | Broken — indicators uncorrelated with human judgment, need vocabulary redesign |

**Vocabulary bias:** Auto-scorer has systematic bias toward skill-enhanced responses. 3 of 4 biggest human-vs-auto disagreements occurred on bare variants (auto-scorer under-rates bare responses that humans find adequate). The indicator vocabulary was designed from skill-enhanced outputs and doesn't capture the reasoning patterns bare Claude uses.

**Variant means (all scenarios):** bare human=2.0, without-stance=4.2, with-stance=4.1. Stance lift washes out when averaged across scenario types — confirms stance is scenario-specific (cross-source reasoning), not a universal quality amplifier.

**Behavioral constraint dilution confirmed (Mar 1, N=7 scenarios):**
- Delegation prohibition: 1/8 (= bare). Anti-sycophancy: 3/8 (= bare). Reconnection framing: 0-1/8 (= bare).
- v3 skill had 50+ constraints — scored at bare parity on 5 of 7 scenarios.

### Three Content Types Across Skills

| Skill | Lines | Knowledge | Stance | Behavioral | B Count |
|---|---|---|---|---|---|
| **Orchestrator** | 422 | Routing tables, command reference, skill selection tree | "Synthesis is comprehension, not reporting" | ≤4 norms (delegation, filter, act-by-default, answer-asked) | ~4 ✅ |
| **Feature-impl** | 599 | Config routing, deliverables table, phase workflows, templates | Harm Assessment only (1 line) | Scope enum, TDD iron law, visual verification, self-review (15 items), leave-it-better, completion gates | 8+ ❌ |
| **Investigation** | 266 | D.E.K.N. template, prior-work table, evidence hierarchy, routing | "Answer by testing not reasoning", "Artifacts are claims", "Test before concluding" (4 lines) | Self-review (11 items), prior-work gate, leave-it-better, checkpoint mandate | 6+ ❌ |
| **Systematic-debugging** | 802 | Four-phase procedures, pattern table, techniques, layer bias, visual tools | "Understand before fixing", "Symptom ≠ root cause", scientific method, rationalizations table (6 lines) | Iron Law gate, failing test mandate, 3-fix escalation, fix-verify coupling, STOP triggers | 5-6 ❌ |
| **Experiment** | 294 | Variant structure, YAML format, skillc commands, analysis tables, failure modes | "Science not exploration", "Hypothesis before tools", "Inspect when surprised" (4 lines) | 11-item Boundaries DO/DO NOT, bare baseline mandate, commit-immediately, no-modification mandates | 8+ ❌ |
| **Architect** | 673 | Fork format, decision navigation, verification spec template, spawn threshold | "Premise before solution", "Evolve by distinction", "Coherence over patches" (5 lines) | Question cap (3-7), self-review, completion gates | ~4 ✅ |
| **Worker-base** | — | Phase reporting format, bd comment syntax, authority table | (none — foundation layer) | Hard limits, completion protocol, discovered work mandate | — |

**Implication:** Worker skills have the same structural problem the orchestrator had — behavioral weight crowding out knowledge and stance. Section-by-section audit (Mar 6) confirms: all 4 audited skills exceed ≤4 behavioral norms. Self-review sections are the single largest behavioral block (~10-15 checklist items each). Feature-impl is the most stance-poor skill (1 line). Experiment has the worst behavioral density (18 MUST/NEVER in 294 lines). Only architect is near-compliant (~4 mandates). The 82% reduction that worked for the orchestrator (2,368 → 422 lines) is the playbook for all skills.

### Implementation: Two Delivery Channels

The three content types map to two delivery channels:

1. **Skill document** — Knowledge + Stance (what transfers reliably)
2. **Hook infrastructure** — Behavioral enforcement (what requires deterministic gates)

Behavioral content in skill documents is probabilistic suggestion. Only hooks provide deterministic enforcement. Dual authority (both hook AND skill text covering same behavior) creates ambiguity and degrades trust in the skill overall.

### Critical Invariants

1. **Skill length ≤ 500 lines / 5,000 tokens** — Beyond this, constraint dilution makes additional content inert
2. **≤ 4 behavioral norms** — Research shows dilution begins at 5 co-resident constraints
3. **Knowledge framing, not prohibition** — "Here's how routing works" beats "NEVER route incorrectly"
4. **Hook-enforced behaviors must NOT appear in skill text** — Dual authority creates confusion about what's enforced vs advisory
5. **Stance transfers only as an attention primer, not an action directive** — Stance items that change HOW agents perceive information (attention primers like "look for implicit assumptions between sources") produce +4 to +7 lift on cross-source reasoning scenarios. Stance items that tell agents WHAT TO DO (action directives like "test before concluding") produce zero lift (54 trials, 3 scenarios, 0 median change). Action directives require tool execution for leverage; in text/--print mode they have none. Single-source defects don't benefit from either type. Stance improves detection but not action — behavioral constraints still needed to close the detection-to-action gap.

6. **Knowledge gaps include missing domain practices, not just missing routing tables.** Agents building features won't infer standard industry practices (accessibility, performance regression, observability, error boundaries, dependency security) from bare capabilities. These are knowledge items — when framed as compact checklists rather than behavioral prohibitions, they fit within the token budget and transfer reliably. (Source: probe `2026-03-06-probe-worker-skill-industry-practice-gaps.md`)
7. **Auto-scorer indicators must be validated per-scenario against human ratings before trusting measurements.** S09/S13 validated (rho=0.980/0.894). S12 validated (rho=0.747). S11 not validated (rho=0.141) — indicators are uncorrelated with human judgment and need vocabulary redesign. Scorer vocabulary bias toward skill-enhanced responses means bare variant scores are systematically under-reported.

---

## Why This Fails

### Failure Mode 1: Constraint Dilution
At 10+ MUST/NEVER statements, agents treat ALL constraints as advisory. The 2,368-line v3 skill had 50+ constraints — effectively equivalent to bare Claude on 5 of 7 test scenarios.

### Failure Mode 2: Instruction Hierarchy Inversion
Claude Code's system prompt has ~500 words promoting Task tool; a skill's ~30 words constraining it face a 17:1 signal disadvantage. System prompt > user prompt by design. Prompt-level constraints cannot reliably override system-level defaults.

### Failure Mode 3: Dual Authority
When both a hook AND skill text prohibit the same action, agents receive conflicting signal types (infrastructure block vs prose guidance). This creates ambiguity about enforcement level and degrades trust in the skill overall.

### Failure Mode 4: Mechanical Staleness
The skill drifts from infrastructure quickly. 72 commits in 3 days introduced 10 changes, 6 requiring skill edits. Without a sync mechanism, the skill describes a system that no longer exists.

---

## Constraints

### Why can't we enforce behavior through skill text alone?

**Constraint:** Instruction hierarchy (system > user) means user-level skill content is structurally subordinate to system prompt defaults.

**Implication:** Behavioral constraints in skills are probabilistic suggestions, not deterministic rules. Only infrastructure hooks provide deterministic enforcement.

**This enables:** Dramatically simpler skill documents focused on knowledge transfer
**This constrains:** All behavioral enforcement must be implemented as hooks, not prose

### Why ≤ 4 behavioral norms?

**Constraint:** Empirically validated dilution threshold. At 5+ co-resident behavioral constraints, compliance drops toward bare baseline.

**Implication:** Every behavioral norm competes with every other. Adding a 5th doesn't add — it degrades the existing 4.

**This enables:** Ruthless prioritization of which behaviors matter most
**This constrains:** Cannot add "just one more" constraint without removing one

### Why must behavioral testing use pattern-match scoring, not LLM-as-judge?

**Constraint:** LLM-as-judge creates a closed evaluation loop — same model family evaluating same model family. Behavioral proxies (what the agent actually does) are more honest than self-assessment.

**Implication:** Test scenarios must define observable action patterns (contains/doesn't-contain), not qualitative rubrics.

**This enables:** Reproducible measurement of skill impact across versions
**This constrains:** Some subtle behaviors (tone, framing quality) are harder to test

---

## Evolution

**2026-01-18:** Skill is evolvable — added frustration trigger protocol as mode-shift gate. Skill operates as monolith (identity + constraints + knowledge all interleaved).

**2026-02-24:** Discovered structural problem. Identity compliance is additive (layers on defaults), but action constraints are subtractive (fight defaults). The skill was trying to do both, succeeding at identity, failing at constraints. Recommended two-layer fix: prompt restructuring + infrastructure enforcement.

**2026-02-28:** Intent spiral case study revealed skill's routing table is its most valuable section — caught ambiguous "evaluate Playwright CLI vs MCP" intent that bare Claude would route incorrectly.

**2026-03-01:** Behavioral testing infrastructure built. Measured: v3 skill scores 22/56 vs bare 17/56. 5-point lift concentrated in knowledge transfer (routing, vocabulary, framing). 5 of 7 scenarios at bare parity = dead-weight constraint text.

**2026-03-04:** Simplified v4 deployed: 2,368→422 lines (82% reduction). Removed all hook-enforced constraint text. Kept ≤4 behavioral norms as knowledge framing. Behavioral gate pending.

**2026-03-05:** 72-commit infrastructure delta created 6 mechanical mismatches. Skill drifts faster than anticipated — sync mechanism needed.

**2026-03-06 (morning):** Higher-N trials (N=6, 36 trials) decisively confirmed stance as a third content type distinct from knowledge. Scenario 09: bare 0/6, without-stance 1/6, with-stance 5/6. Model expanded from orchestrator-specific to universal taxonomy — the knowledge/behavioral/stance decomposition applies to all skills. Worker skill analysis shows investigation, systematic-debugging, and architect all have the same structural problem the orchestrator had: behavioral weight crowding out knowledge and stance.

**2026-03-06 (afternoon):** Generalization experiment (36 more trials, 3 new scenarios). Stance confirmed as cross-source reasoning primer: +5 on relationship tracing (S12), +4 on information freshness (S13), -2 on single-source absence detection (S11). The mechanism is specific — stance helps when defects hide between information sources, not when they're visible within one source. Detection-to-action gap discovered: agents notice problems but still approve completion. Action indicators at 0/6 across all variants.

**2026-03-06 (section audit):** Full section-by-section taxonomy audit of 4 worker skills (feature-impl, investigation, systematic-debugging, experiment, architect). All exceed ≤4 behavioral threshold except architect. Self-review is the largest behavioral block across all skills. Feature-impl needs stance injection. Model table expanded from summary to include line counts and behavioral compliance status.

**2026-03-06 (investigation stance):** Contrastive experiment (54 trials, 3 scenarios × 3 variants × N=6) shows investigation stance produces zero lift. Discovery: not all stances are equal. The orchestrator's attention primer ("look for implicit assumptions") works because it changes perception. The investigation's action directive ("test before concluding") doesn't work because agents can't execute tests in --print mode. Model refined: Invariant 5 updated to distinguish attention primers from action directives.

**2026-03-06 (calibration):** Human calibration experiment validates measurement program. 24 blind-rated responses across 4 scenarios × 3 variants. Overall Spearman rho=0.637 passes r>0.6 gate. Per-scenario analysis reveals S11 indicators are broken (rho=0.141) — need vocabulary redesign. S09 (0.980) and S13 (0.894) indicators are near-perfect proxies for human judgment. Scorer vocabulary bias toward skill-enhanced responses confirmed: 3/4 biggest disagreements on bare variants. Stance lift washes out in aggregate (without-stance 4.2 vs with-stance 4.1) — confirms stance is scenario-specific, not universal.

---

## Open Questions

1. **Do worker skill stances actually transfer?** Investigation stance ("test before concluding") does NOT transfer — zero lift across 54 trials. Root cause: it's an action directive, not an attention primer. The critical distinction is attention primers (change perception) vs action directives (change behavior). Systematic-debugging ("understand before fixing") and architect ("decide what should exist") are untested — both may be attention primers (closer to the orchestrator pattern). Next: reframe investigation stance as attention primer ("look for what the artifact DIDN'T examine") and retest.

2. **What's the right stance density?** The orchestrator has ~3 stance items. Is there a saturation point for stance like there is for behavioral constraints (5+)? *Update (Mar 6):* Human calibration data shows variant means across all scenarios: bare=2.0, without-stance=4.2, with-stance=4.1. Stance lift washes out when averaged — confirms stance is scenario-specific (cross-source reasoning), not universal. Density experiments should focus on per-scenario-type effects, not aggregate scores.

3. **Does stance interact with knowledge?** The N=6 data shows knowledge alone (without-stance) scores 17% on implicit contradictions vs 83% with stance. Does stance amplify knowledge, or are they independent? *Update (Mar 6):* Human calibration confirms the aggregated without-stance and with-stance means are nearly identical (4.2 vs 4.1), but per-scenario the picture differs dramatically (S09 rho=0.980 vs S11 rho=0.141). The interaction is scenario-type-dependent, not a universal amplification effect.

4. **How do we close the detection-to-action gap?** Stance improves detection (agents notice cross-source defects) but action indicators are at floor — agents still approve completion. Is this a behavioral constraint problem (need "do not approve when issues found"), an indicator design problem (detection vocabulary too narrow), or a fundamental limitation of `--print` mode (no tool execution)?

5. **Does stance hurt single-source scenarios?** S11 showed bare median 5 → stance median 3. Is this noise (N=6), or does cross-source priming actively distract from structurally visible defects?

---

## Actionable Implications

### For skill authors
- **Identify the stance** — every skill should have 1-3 lines of epistemic orientation. If you can't name it, the skill doesn't know what it's for.
- **Audit behavioral weight** — count MUST/NEVER/checklist items. If >4, move excess to hooks.
- **Test stance transfer** — write one contrastive scenario where stance and non-stance produce observably different responses.

### For the skill system
- **Worker skill simplification** — apply the orchestrator playbook (strip behavioral → hooks, keep knowledge + stance) to investigation (266 lines, 6+ behavioral mandates), systematic-debugging (752 lines), architect (673 lines), codebase-audit (1,490 lines).
- **Measurement infrastructure** — extend skillc test scenarios to worker skills. Single-turn stance scenarios are portable; multi-turn procedure testing remains an open design problem.

### Industry practice gaps by skill (from probe 2026-03-06)

| Skill | Critical Gap | Token Budget Status |
|-------|-------------|---------------------|
| feature-impl | **Accessibility (a11y) — completely absent.** Also missing: performance regression, error boundaries, observability, dependency audit. | 5,105 tokens (at budget) — add as concise checklist items or reference docs |
| systematic-debugging | Security impact assessment of discovered bugs — no check for whether the bug found is also a CVE/security escalation. | Within budget |
| ux-audit | Performance dimension absent (no Lighthouse, no Web Vitals). | Comprehensive otherwise |
| codebase-audit | Accessibility dimension absent. Dependency health dimension absent. | Within budget |
| design-session | Non-functional requirements prompting absent — a11y/performance/security specified in design, not retrofitted in implementation. | Within budget |

**The a11y gap is systemic:** missing in feature-impl (where code is written), codebase-audit (where issues are caught), and design-session (where requirements are specified). Only ux-audit has comprehensive a11y coverage — but it runs after features are built. Retrofitting a11y costs 5-10x more than building it in.

**Token budget tension resolution:** Add domain practice gaps as compact knowledge-framed checklist items (like the feature-gate addition), not MUST/NEVER constraints. Concise references fit within ~200 additional tokens. Detailed methodology goes in reference docs (progressive disclosure, already used by investigation skill).

---

## References

**Investigations:**
- `.kb/investigations/2026-01-18-inv-update-orchestrator-skill-add-frustration.md` - Added frustration trigger protocol
- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` - Discovered instruction hierarchy problem, 17:1 signal disadvantage
- `.kb/investigations/2026-03-01-design-infrastructure-systematic-orchestrator-skill.md` - Built behavioral testing infrastructure, 3-layer measurement
- `.kb/investigations/2026-03-04-design-simplify-orchestrator-skill.md` - Validated knowledge-only approach, deployed v4
- `.kb/investigations/2026-03-05-inv-design-orchestrator-skill-update-incorporating.md` - 72-commit delta sync, 6 surgical edits

**Evidence:**
- `evidence/2026-03-05-higher-n-09-10/` - Raw trial data: 36 trials, 3 variants × 2 scenarios × 6 runs
- `evidence/2026-03-06-human-calibration/` - Blind rating sheet, answer key, 24 transcripts across 4 scenarios × 3 variants
- `evidence/2026-03-06-investigation-stance-contrastive/` - 54 trials: investigation stance contrastive experiment (3 scenarios × 3 variants × N=6)
- `.kb/plans/2026-03-05-comprehension-measurement-program.md` - Research program design

**Decisions informed by this model:**
- `.kb/decisions/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` - Two-layer enforcement architecture

**Related models:**
- `.kb/models/architectural-enforcement/` - How hook infrastructure works
- `.kb/models/coaching-plugin/` - Agent behavioral coaching layer

**Threads:**
- `throughput-completions-vs-comprehension-completions` - Where the stance measurement program originated

### Merged Probes

| Probe | Date | Verdict | Key Finding |
|-------|------|---------|-------------|
| `probes/2026-03-06-probe-worker-skill-industry-practice-gaps.md` | 2026-03-06 | Extends | Knowledge gaps in skills extend beyond routing tables to missing domain practices. Systemic a11y absence across feature-impl, codebase-audit, design-session. Performance, observability, error boundary, dependency audit gaps in feature-impl. Feature-impl is at token budget (5,105) — concise checklist framing is the resolution. Confirms Invariant 3: knowledge framing (not prohibition) is the right format. |
| `probes/2026-03-06-probe-investigation-stance-transfer.md` | 2026-03-06 | Extends (partial contradiction) | Investigation stance ("test before concluding") produces zero lift (54 trials, bare median 4, stance median 3-4). Root cause: it's an action directive, not an attention primer. Attention primers (orchestrator) change perception and work in text output. Action directives (investigation) tell agents what to do but have no leverage when agents can't execute. Invariant 5 refined from "stance is a cross-source reasoning primer" to "stance transfers only as attention primer, not action directive." |
| `probes/2026-03-11-probe-exploration-mode-routing-dilution.md` | 2026-03-11 | Confirms | Adding 8 lines (~235 tokens) of exploration mode routing to orchestrator skill — all classified as knowledge content (routing tables, command reference). Structural analysis: 0/10 non-explore scenarios misrouted. Model's dilution mechanism is behavioral-only; knowledge additions don't dilute. Empirical A/B blocked by global Stop hook. Confirms knowledge transfers without degradation; budget already exceeded pre-addition (503 lines). |

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-05-design-claude-design-skill-evaluation.md
- .kb/investigations/2026-03-05-inv-skill-system-audit.md
