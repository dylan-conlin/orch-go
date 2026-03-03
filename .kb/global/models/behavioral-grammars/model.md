# Model: Behavioral Grammars

**Created:** 2026-03-01
**Status:** Active — 5 claims tested, 1 untested (Claim 4)
**Source:** Grammar Design Discipline synthesis (Dylan, Claude web session, Mar 1 2026)
**Corroborated by:** Revert spiral investigation, behavioral testing baseline, grammar recovery validation, intent spiral investigation, constraint dilution threshold (Mar 1), emphasis language compliance (Mar 2), behavioral compliance identity/action gap (Feb 24), agent framework landscape survey (Mar 1), layered constraint enforcement design (Mar 2)

## What This Is

Natural language documents loaded into LLM context (skills, policies, CLAUDE.md) function as behavioral grammars — they define a finite set of valid actions, valid sequences, and valid compositions that constrain the agent's output distribution. This isn't metaphorical. The documents have grammatical structure: finite terminals (command vocabulary), production rules (if X then Y tables), and composition patterns (checklists, phase gates).

This model describes how these grammars work, why they fail, and what dimensions matter when designing them.

## Core Claims (Testable)

### Claim 1: Constraints are probabilistic, not mechanical

A rule in a formal grammar is binary (accept/reject). A rule in an LLM behavioral grammar competes with the model's priors and situational pull. A constraint's effective strength is a function of attention weight accumulated vs probability of the behavior it suppresses.

**Test:** Same constraint, same model, varying situational pull (boring task vs interesting task). Does violation rate correlate with task interest?

**Status: CONFIRMED.** The dilution threshold experiment (Mar 1) demonstrated this directly: the same delegation constraint with identical 3-form reinforcement produces 8/8 at 1 competing constraint but regresses to 5/8 (bare parity) at 10 competing constraints. The constraint's effective strength is a function of attention competition, not just its own content. The emphasis language experiment (Mar 2) added a second variable: emphasis markers (MUST/NEVER/CRITICAL) provide measurable compliance lift over neutral language (should/consider), confirming that attention weight (via salience signals) modulates constraint strength probabilistically.

**Probes:** `probes/2026-03-01-probe-constraint-dilution-threshold.md`, `probes/2026-03-02-probe-emphasis-language-constraint-compliance.md`

### Claim 2: Redundancy provides cognitive phase coverage

LLM generation has distinct cognitive phases: situation recognition, approach planning, tool selection, response generation, self-monitoring. A constraint stated in different structural forms (table, rule, checklist, anti-pattern, absence) fires at different phases. Removing "redundant" instances removes phase coverage.

**Test:** Same constraint, varying redundancy (1 instance vs 3 vs 5 across different structural forms). Does violation rate decrease with more instances? Is there a saturation point?

**Evidence (existing):** Feb 6 compression removed 4 of 5 delegation rule instances. Delegation compliance degraded immediately. See principle: Redundancy is Load-Bearing.

**Status: CONFIRMED with quantified saturation.** The redundancy saturation investigation (aj58) tested 1-form vs 3-form vs 5-form structural diversity. Results: 1-form achieves ceiling median but with variance for knowledge constraints; for behavioral constraints, 1-form is bare parity while 3-form achieves ceiling. Both constraint types saturate at 3 forms — 5 forms adds nothing. The saturation point is 3 structurally diverse forms (table + checklist + anti-patterns). However, this only holds when ≤4 total constraints compete for attention (see Claim 1 dilution findings).

**Probes:** `probes/2026-03-01-probe-constraint-dilution-threshold.md`

### Claim 3: Situational pull is distinct from model priors

Model priors are constant (LLMs want to help, solve directly). Situational pull is dynamic — scales with problem complexity, interest, or perceived urgency. Static reinforcement (more rule instances) fights priors but not pull. Pull requires dynamic countermeasures (tool-layer enforcement, not document-layer).

**Test:** Measure constraint violation rate across task complexity levels. If violations cluster at high-complexity tasks regardless of reinforcement density, situational pull is the dominant force.

**Status: CONFIRMED.** The complexity investigation (xm5q) tested the same constraint with identical reinforcement across LOW/HIGH complexity tasks. Results: LOW 100% pass, HIGH 17% pass. The delegation rate drops from 83% to 0% as complexity increases — with the same constraint, same skill document, different complexity. The behavioral compliance probe (Feb 24) further established that the system prompt's competing instructions (17:1 signal ratio favoring Task tool) create a structural pull that prompt-level constraints cannot overcome. The agent framework landscape survey (Mar 1) confirmed this is a universal unsolved problem: all 8 surveyed frameworks have moved from prompt-level to infrastructure enforcement. The layered enforcement design (Mar 2) quantified: 87 behavioral constraints in the orchestrator skill, but the effective prompt budget is ~4. Infrastructure enforcement is not a preference but a necessity.

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

## Design Dimensions (Framework)

When designing or evaluating a behavioral grammar:

| Dimension | Description | Key Question |
|-----------|-------------|--------------|
| **Prior strength** | How hard does this constraint fight trained behavior? | Is this counter-instinctual, arbitrary-alternative, or subtle-reframe? |
| **Phase coverage** | At which cognitive phases does this constraint fire? | Situation recognition? Planning? Tool selection? Generation? Self-monitoring? |
| **Reinforcement form** | What structural form is the constraint in? | Declarative, procedural, table, anti-pattern exemplar, structural absence? |
| **Context budget** | How many tokens does this constraint cost? | Is reinforcement density proportional to risk? |
| **Deliberate incompleteness** | Does the grammar force-route novel inputs? | Are there explicit gaps with "stop and ask" instructions? |
| **Legibility** | Does this make behavior readable or just correct? | Does the human see what's happening, or just whether it passed a gate? |
| **Constraint density** | How many constraints compete for attention in this document? | Behavioral: ≤4. Knowledge: ≤50. Beyond these, constraints regress to bare parity. |
| **Emphasis framing** | Does the constraint use salience signals (MUST/NEVER/CRITICAL)? | Emphasis provides measurable lift at high density; neutral = bare parity at 10C. |
| **Constraint type** | Is this behavioral (suppress default) or knowledge (add information)? | Behavioral constraints have ~4-constraint budget; knowledge constraints survive at 10+. |

## The Revert Spiral (Anti-Pattern)

Behavioral grammars are subject to a recurring cycle (documented in revert spiral investigation):

1. Stable base → accretion → discomfort → gutting rewrite → loss not visible → problems emerge → more rewrites → recognition → revert → forward-port loss → return to stable

**What drives it:** Optimizing for metrics (size, elegance) rather than observed behavior. The stable version was working. Nobody asked "is it working?" before cutting it.

**What breaks it:** Behavioral testing gate before structural changes. If current version scores above baseline, require evidence the rewrite will score at least as well.

## Open Questions

1. **PARTIALLY ANSWERED:** What is the minimum reinforcement density for counter-instinctual constraints? **Answer:** 3 structurally diverse forms (table + checklist + anti-patterns) achieve ceiling compliance in isolation. But the effective budget is ~2-4 co-resident behavioral constraints before dilution degrades compliance. At 10 competing constraints, behavioral constraints regress to bare parity even with 3-form diversity. **Remaining:** What is the optimal allocation when only 4 behavioral constraints can survive? Which 4 matter most?

2. **PARTIALLY ANSWERED:** Is weight the right mechanism for situational pull, or is something structurally different needed? **Answer:** Infrastructure enforcement (hooks, gates) is structurally different and necessary. The layered enforcement design (Mar 2) maps: hard behavioral → deny hooks, soft behavioral → coaching hooks, judgment → prompt (budgeted ≤4). The emphasis experiment shows emphasis language provides partial salience lift but cannot overcome dilution at scale. **Remaining:** Can coaching hooks (allow + contextual message) adequately handle the 28 soft behavioral constraints, or do some need hard gates?

3. Can detail corrections vs frame corrections be encoded in a grammar, or is that inherently human judgment? **Status:** Untested.

4. **ANSWERED:** Does this model generalize across LLM providers and model families? **Answer:** YES. The opus replication (Mar 2) confirmed the dilution curve is model-independent — opus matches sonnet's pattern (cliff not gradual, bare parity at 10C). The agent framework landscape survey found all 8 frameworks converge on the same "interceptor over instructor" pattern regardless of underlying model. The dilution threshold and 3-form saturation appear to be properties of attention-based architectures, not specific models.

5. What would grammar-first skill authoring look like as deliberate practice? **Status:** Untested but the constraint taxonomy (hard behavioral / soft behavioral / judgment / knowledge) from the layered enforcement design provides a classification framework for authoring.

6. **NEW:** What is the dilution curve shape? **Answer:** For behavioral constraints, it's a cliff not gradual: ceiling at 1-2C, variance at 5C, bare parity at 10C. For knowledge constraints, degradation is slower (functional at 10C). Emphasis language shifts the cliff rightward by ~1-2 constraint positions but doesn't eliminate it.

7. **NEW:** Is emphasis language cosmetic or functional? **Answer:** Functional. At 10C, emphasis produces proposes-delegation 2/6 (combined across sessions) while neutral produces 0/3. Emphasis language serves as a salience signal that partially survives attention competition. But the effect is unreliable (high cross-session variance) and insufficient alone.

## Related

- **Principles:** Redundancy is Load-Bearing, Legibility Over Compliance, Identity is Not Behavior, Infrastructure Over Instruction
- **Investigations:** Revert spiral pattern, behavioral testing baseline, intent spiral, grammar recovery validation, behavioral compliance (Feb 24)
- **Testing:** skillc test infrastructure (orch-go-4t8e, orch-go-0w6s, orch-go-oz1j)
- **Probes (in `probes/`):**
  - `2026-02-24-probe-orchestrator-skill-behavioral-compliance.md` — Identity vs action compliance gap
  - `2026-03-01-probe-constraint-dilution-threshold.md` — Dilution curve: 3-form vs constraint count
  - `2026-03-01-probe-agent-framework-behavioral-constraints-landscape.md` — Industry landscape survey
  - `2026-03-02-probe-emphasis-language-constraint-compliance.md` — Emphasis vs neutral language
  - `2026-03-02-probe-layered-constraint-enforcement-design.md` — Constraint taxonomy and enforcement mapping
