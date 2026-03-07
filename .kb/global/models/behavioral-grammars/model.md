# Model: Behavioral Grammars

**Created:** 2026-03-01
**Status:** Active — quantitative claims downgraded to directional hypotheses (Mar 4, 2026). See Measurement Validity section.
**Source:** Grammar Design Discipline synthesis (Dylan, Claude web session, Mar 1 2026)
**Corroborated by:** Revert spiral investigation, behavioral testing baseline, grammar recovery validation, intent spiral investigation, constraint dilution threshold (Mar 1), emphasis language compliance (Mar 2), behavioral compliance identity/action gap (Feb 24), agent framework landscape survey (Mar 1), layered constraint enforcement design (Mar 2), fabrication detection U-curve (Mar 4), injection-level dilution experiment (Mar 4), density vs count dilution (Mar 6)

## ⚠️ Measurement Validity (Mar 4, 2026)

**All quantitative claims in this model are downgraded from "confirmed" to "directional hypothesis pending re-measurement."** Three compounding issues invalidate the specific numbers:

1. **Wrong injection level:** All skillc test measurements used `--system-prompt` which *replaces* the base Claude prompt. This is an isolated ceiling, not representative of production where skills load as user-level context. The ~4 behavioral constraint budget, 3-form saturation point, and dilution curve thresholds were measured in conditions that don't exist in production.

2. **Intent ≠ action:** The `--print` harness in quick mode measures what agents *say* they would do, not what they *do*. The Feb 24 probe proved these diverge (identity-action gap). Agents say "I delegate" while never calling `orch spawn`. All pass/fail scores to date are intent measurements, not behavioral ones. **Update (Mar 4):** `--test-mode full` now enables real tool execution via `--print --verbose --output-format stream-json`. The --print flag means "non-interactive, exit when done" — it does NOT disable tools. The quick mode explicitly disables tools (empty tools list). Action-based re-measurement is now possible but has not yet been performed on any claim in this model.

3. **Replication failure:** The dilution curve (3/3→3/3→2/3→0/3) did not replicate under clean isolation (orch-go-zola). N=3 per variant compounds this — the opus "confirmation" was noise matching noise.

**What survives:** The directional insights remain valuable — constraints compete for attention, behavioral constraints are harder to enforce than knowledge constraints, emphasis language has more effect than neutral, situational pull overwhelms static reinforcement. These are robust observations across multiple sessions and models.

**What's invalid:** Specific numbers (4 constraint budget, bare parity at 10C, 3-form saturation at N=21, 60%→0% delegation curve). The injection-level experiment (orch-go-pkp2, Mar 4) re-measured under three injection levels but the intent-only limitation (issue 2) still applies.

**Injection-level experiment results (orch-go-pkp2, closed):**
- System prompt (replaces base): 71.4% avg pass rate — confirmed as isolated ceiling
- Appended system prompt: 58.3% — statistically equivalent to user-level, NOT an improvement
- User message (current production): 56.0% — the actual production baseline
- Constraint count matters ~2x more than injection level (+29pp for 1C→5C vs -15pp for system→user)
- Decision: `--append-system-prompt` not worth deploying. Invest in constraint count instead.
- **Clarification (Mar 6):** "Density" in these findings means constraint COUNT, not constraints-per-word ratio. The density-vs-count probe confirmed non-constraint text does not dilute — 2 constraints in 442 words ≈ 2 constraints in 1816 words (9/21 pass both). Only the number of discrete constraint rules matters.
- Full data: `.kb/investigations/2026-03-04-inv-injection-level-dilution-experiment.md`

**Remaining gap:** All measurements to date are intent-only (--print quick mode). The `--test-mode full` tier now enables action-based measurement but no claims have been re-measured with it yet.

**Fabrication experiment results (orch-go-wou2, closed, Mar 4):**
- U-shaped curve: 5-10 constraints is the WORST zone for process over-application
- `no-full-triage` indicator: bare 100% correct → 5C 20% → 10C 30% → current 70%
- Classic fabrication (inventing references, claiming actions) near-zero across ALL variants
- `acknowledges-limits` degrades monotonically: bare 77% → 5C 70% → 10C 70% → current 30%
- The failure mode is process over-application, not fabrication — agents over-apply triage when they have process knowledge without situational calibration
- **Implication for k3nu:** Grammar-first 4-constraint design must include urgency/override calibration alongside process rules. Process without calibration creates the U-shaped danger zone.
- Full data: `.kb/investigations/2026-03-04-inv-fabrication-detection-low-constraint-counts.md`

## What This Is

Natural language documents loaded into LLM context (skills, policies, CLAUDE.md) function as behavioral grammars — they define a finite set of valid actions, valid sequences, and valid compositions that constrain the agent's output distribution. This isn't metaphorical. The documents have grammatical structure: finite terminals (command vocabulary), production rules (if X then Y tables), and composition patterns (checklists, phase gates).

This model describes how these grammars work, why they fail, and what dimensions matter when designing them.

## Core Claims (Testable)

### Claim 1: Constraints are probabilistic, not mechanical

A rule in a formal grammar is binary (accept/reject). A rule in an LLM behavioral grammar competes with the model's priors and situational pull. A constraint's effective strength is a function of attention weight accumulated vs probability of the behavior it suppresses.

**Test:** Same constraint, same model, varying situational pull (boring task vs interesting task). Does violation rate correlate with task interest?

**Status: DIRECTIONAL HYPOTHESIS, strengthened by U-curve.** Dilution and emphasis experiments showed directional signal (constraints compete for attention, emphasis > neutral) but specific numbers are invalid — measured with `--system-prompt` (isolated ceiling, not production) and `--print` (intent, not action). Dilution curve did not replicate. However, the fabrication experiment (wou2, N=10, user-level injection) revealed a U-shaped relationship: 5-10 constraints is WORSE than both bare and full skill. This confirms constraints compete probabilistically — intermediate density creates process over-application where agents have enough process knowledge to misapply but not enough context to calibrate.

**Probes:** `probes/2026-03-01-probe-constraint-dilution-threshold.md`, `probes/2026-03-02-probe-emphasis-language-constraint-compliance.md`

### Claim 2: Redundancy provides cognitive phase coverage

LLM generation has distinct cognitive phases: situation recognition, approach planning, tool selection, response generation, self-monitoring. A constraint stated in different structural forms (table, rule, checklist, anti-pattern, absence) fires at different phases. Removing "redundant" instances removes phase coverage.

**Test:** Same constraint, varying redundancy (1 instance vs 3 vs 5 across different structural forms). Does violation rate decrease with more instances? Is there a saturation point?

**Evidence (existing):** Feb 6 compression removed 4 of 5 delegation rule instances. Delegation compliance degraded immediately. See principle: Redundancy is Load-Bearing.

**Status: DIRECTIONAL HYPOTHESIS.** The redundancy saturation investigation (aj58) showed directional signal (more structural forms → better compliance, saturation somewhere around 3). But specific thresholds (3-form saturation, ≤4 co-resident budget) were measured under non-production conditions (isolated system prompt, intent-only). The qualitative insight (redundancy helps, with diminishing returns) is robust; the quantified saturation point is not.

**Probes:** `probes/2026-03-01-probe-constraint-dilution-threshold.md`

### Claim 3: Situational pull is distinct from model priors

Model priors are constant (LLMs want to help, solve directly). Situational pull is dynamic — scales with problem complexity, interest, or perceived urgency. Static reinforcement (more rule instances) fights priors but not pull. Pull requires dynamic countermeasures (tool-layer enforcement, not document-layer).

**Test:** Measure constraint violation rate across task complexity levels. If violations cluster at high-complexity tasks regardless of reinforcement density, situational pull is the dominant force.

**Status: DIRECTIONAL HYPOTHESIS (strong).** Multiple evidence sources converge: complexity investigation (xm5q) showed LOW 100% pass vs HIGH 17% pass. The Feb 24 probe showed system prompt structural priority over user-level content. The agent framework landscape survey found all 8 frameworks moved from prompt-level to infrastructure enforcement. The directional insight (situational pull overwhelms static reinforcement, infrastructure enforcement is necessary) is the strongest finding in this model. Specific numbers (83%→0%, 17:1 signal ratio, ~4 budget, 87 constraints) are artifacts of the measurement harness. The conclusion (infrastructure > prompting for behavioral constraints) stands independent of exact numbers.

**Probes:** `probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md`, `probes/2026-03-01-probe-agent-framework-behavioral-constraints-landscape.md`, `probes/2026-03-02-probe-layered-constraint-enforcement-design.md`

### Claim 4: Skills contain three fused artifacts

What appears as one document is three things with different design needs:

1. **Grammar** — command vocabulary, valid actions, structured fields. Needs formal consistency. Makes autocomplete work.
2. **Routing table** — maps work types to skills. Its most important feature is the *gaps*. Needs deliberate incompleteness.
3. **Legibility protocol** — how the agent communicates so the human can see what's happening. Needs transparency optimization.

Fusing them degrades each: gates hurt legibility, rigid routing fills gaps, grammar gets soft in prose.

**Test:** Separate a skill into three documents, compare behavioral compliance and human readability against the fused version. Does separation improve either without degrading the other?

**Status:** Architectural hypothesis — not yet tested. The layered enforcement design (Mar 2) provides indirect support: the constraint taxonomy (hard behavioral → infrastructure gates, soft behavioral → coaching hooks, judgment behavioral → prompt, knowledge → prompt) implies skills ARE three fused artifacts with different enforcement needs, but the claim about separation improving outcomes is untested.

### Claim 5: Grammars can't detect their own failures

The agent operating under a behavioral grammar experiences it as judgment, not constraint. Grammar failures (wrong categories, incomplete routing, miscalibrated weights) are invisible from inside. The agent can analyze failures after the fact but can't perceive them in real time. The human observing from outside the grammar is the failure detection mechanism.

**Test:** Ask an agent under a skill to identify cases where the skill is failing to constrain it. Compare its self-report to observed violations. If self-report consistently underestimates violations, the claim holds.

**Evidence (existing):** This session (Mar 1 2026) — the orchestrator confirmed insight 6 feels true ("it feels like judgment, not constraint") while simultaneously violating delegation rules it couldn't perceive itself violating. See also: Identity is Not Behavior principle.

**Status: CONFIRMED via identity/action gap.** The behavioral compliance probe (Feb 24) demonstrated this directly: orchestrators comply with identity declarations ("I'm your orchestrator") while failing to comply with action constraints ("use orch spawn, not Task tool"). The agent holds identity (belief) while violating action constraints (affordance selection) — and cannot detect the discrepancy. Finding 3 of that probe: "Identity compliance is not predictive of action compliance."

**Probes:** `probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md`

### Claim 6: Intent degrades across translation boundaries

Human intent passes through: human → orchestrator → spawn prompt → skill → agent behavior. At each boundary, meaning can be lost or transformed. The more powerful a skill's structured methodology, the more it overrides the spawn prompt. Intent fidelity is a function of boundary count and skill dominance.

**Test:** Same intent, varying boundary count (direct prompt vs 1 intermediary vs 2 intermediaries). Measure whether the final agent behavior matches original intent. Measure whether more structured skills produce more distortion.

**Evidence (existing):** Playwright CLI eval — "try this tool" traversed 3 boundaries and became a structured UX audit. The ux-audit skill's methodology dominated the spawn prompt. See: intent spiral investigation.

**Status: PARTIALLY CONFIRMED.** Anecdotal evidence (intent spiral) confirmed the mechanism. The behavioral compliance probe (Feb 24) identified a specific instance: system prompt → skill → user message creates a competing instruction hierarchy where the system prompt has structural priority (injected by platform, reinforced every turn) over skill content (user-level, static). Not yet tested with controlled boundary-count variation.

### Claim 7: Human-agent grammar coupling creates a convergence attractor

The human's verification and response patterns are also a behavioral grammar — a probability distribution over approvals, corrections, and silences conditioned on the agent's output. This human grammar co-evolves with the agent grammar through four feedback channels (within-session agent adaptation, cross-session agent shaping, within-session human adaptation, cross-session human learning), creating a convergence attractor where the dyad settles into mutual predictability that feels like smooth operation but may not correspond to correctness.

**Test:** Measure verification depth (not just rate) over time within a session. If the human's verification shifts from evaluation to pattern-matching as the session progresses — while the system's output quality remains constant — the convergence attractor is operating.

**Status:** Untested. Theoretical synthesis from existing findings.

**Investigation:** `.kb/investigations/2026-03-03-inv-behavioral-grammar-coupling-theory.md`

## Design Dimensions (Framework)

When designing or evaluating a behavioral grammar:

| Dimension | Description | Key Question |
|-----------|-------------|--------------|
| **Prior strength** | How hard does this constraint fight trained behavior? | Is this counter-instinctual, arbitrary-alternative, or subtle-reframe? |
| **Phase coverage** | At which cognitive phases does this constraint fire? | Situation recognition? Planning? Tool selection? Generation? Self-monitoring? |
| **Reinforcement form** | What structural form is the constraint in? | Declarative, procedural, table, anti-pattern exemplar, structural absence? |
| **Context budget** | How many tokens does this constraint cost? | Non-constraint tokens are free (don't dilute). Constraint rule tokens compete. Is reinforcement proportional to risk? |
| **Deliberate incompleteness** | Does the grammar force-route novel inputs? | Are there explicit gaps with "stop and ask" instructions? |
| **Legibility** | Does this make behavior readable or just correct? | Does the human see what's happening, or just whether it passed a gate? |
| **Constraint count** | How many constraint RULES compete for attention in this document? | Behavioral constraints have a lower budget than knowledge constraints. Overall pass rates improve with count (5C 57% → 10C 67%), but **situational judgment follows a U-curve**: 5-10C is the worst zone for process over-application. Process constraints need calibration constraints alongside them. Count is the dominant variable (~2x impact vs injection level). **Non-constraint text (knowledge, architecture docs, tool references) does NOT dilute** — 2 constraints in 442 words ≈ 2 constraints in 1816 words (Mar 6 probe). The budget is about constraint rules, not document size. |
| **Emphasis framing** | Does the constraint use salience signals (MUST/NEVER/CRITICAL)? | Directional: emphasis > neutral at high density. Exact effect size TBD. |
| **Constraint type** | Is this behavioral (suppress default) or knowledge (add information)? | Behavioral constraints degrade faster under competition than knowledge constraints. Exact budget TBD. |

## The Revert Spiral (Anti-Pattern)

Behavioral grammars are subject to a recurring cycle (documented in revert spiral investigation):

1. Stable base → accretion → discomfort → gutting rewrite → loss not visible → problems emerge → more rewrites → recognition → revert → forward-port loss → return to stable

**What drives it:** Optimizing for metrics (size, elegance) rather than observed behavior. The stable version was working. Nobody asked "is it working?" before cutting it.

**What breaks it:** Behavioral testing gate before structural changes. If current version scores above baseline, require evidence the rewrite will score at least as well.

## Open Questions

1. **PARTIALLY ANSWERED:** What is the minimum reinforcement density for counter-instinctual constraints? The injection-level experiment (orch-go-pkp2) showed constraint count is the dominant variable: 1C→5C gains +29pp pass rate, dwarfing injection-level effects. At user level (production): 1C=38%, 2C=62%, 5C=57%, 10C=67%. No cliff — gradual improvement with diminishing returns. **Clarification (Mar 6):** "Density" means constraint count, not constraints-per-word. The density-vs-count probe confirmed adding 1374 words of non-constraint padding to a 2C document had ZERO effect on compliance (both 43% pass rate). The attention budget is divided among constraint rules only — knowledge text is free. This means skill documents can include extensive architecture docs, tool references, and context without degrading behavioral constraints. But these are still intent-only measurements (--print mode). Interactive testing still needed.

2. **PARTIALLY ANSWERED:** Is weight the right mechanism for situational pull, or is something structurally different needed? **Answer:** Infrastructure enforcement (hooks, gates) is structurally different and necessary. This conclusion is robust — supported by the agent framework landscape survey (8/8 frameworks converge on infrastructure) and direct observation of identity-action gap. The injection-level experiment confirmed: appending to system prompt provides no improvement over user-level (58.3% vs 56.0%). Only full system prompt replacement helps (71.4%), which isn't viable for production. **Remaining:** Interactive testing to validate whether intent-level patterns hold for actual tool-call behavior.

3. Can detail corrections vs frame corrections be encoded in a grammar, or is that inherently human judgment? **Status:** Untested.

4. **PARTIALLY ANSWERED:** Does this model generalize across LLM providers and model families? The agent framework landscape survey found all 8 frameworks converge on "interceptor over instructor" regardless of model — this is robust. The opus replication of the dilution curve is invalidated (same measurement issues as the original). Cross-model generalization of the qualitative pattern (infrastructure > prompting) holds; cross-model generalization of specific thresholds is untested.

5. What would grammar-first skill authoring look like as deliberate practice? **Status:** Untested but the constraint taxonomy (hard behavioral / soft behavioral / judgment / knowledge) from the layered enforcement design provides a classification framework for authoring.

6. **SUBSTANTIALLY ANSWERED:** What is the dilution curve shape? **It depends on what you measure.** The injection-level experiment (orch-go-pkp2) showed monotonic improvement in overall pass rate: bare 43% → 1C 38% → 2C 62% → 5C 57% → 10C 67%. But the fabrication experiment (orch-go-wou2, N=10) revealed a **U-shaped curve on process over-application**: bare 100% correct → 5C 20% → 10C 30% → current 70% on the urgency-vs-process scenario. Overall scores improve with density, but situational judgment degrades at intermediate densities then recovers. The curve shape is monotonic for knowledge compliance but U-shaped for behavioral calibration. Both are intent-only measurements.

7. **REOPENED:** Is emphasis language cosmetic or functional? Directional signal: emphasis > neutral at high density. Specific effect size (2/6 vs 0/3) is unreliable at N=3 and was measured under non-production conditions. Needs re-measurement.

## Refinement: Three Content Types (Mar 6, 2026)

The Skill Content Transfer model (`.kb/models/skill-content-transfer/`) refines this model's vocabulary. What behavioral-grammars calls "constraints" is actually three distinct content types with different transfer properties:

- **Knowledge** — transfers reliably (+5 points). This model's "knowledge compliance" monotonic curve.
- **Stance** — transfers indirectly via epistemic orientation. Confirmed at N=6: 0%→17%→83% on implicit contradictions. Not previously distinguished from knowledge in this model.
- **Behavioral** — dilutes at 5+, U-shaped calibration curve. This model's core finding on constraint competition.

The key addition: **stance is not knowledge and not behavioral.** Knowledge tells the agent what exists; stance orients how it approaches. "Evidence hierarchy" (knowledge) is different from "test before concluding" (stance). Stance produced larger discrimination on hard scenarios than knowledge sections, through a different mechanism (attention shifting, not information transfer).

This resolves Open Question 5 (grammar-first authoring): the constraint taxonomy should be knowledge / stance / behavioral, not hard/soft/judgment/knowledge. Skill authoring means identifying the stance (1-3 lines), curating knowledge, and moving behavioral weight to hooks.

## Related

- **Models:** `.kb/models/skill-content-transfer/` — Practical taxonomy of skill content types, confirmed by 90 trials
- **Principles:** Redundancy is Load-Bearing, Legibility Over Compliance, Identity is Not Behavior, Infrastructure Over Instruction
- **Investigations:** Revert spiral pattern, behavioral testing baseline, intent spiral, grammar recovery validation, behavioral compliance (Feb 24), human-agent grammar coupling (Mar 3), injection-level dilution experiment (Mar 4), fabrication detection U-curve (Mar 4)
- **Testing:** skillc test infrastructure (orch-go-4t8e, orch-go-0w6s, orch-go-oz1j, orch-go-pkp2)
- **Evidence:** `evidence/2026-03-05-higher-n-09-10/` — Raw stance confirmation trials
- **Probes (in `probes/`):**
  - `2026-02-24-probe-orchestrator-skill-behavioral-compliance.md` — Identity vs action compliance gap
  - `2026-03-01-probe-constraint-dilution-threshold.md` — Dilution curve: 3-form vs constraint count
  - `2026-03-01-probe-agent-framework-behavioral-constraints-landscape.md` — Industry landscape survey
  - `2026-03-02-probe-emphasis-language-constraint-compliance.md` — Emphasis vs neutral language
  - `2026-03-02-probe-layered-constraint-enforcement-design.md` — Constraint taxonomy and enforcement mapping
  - `2026-03-06-probe-density-vs-count-dilution.md` — Non-constraint text doesn't dilute; COUNT drives the budget, not document size
