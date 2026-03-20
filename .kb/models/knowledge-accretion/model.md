# Model: Knowledge Accretion

**Domain:** Multi-Agent Knowledge Systems
**Last Updated:** 2026-03-20
**Validation Status:** WORKING HYPOTHESIS — not externally validated. Built by one person with AI agents that optimize for coherence. Independent external review (Codex, Mar 10) identified core claims as restatements of existing governance/coordination concepts (Ostrom, Conway, Brooks) with agent vocabulary. Observations are real; theoretical framing is overclaimed. See `.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md`.
**Synthesized From:**
- `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-accretion-accretion-attractor-gate-dynamics.md` — Empirical measurement across 1,166 investigations, 32 models, 187 probes
- `.kb/models/harness-engineering/model.md` — Hard/soft harness taxonomy, accretion as thermodynamics, compliance vs coordination failure
- `.kb/models/system-learning-loop/model.md` — Gap→pattern→suggestion→improvement as proto-knowledge-accretion
- `.kb/models/skill-content-transfer/model.md` — Attention primers vs action directives, three-type vocabulary
- `.kb/models/entropy-spiral/model.md` — Feedback loops, control plane immutability, 1,625 lost commits

---

## Summary (30 seconds)

**Observed pattern (one system, 3 months):** When multiple amnesiac AI agents contribute to shared files and knowledge artifacts, those artifacts grow and degrade even though individual contributions are correct. Measured in orch-go: 85.5% orphan rate (997/1,166 investigations unconnected to any model), daemon.go grew +892 lines from 30 correct commits, 100% merge conflict rate in coordination demo (N=10). These observations are consistent with known coordination cost and governance concepts (Ostrom, Conway, Brooks). Whether the pattern generalizes beyond this one system is untested. The "five conditions" framing is a working hypothesis for organizing these observations, not a validated diagnostic framework.

---

## Theory Type

**Status: overclaimed.** The name "knowledge accretion" implies a level of generality and novelty that independent review did not support. The observations it contains (file growth, orphan rates, coordination failures) are real but well-described by existing concepts: Ostrom's commons governance, Conway's Law, coordination costs, institutional drift, tech debt dynamics. The vocabulary ("accretion dynamics," "substrate-independent," "entropy") adds no predictive power beyond these existing concepts.

**What to keep:** The observations, the measurements, the five conditions as an organizing checklist for this system.
**What to stop claiming:** That this is a novel framework, that it's "substrate-independent," that it constitutes a new diagnostic discipline.

---

## Core Claim

**Working hypothesis:** Shared artifacts degrade from correct contributions when agents lack coordination. This is a restatement of known coordination cost dynamics applied to AI agent workflows.

Five conditions appear to correlate with degradation in orch-go (untested elsewhere):

1. **Multiple agents write** to the substrate
2. **Agents are amnesiac** — no cross-session memory
3. **Contributions are locally correct** — each passes local validation
4. **Contributions must compose non-trivially** — coherence between contributions is required and not automatic
5. **No structural coordination mechanism exists** — locally correct + locally correct ≠ globally correct

Conditions 1-3 are context-setters (common in most modern systems). Condition 4 distinguishes compositional substrates (code, knowledge bases, schemas) from additive substrates (append-only logs, sensor data, votes) where contributions are independent and cannot compose incorrectly. Condition 5 is the lever — the presence or absence of coordination mechanisms determines whether accretion occurs or is managed.

**Note:** A previous version of this model included a formula (`accretion_risk = f(amnesia_level × compositional_complexity / coordination_strength)`). This was removed — none of the variables have units or measured values. It was a formula-shaped sentence, not a measurement.

**Observed in:** Code (daemon.go +892 lines from 30 correct commits), knowledge artifacts (85.5% orphan rate), runtime behavior (O(n) operations where N grew silently from correct usage — 5 instances measured 2026-03-20), governance infrastructure itself (35% of codebase is accretion management, growing faster than core — 18%→23% in March 2026, confirming KA-10). Whether the pattern holds in other contexts is unknown — no external validation exists.

---

## Core Mechanism

### 1. Accretion Dynamics

Accretion is entropy: individually correct contributions compose into structural degradation when shared infrastructure is missing. In code, this manifests as file bloat and duplication. In knowledge, it manifests as orphan investigations and semantic overlap.

**Empirical measurement (2026-03-09, updated with era-adjusted analysis):**

| Category | Connected | Total | Orphan Rate |
|----------|-----------|-------|-------------|
| Pre-model era (Dec 2025 - Jan 2026) | 51 | 969 | **94.7%** |
| Model era (Feb - Mar 2026) | 94 | 196 | **52.0%** |
| **All investigations** | **145** | **1,166** | **87.6%** |

**Orphan taxonomy (from 35-file sample, Mar 9):**

| Category | Rate | Natural? |
|----------|------|----------|
| Implementation-as-investigation | 30-45% | Yes — wrong skill routing |
| Audit/design | 25-33% | Yes — point-in-time snapshots |
| Exploratory | 15-20% | Yes — one-off questions |
| Genuinely lost | ~20% of orphans | **No — knowledge loss** |
| Negative results | 5-7% | Yes — valuable to record |
| Superseded | 3-5% | Yes — natural lifecycle |

**Automated stratified analysis (Mar 17, N=1170 orphans via `orch kb orphans --stratified`):**

| Category | Count | % of Orphans | Natural? |
|----------|-------|--------------|----------|
| empty | 0 | 0.0% | Yes — template-only files |
| negative-result | 145 | 12.4% | Yes — already-fixed, not-a-bug, works-as-expected |
| superseded | 25 | 2.1% | Yes — replaced/merged into later work |
| positive-unlinked | 1000 | 85.5% | **No — genuine findings without model connection** |

**Key finding:** The original sample-based estimate (~20% genuinely lost) was too conservative. Automated content analysis shows 85.5% of orphans have positive findings that aren't connected to any model, decision, or guide. The negative-result rate (12.4%) is higher than the sample estimated (5-7%), confirming that automated detection catches more signals than filename-based sampling.

**Natural baseline: 40-50% orphan rate is healthy.** The actionable signal is the positive-unlinked rate (78.4% of total investigations), not the raw orphan rate. However, most positive-unlinked orphans are pre-model era implementation work that was structurally impossible to connect at creation time.

**Knowledge accretion signatures:**
- **Orphan investigations** — work product with no structural connection to synthesized understanding (models). 87.6% overall, but 52.0% in the model era. Pre-model era (83% of corpus) inflates the aggregate.
- **Quick entry duplication** — confirmed duplicate pair (kb-69d5cf and kb-9f3964, both recording "verify.Comment uses Text field"). No automated dedup checking exists.
- **Synthesis backlog** — 4 clusters totaling 17 investigations that should be models but aren't. Detection is retroactive (detects bloat after accumulation, not at contribution time).
- **Investigation overlap** — multiple investigations covering the same ground without awareness of each other (no Prior Work gate enforced).

**Code accretion signatures (for comparison):**
- **File bloat** — daemon.go +892 lines in 60 days from 30 correct commits
- **Cross-cutting duplication** — 6 concerns independently reimplemented across 4-9 files (~2,100 lines)
- **Re-accretion** — spawn_cmd.go shrank -1,755 then regrew +483 in 3 weeks

**The parallel:** In both substrates, the mechanism is identical — agents contributing locally correct work that doesn't compose. The difference is measurement maturity: code has lines-per-file, duplication detection, and hotspot analysis. Knowledge has only the orphan rate and synthesis backlog (both measured here for the first time).

### 2. Attractor Taxonomy

Models and packages serve the same function: structural destinations that route contributions. But not all models behave as attractors. Three distinct behaviors observed:

| Model | Created | Pre-model ref rate | Post-model ref rate | Probes | Behavior |
|-------|---------|-------------------|---------------------|--------|----------|
| harness-engineering | 2026-03-08 | 2.0% | 100% | 3 | **Strong attractor** (launch burst) |
| daemon-autonomous-operation | 2026-02-08 | 12.5% | 50.0% | 34 | **Sustained attractor** |
| entropy-spiral | 2026-02-25 | 12.5% | 8.8% | 2 | **Capstone** (settled topic) |
| beads-database-corruption | — | — | — | 0 | **Dormant** (32 days stale) |

**Three model behaviors:**

- **Attractor** — Model creation increases investigation density toward it. The model actively pulls new work. Mechanism: agents spawned in the model's domain receive model claims via kb context injection, which frames their investigation. Daemon-autonomous-operation attracted 34 probes across 13 dates over 21 days — sustained gravitational pull.

- **Capstone** — Model creation *decreases* investigation density. It synthesizes and settles a topic. Entropy-spiral's reference rate dropped from 12.5% to 8.8% after formalization. The model absorbed the open questions; there was less to investigate.

- **Dormant** — Model exists but generates no probes or investigations. 7 models with 0 probes. These are either complete (all claims verified), abandoned (topic no longer relevant), or forgotten (no agent is spawned in their domain).

**Knowledge attractors vs code attractors:**

In code, a package attractor pulls code mechanically — imports, function calls, compilation. When `pkg/spawn/backends/` exists, Go's type system routes spawn code there.

In knowledge, a model attractor pulls attention — agents spawned in the model's domain receive model claims via kb context injection, which frames their investigation. The mechanism is **attention priming** (same as stance transfer from the skill-content-transfer model), not structural routing.

This is a key asymmetry: code attractors are structurally coupled (compiler enforces), knowledge attractors are attention-primed (context injection influences). Attention-primed attractors may be fundamentally weaker than structurally-coupled attractors — or they may just be ungated.

### Coordination Taxonomy

Coordination mechanisms come from three sources. Accretion occurs when ALL sources are absent for a given substrate:

| Source | Mechanism | Examples | Digital Substrates? |
|--------|-----------|----------|-------------------|
| **Explicit** | Engineered rules and enforcement | Type systems, schemas, CI, code review, pre-commit hooks | Yes — primary source |
| **Substrate-embedded** | Mathematical/physical properties of the substrate guarantee coherence | CRDTs (convergence by construction), strongly-typed languages (type safety) | Rare — must be engineered |
| **Environmental** | The environment mediates between agents | Stigmergy (pheromone trails), physical constraints, chemical gradients | No — biological substrates only |

**Why digital substrates require engineered coordination:** A `.go` file doesn't resist bloat through physics. A `.kb/` directory doesn't resist orphan investigations through chemistry. Unlike biological substrates (where environmental coordination emerges naturally), digital substrates lack implicit coordination — requiring explicit engineering of gates and attractors. CRDTs are the notable exception: a digital substrate with coordination embedded in its mathematical structure.

### 3. Gate Deficit

Knowledge transitions are mostly ungated, with two exceptions discovered via intervention audit (2026-03-20):

| Transition | Status | Gate Mechanism |
|------------|--------|---------------|
| Investigation → model | **UNGATED** | No automated model update when investigation contradicts model |
| Probe → model update | **HARD GATE** | `pkg/verify/probe_model_merge.go` blocks completion when probes with "contradicts" or "extends" verdicts haven't been merged into parent model. `result.Passed = false` on unmerged probes |
| Model template → commit | **HARD GATE** | `pkg/verify/model_stub_precommit.go` blocks committing model.md files with unfilled template placeholders. Override: `FORCE_MODEL_STUB=1` |
| Quick entry → decision | **UNGATED** | No dedup checking against existing entries or decisions |
| Decision → implementation | **UNGATED** (1/56 exception) | Only 1 of 56 decisions has a `kb agreements` check |
| Investigation prior work | **SOFT** (52% adoption) | Template includes it, 48% of investigations skip it |
| Knowledge consistency at commit | **UNGATED** | Pre-commit hooks only run on `*.go` files, not `.kb/` files |

**Correction (2026-03-20):** Previous versions of this table claimed "zero hard knowledge gates." The probe-to-model merge gate was implemented but not recorded here. The model-stub gate was acknowledged in invariant #1 but missing from this table. Two hard knowledge gates exist; four transitions remain ungated.

**The parallel to code is directional, not exact.** The harness-engineering model documented that "every convention without a gate will eventually be violated." Most knowledge transitions confirm this:

- Prior Work tables are a convention → 48% of investigations skip them
- Quick entry uniqueness is a convention → confirmed duplicates exist
- Decision enforcement is a convention → 1.8% enforcement rate

But probe-to-model merge is now a hard gate (not just a convention), and model-stub validation is a hard gate. These two gates prevent the specific failure modes they target. The remaining four ungated transitions still degrade under pressure.

### 3a. Intervention Effectiveness (2026-03-20 Audit)

**Of 31 interventions proposed or implemented in this model, only 4 (13%) have measurable evidence of reducing their target.** The rest are advisory (no behavioral change from warnings), not implemented, removed (negative ROI), or measurement-only.

**What demonstrably reduced accretion:**

1. **Daemon extraction cascades** (triggered by gate *events*, not gate *blocks*): 12→3 CRITICAL files (75% reduction)
2. **Model/probe directory system** (structural attractor): orphan rate 94.7%→52.0% in model-era
3. **Model-stub pre-commit gate** (preventive hard gate): blocks unfilled templates
4. **Probe-to-model merge gate** (completion hard gate): forces findings into parent models

**Effectiveness hierarchy (empirical):** Structural attractors > signaling mechanisms (event→daemon) > blocking gates (bypassed 100%) > advisory gates (ignored) > metrics-only (awareness without action).

**Gate lifecycle arc (observed pattern):** Every blocking gate followed: designed → measured → found inert or high-FP → downgraded/removed. Health score gate: 1 day. Self-review gate: 1 week (79% FP, 0 TP). Accretion blocking: ~3 weeks (100% bypass). Gates survive only when structurally unbypassable (`go build`, model-stub precommit) or when they trigger automated responses (daemon extraction via events).

**Full scorecard:** See `probes/2026-03-20-probe-intervention-effectiveness-audit.md`

### 4. Entropy Metrics

**Currently tracked (via `kb reflect`):**
- Synthesis opportunities (clusters of 3+ investigations on same topic)
- Stale decisions (>7 days, zero citations)
- Stale models (days without a probe)
- Recurring defect classes

**What's missing — knowledge equivalents of code metrics:**

| Code Metric | Knowledge Equivalent | Status |
|-------------|---------------------|--------|
| Lines of code per file | Claims per model | **Not tracked** |
| File bloat (>1,500 lines) | Model bloat (>N claims unprobed) | **Not tracked** |
| Fix:feat ratio | Contradiction:extension ratio | **Not tracked** |
| Duplication across files | Semantic overlap across investigations | **Not tracked** |
| Churn rate | Investigation-to-model conversion rate | **Partially** (synthesis opportunities) |
| Dead code | Dead investigations (orphaned, never cited) | **Partially** (85.5% orphan rate) |

**Proposed knowledge entropy metrics:**

1. **Orphan rate** — % investigations not connected to any model (baseline: 85.5%). Should decrease over time as models absorb findings. High orphan rate signals systemic under-synthesis.

2. **Model probe freshness** — days since last probe per model. Sustained staleness signals dormancy or completeness. The distinction matters: dormant models may need retirement; complete models need no action.

3. **Contradicts backlog** — count of "contradicts" probe verdicts sitting unmerged (baseline was 4, resolved to 0 on 2026-03-09). These are the knowledge equivalent of known bugs left unfixed. Direct measure of model accuracy debt. Expected to recur without hard gates.

4. **Synthesis debt** — investigation clusters without models (baseline: 4 clusters, 17 investigations). Measures how much investigative work product remains unsynthesized.

5. **Claim density** — claims per model (equivalent of lines per file). Too many claims → model needs splitting. Models should be focused enough that each claim can be probed.

6. **Quick entry duplication rate** — duplicate entries in entries.jsonl. Measures whether the "quick capture" path is creating noise instead of signal. No automated dedup exists.

7. **Composite health score** — `orch health` implements a 5-dimension 0-100 score: gate coverage, accretion control, fix:feat balance, hotspot control, bloat percentage. First concrete implementation of entropy measurement for code substrate. Current score: 73/100 (C). Key finding: **measurement-improvement bias** — when a broken metric is fixed, the improvement in the metric appears as improvement in the thing being measured. The health score jumped 37→69 in a single snapshot when `total_source_files` tracking was added (a pure measurement fix, not structural change). Systems tracking their own health need to distinguish "we got healthier" from "we got better at measuring." This has a knowledge-accretion analogue: if we track orphan rate but change the counting method, an apparent rate drop could be measurement improvement rather than actual synthesis.

8. **False ground truth** — Ground truth mechanisms that use unused feedback channels produce false positive signal. The `GroundTruthAdjustedRate()` formula blends self-reported success (70%) with rework-based ground truth (30%). But with 0 rework events across 817 completions, `reworkRate=0.0` is treated as "everything is correct" rather than "nobody uses this channel." The formula inflates self-reported success by +7.3 percentage points (75.7%→83.0%). Similarly, the merge rate metric reports 100% because all work commits directly to main with no PR workflow — the metric measures the same thing as completion. Three diagnostic tests for false ground truth: (1) Is the negative signal channel actually populated? (2) Does the metric measure something distinct from what it validates? (3) Would injecting a known-bad input be detected?

---

## Substrate Generalization

The physics hold for any shared mutable substrate where the four conditions are met (multiple agents, amnesiac, locally correct, no structural coordination).

| Substrate | Accretion | Attractors | Gates | Entropy Signal | Status |
|-----------|-----------|-----------|-------|----------------|--------|
| **Code** (orch-go) | daemon.go +892 lines, 6 cross-cutting dupes | pkg/ packages (structural coupling) | Pre-commit, spawn, completion, `go build` | Fix:feat ratio, hotspot analysis | **Confirmed** |
| **Knowledge** (.kb/) | 87.6% orphan investigations (52% model-era) | Models pull probes (structural coupling via directory) | None hard (all advisory) | Orphan rate, synthesis backlog | **Confirmed** |
| **Runtime behavior** (orch-go) | O(n) operations degrade silently as workspace/event/KB counts grow; 153K-line events.jsonl parsed on every `orch stats`; ~895 workspaces scanned cross-project on every `orch status` | None — no structural destination routes "scan" decisions | None — no metric tracks N values or alerts on latency thresholds | **None** — degradation invisible until user pain | **Confirmed** — 5 instances measured (2026-03-20 probe) |
| **OPSEC** (price-watch) | 5 detection signals over 5 months, invisible until catastrophic failure | Safe patterns contaminate unsafe contexts; middleware routes to proxy | Convention → middleware → startup → network isolation | **None pre-detection** — adversary IS the measurement | **Confirmed** — see `price-watch/.kb/models/opsec-substrate/model.md` |
| **Database schemas** | Column bloat, unused tables | Normalized entity structure | Migration validation, FK constraints | Dead columns, orphan tables | Hypothesized |
| **Config systems** | Setting sprawl | Config categories/namespaces | Schema validation | Unused settings, duplicate keys | Hypothesized |
| **API surfaces** | Endpoint bloat | Resource-oriented design | Contract testing, versioning | Deprecated endpoints, inconsistent naming | Hypothesized |
| **Documentation** | Doc sprawl, contradictions | Doc hierarchy/taxonomy | Link validation, freshness checks | Stale pages, orphaned docs | Hypothesized |
| **Prompt context** (CLAUDE.md, skills, SPAWN_CONTEXT.md) | 93→753 lines in 91 days (8x); 3 contradictory default-model claims; 92% passive reference content; 35% auto-committed | Sections act as attractors — "Event Tracking" pulls event rows, "Commands" pulls subcommands | **None** — no size limit, no staleness check, no consistency validation, no relevance filtering | Contradiction count, directive-to-reference ratio, section staleness | **Confirmed** — see probe 2026-03-20 |

**Minimal substrate properties (for accretion to produce degradation):**

1. **Mutable** — agents can add/modify content
2. **Shared** — multiple agents read/write
3. **Compositional** — individual contributions must compose non-trivially into a coherent whole (this is condition 4 in the five-condition formulation; additive/self-similar substrates like logs, sensor data, and coral reefs don't degrade because composition is trivial)
4. **Amnesiac** — no single agent has full context of all prior contributions

**The generalization of harness engineering:** Governing any shared mutable substrate where amnesiac agents contribute requires the same three mechanisms:

- **Attractors** — structural destinations that route contributions (packages, models, schemas, resource hierarchies)
- **Gates** — enforcement that blocks wrong paths (pre-commit, build, migration validation, contract testing)
- **Entropy measurement** — detection of when composition is failing (bloat, orphans, contradictions, staleness)

### Substrate vs Orchestration Separation (2026-03-09)

The investigation/probe/model cycle separates cleanly into **substrate** (the knowledge system) and **orchestration** (the infrastructure that drives the cycle at scale). Empirically tested by identifying what runs without the orch stack:

**Minimal substrate (5 components):**

| Component | Role | Irreducible? |
|-----------|------|-------------|
| Agent runtime (Claude Code) | Asks questions, tests, observes, writes | Yes — the cycle needs an agent |
| `kb` CLI | Context retrieval, artifact creation | Yes — stores conventions (could be replaced by raw file knowledge) |
| Git | Version control for `.kb/` artifacts | Yes — audit trail, collaboration |
| `.kb/` directory | The shared mutable substrate | Yes — where artifacts live |
| Investigation skill | Cycle conventions (probe mode, templates, merge protocol) | Yes — without it, agent doesn't know the cycle |

**Orchestration additions (valuable but not cycle-required):**

| Component | What It Adds | Why Not Minimal |
|-----------|-------------|-----------------|
| `orch spawn` | Pre-computed SPAWN_CONTEXT.md with extracted model sections | Agent can run `kb context` + read model files directly |
| beads (bd) | Issue tracking, phase reporting | Tracking infrastructure, not knowledge |
| daemon | Autonomous spawn of triage:ready | Scale mechanism, not cycle mechanism |
| Completion verification | Quality gates on deliverables | Reliability infrastructure |
| Dashboard/tmux/OpenCode | Visibility, multi-model, monitoring | Operational infrastructure |

**The context injection gap:** `kb context` returns model file paths but not extracted model sections (Summary, Critical Invariants, Why This Fails). orch's `extractModelSectionsForSpawn()` (in `kbcontext.go`, ~300 lines of the 1,496-line file) handles this extraction. Without it, the agent reads model files directly — 2-4 extra tool calls, fully functional. A `kb context --extract-models` flag would bridge this gap at the kb CLI level.

**Additional gaps for standalone use (2026-03-09 audit):** The context injection gap is necessary but not sufficient. A public-release audit of kb-cli found 6 additional gaps beyond `--extract-models`: (1) `kb ask` hard-depends on opencode binary with no fallback, (2) no LICENSE file despite MIT claim, (3) README documents <40% of commands, (4) CLAUDE.md is template-only with no project context, (5) 3 failing tests, (6) `groups.yaml` hardcoded to `~/.orch/` path. The coupling is tiered: hard-coupled commands (ask, link) break without orch infrastructure; soft-coupled commands (reflect --create-issue, context --siblings) degrade gracefully; core cycle commands (init, create, search, context local) are fully standalone.

**Implication for substrate independence:** The knowledge cycle doesn't depend on the orchestration substrate it was built in. A fresh repo with `kb init` + Claude Code + investigation skill can run the full investigation/probe/model cycle. This strengthens the substrate generalization claim — the physics are about the `.kb/` directory and its conventions, not the `orch` binary. However, the current kb-cli binary is not yet release-ready for external users — it requires ~7 focused changes to decouple from orch infrastructure.

---

## Critical Invariants

1. **Every convention without a gate will eventually be violated — in knowledge too.** The knowledge system now has one hard gate: the model-stub pre-commit gate (`orch precommit model-stub`) blocks committing model.md files with unfilled template placeholders. Prior to this gate, model quality was maintained behaviorally (all 37 existing models are fully filled), but the invariant predicts this would eventually fail. Other knowledge conventions (Prior Work tables, probe-to-model merge, dedup checking) remain soft and are violated at significant rates. This is the same invariant from harness-engineering, empirically confirmed in a second substrate.

2. **Models are the fundamental unit of knowledge organization.** Without models, knowledge is homeless. Pre-model era's 94.7% orphan rate vs model era's 52.0% demonstrates that models (and their probe system) provide the gravitational centers that organize investigative work product. The probe system is particularly effective — it structurally couples findings to models via directory placement.

3. **Attention-primed attractors become structurally-coupled via the probe system.** Code attractors work through compilation — imports route code mechanically. Knowledge attractors initially work through context injection — kb context frames agent investigation. But the probe system converts this to structural coupling: probes live in `.kb/models/{name}/probes/`, creating a directory-level connection. This explains the orphan rate drop from 94.7% (pre-probe) to 52.0% (post-probe). The probe system is the knowledge equivalent of Go's import system — it makes the connection structural rather than attention-dependent. Open question remains whether further gating could approach code's coupling rates.

4. **The orphan rate decomposes into six categories; the natural baseline is 40-50%.** The raw 85.5% (now 87.6% by strict measurement) is inflated by pre-model era artifacts (83% of corpus, 94.7% orphan rate). The model-era rate is **52.0%** — within the healthy range for an exploratory system. Orphans decompose into: implementation-as-investigation (~30-45%), audit/design (~25-33%), exploratory (~15-20%), genuinely lost (~20% of orphans, ~10% of total), negative results (~5-7%), and superseded (~3-5%). The actionable signal is the "genuinely lost" rate (~10% of investigations), not the raw orphan rate. Analogous to dead code: 5-15% dead code is healthy in code; 40-50% orphan rate is healthy in knowledge due to inherently higher exploration rates.

5. **Accretion, attractors, gates, and entropy are substrate-independent.** They emerge from system properties (multiple writers, no persistent memory, local correctness, non-trivial composition, no structural coordination), not substrate properties. Code, knowledge, OPSEC, and runtime behavior are four confirmed instances. Runtime behavior extends the evidence to a substrate where the *code* doesn't change — only the data it operates on grows. OPSEC extends to adversarial substrates where entropy is invisible to internal metrics. The same dynamics should appear in any substrate meeting the five conditions.

6. **The framework is falsifiable and conditionally predictive — diagnostic, not legislative.** Systematic search across 15+ candidate counterexamples (natural systems: ant colonies, coral reefs, immune systems; engineered: CRDTs, blockchains, event stores; human: Wikipedia, scientific literature, shared drives) found no clean counterexamples. Every system that resists accretion does so through coordination (explicit, substrate-embedded, or environmental). The framework makes testable predictions: (a) where accretion will concentrate (at coordination gaps), (b) what interventions will reduce it (gates at compositional boundaries), (c) that removing coordination will introduce accretion. It does NOT predict accretion form, rate, or threshold — these are substrate-specific. This is an Ostrom-scale diagnostic framework — structural conditions empirically derived from one system and stress-tested against 15+ — not a quantitative physical law. The name "knowledge accretion" is shorthand for the substrate-independent dynamics; the framing is institutional analysis, not natural science.

---

## Relationship to Existing Models

### Harness Engineering (Code Instance of Substrate Physics)

Knowledge accretion provides the theoretical grounding for why harness engineering generalizes. Harness engineering describes the *discipline* — making wrong paths mechanically impossible for AI agents. Knowledge accretion explains the *theory* — why the discipline works across substrates.

The mapping:
- Harness engineering's hard/soft taxonomy → applies to knowledge (all knowledge harness is currently soft)
- Harness engineering's "every convention without a gate" invariant → confirmed in knowledge substrate
- Harness engineering's accretion as thermodynamics → knowledge accretion follows the same thermodynamic pattern
- Harness engineering's compliance vs coordination distinction → knowledge failures are coordination failures (agents each investigate correctly but collectively produce 85.5% orphan rate)

Harness engineering is the code-specific instance of substrate governance. Knowledge accretion is the general theory.

### System Learning Loop (Knowledge Accretion in One Domain)

The system-learning-loop model describes knowledge accretion without naming it. Its gap→pattern→suggestion→improvement cycle maps directly to the attractor/gate framework:

| System-Learning-Loop | Knowledge Accretion | Code Physics |
|---------------------|-------------------|--------------|
| Gap recording | Entropy detection | Hotspot analysis |
| Pattern detection (threshold=3) | Attractor formation criterion | "3+ investigations → model" threshold |
| Suggestion generation | Gate recommendation | Pre-commit gate |
| Resolution tracking | Gate enforcement | Completion verification |
| 30-day retention | Entropy decay | Stale code pruning |

The system-learning-loop is a specialized instance operating on one substrate (context gaps). The same physics apply to all knowledge artifacts (investigations, models, decisions, quick entries) and to any shared mutable substrate.

RecurrenceThreshold=3 is an attractor formation criterion — when enough signal accumulates, a structural attractor forms. This is the same threshold as "3+ investigations trigger model creation."

### Skill Content Transfer (Soft Harness Dynamics)

The skill-content-transfer model's three-type vocabulary provides the mechanism for how knowledge attractors work:

- Knowledge attractors use **attention priming** (the "stance" type in skill-content-transfer terminology)
- Models injected via kb context prime agent attention toward the model's domain
- This is the same mechanism that makes stance items work in skills — changing what agents notice, not what agents do

The key finding from skill-content-transfer applies directly: **attention primers transfer; action directives do not.** This explains why knowledge models can pull investigations toward them (attention priming) but cannot enforce that findings get merged back (that would require an action directive, which needs a gate).

---

## Why This Fails

### Failure Mode 1: Knowledge Accretion Outpaces Synthesis

997 orphan investigations exist because investigations are cheaper to create than models. Each investigation takes one agent session; model creation requires cross-investigation synthesis, which requires reading and reconciling multiple sources. The economics favor production over synthesis.

### Failure Mode 2: Advisory Gates Are Non-Gates

Every knowledge gate is advisory. The "merge findings before completion" instruction in probe templates is a convention — contradicts verdicts historically accumulated (4 at baseline before batch resolution). Prior Work tables are a convention — 48% skip them. Without hard enforcement, conventions degrade under time pressure. This is the same dynamic as daemon.go growing past the stated 1,500-line convention in CLAUDE.md.

### Failure Mode 3: Attention-Primed Attractors Lose Under Pressure

When kb context injection includes a relevant model, agents are primed to engage with it. But when the task is urgent or the context window is crowded, the attention primer competes with task pressure. Code attractors (package structure) don't compete — they're enforced by the compiler. Knowledge attractors can be ignored.

### Failure Mode 4: Tooling Inverts the Importance Hierarchy

The model's critical invariant #2 states models are "the fundamental unit of knowledge organization." But the tooling inverts this: `kb create investigation` exists, `kb create model` and `kb create probe` do not. `kb init` creates `investigations/`, `decisions/`, `guides/` directories but not `models/`. The most important artifacts have the least tooling support, while the highest-volume artifact (investigations) has the most. This creates a structural bias toward investigation production over model synthesis — contributing to the orphan rate through tooling design, not just agent behavior. For external adoption, this gap is blocking: the first user cannot be expected to manually create directory structures and templates for the system's fundamental unit.

### Failure Mode 5: Creation/Removal Cost Asymmetry (Universal Ratchet)

Across every substrate studied, adding is cheaper than removing. Adding a file, column, flag, API endpoint, or investigation is a single-agent action. Removing one requires coordinating with unknown dependents. This asymmetry produces a ratchet: growth is easy, shrinkage requires coordination that amnesiac agents cannot provide. Empirically confirmed: 73% of feature flags never removed (FlagShark), 85% of shared drive data is dark/ROT (Veritas), 39% of organizations cannot inventory their APIs (APIsec). Even when coordination mechanisms exist, the ratchet persists because removal coordination is always more expensive than creation.

### Failure Mode 6: Anti-Accretion Mechanisms Create Second-Order Pathologies

Gates and coordination mechanisms can themselves accrete or create new problems: Wikipedia bots conflict with each other ("behave and interact unpredictably" — PLOS ONE). Stack Overflow's aggressive moderation drove away contributors. Scientific peer review creates publication bias. The cure for accretion, if applied without meta-coordination, can shift accretion to a different dimension (e.g., preventing spatial duplication while enabling temporal decay). This suggests a hierarchy: substrate coordination → meta-coordination of coordination mechanisms.

### Failure Mode 7: No Contradiction Resolution Mechanism

When a probe contradicts a model claim, the finding is recorded in the probe file but no mechanism forces the model to be updated. Contradicts verdicts historically accumulated (4 at baseline, batch-resolved 2026-03-09) and will recur without hard gates. Over time, models accumulate stale or contradicted claims that new agents receive as authoritative via kb context injection — creating a knowledge equivalent of stale cache invalidation.

### Failure Mode 8: False Ground Truth (Phase 2 Epistemic Failure)

The measurement infrastructure can produce metrics that look like independent validation but are structurally dependent on self-reported signals. Three instances observed (2026-03-18):

1. **Ground truth inflation:** `GroundTruthAdjustedRate()` inflates self-reported success by +7.3pp because rework_rate=0.0 (from 0 rework events across 817 completions) is treated as evidence of quality rather than evidence that the feedback channel is unused. The `hasReworkData` check triggers on completion volume (`TotalCompletions >= 10`), not on rework volume.

2. **Tautological validation:** Merge rate shows 100% because all work commits to main with no PR workflow. The metric measures the same thing as completion — it adds no independent signal.

3. **Self-referential detector outcomes:** Detector "useful rate" counts completed/resolved, but "completed" is self-reported agent success. The feedback loop is: detector creates issue → agent reports "done" → system counts as "useful."

This is a distinct failure mode from measurement-improvement bias (entropy metric #7). Measurement-improvement bias makes the system look better from better measurement. False ground truth makes the system look better from the absence of negative signal — the infrastructure of outcome verification exists but the negative signal channels are structurally unused.

**The phase transition:** Phase 1 failures (mechanical) are visible and self-correcting. Phase 2 failures (epistemic) are invisible to Phase 1 infrastructure because that infrastructure measures existence and syntax, not correctness and impact. The system must evolve its measurement surface from "does it exist/compile/run?" to "did it actually improve anything?"

---

## Adoption Sequencing

The ideal domain for the physics (institutional amnesia in regulated/complex organizations) is different from the ideal first user of the system. All three candidate profiles — solo researcher with AI agents, R&D lab, startup with turnover — meet the four conditions for substrate dynamics. The differentiator is adoption friction and time to first value, not whether the physics apply.

**First user: Solo Technical Researcher (STR).** Developer or researcher working alone on a complex project with AI agents (Claude Code, Cursor). Lowest friction: already in Git, already uses CLI, already has amnesiac AI agents. Fastest time to value: personal pain (forgetting own decisions) is immediate, not contingent on team events. No organizational buy-in required. This replicates Dylan's archetype with the system pre-built.

**Adoption sequence (empirically validated across ADRs, Zettelkasten/Obsidian, ELNs):**
1. Individual practitioner solves their own problem (solo researcher adopts kb)
2. Nearby collaborators see value through exposure (lab mates, team members see models preventing re-investigation)
3. Champions carry practice to new contexts (researcher moves to new lab/company)
4. Institutional endorsement codifies the practice (R&D lab adopts kb as standard)

Evidence: ADRs took 7 years from Nygard's projects (2011) to ThoughtWorks "Adopt" (2018) to UK Government mandate (2025). Benchling gave product free to 200K+ academics who carried it to industry — 1 in 4 biotech IPOs (2020-22) built on Benchling. Obsidian's 1.5M MAU are predominantly individual; team features were added later. ELN adoption requires supervisor support (Southampton study) but initial spark comes from individual practitioners.

**R&D labs are the second user, not the first.** Their pain is strongest (postdoc turnover, unreproduced experiments, lost protocols), but adoption friction is highest (IT policies, non-Git workflows, PI buy-in required, training needed). The path to lab adoption runs through a single researcher champion.

### Distribution Channels for First STR User

The STR community is large and discoverable (probe 2026-03-11). ~100k+ users fit the STR profile across concentrated online communities:

| Channel | Size/Reach | Signal Quality | Effort |
|---------|-----------|----------------|--------|
| r/ClaudeCode | 96k members, 4.2k weekly contributors | Very high — exact target audience | Low-medium (2-3 weeks value-first engagement) |
| r/ClaudeAI | ~300k members | High — Claude Code crossover | Same approach |
| awesome-claude-code (GitHub) | 21.6k stars | Very high — curated, passive exposure | Very low (1 PR) |
| Show HN | 10k-30k visitors per front page hit | High — right audience | Low (1 post + comment engagement) |
| Anthropic Discord | 68.5k members | Medium — ephemeral format | Medium (weeks of helping first) |

**Demand signal is active, not latent.** GitHub Issue #28196 on anthropics/claude-code requests "Built-in Personal Knowledge Base with Semantic RAG." Cursor Forum has multiple threads requesting persistent agent memory. Stack Overflow 2025 survey: developers spend 23% of AI interaction time re-providing context. "Stop Claude Code from forgetting everything" (Show HN) got 202 points and 226 comments.

**Positioning principle: pain-point framing >> feature-list framing.** Show HN data shows ~100:1 engagement ratio between pain-point titles ("Stop Claude Code from forgetting") vs. architecture descriptions ("multi-agent orchestrator"). kb-cli should be positioned as "stop re-investigating solved problems" not "structured knowledge management CLI."

**The "orchestrator" space is saturated; "knowledge management methodology" is not.** 10+ Claude Code orchestrator tools posted Jan-Mar 2026, most getting 1-5 HN points. But no tool positions itself as a knowledge management methodology (investigation → probe → model cycle). kb-cli's angle is methodology, not infrastructure — distinct from MCP-based memory solutions.

**Conversion funnel reality:** HN front page → ~5% star rate → <10% install rate → <10% retain rate. ~0.1% of top-of-funnel become regular users. A front-page HN post might produce 5-15 ongoing users. Sustained multi-channel presence over weeks is more reliable than a single viral moment. Aider case study: 3 HN points at launch → 41,600 stars through 2+ years of sustained iteration — persistence beats virality.

---

## Open Questions

1. **~~Is the 85.5% orphan rate a problem or a natural property?~~ ANSWERED:** The rate decomposes into six categories (see probe 2026-03-09-natural-orphan-baseline-categorization). Natural baseline is 40-50%. The raw 85.5% is inflated by pre-model era (83% of corpus). Model-era rate is 52% — healthy. "Genuinely lost" knowledge is ~10% of investigations — the real actionable metric. Dead code analogy confirmed: 5-15% dead code = healthy in code; 40-50% orphan rate = healthy in knowledge.

2. **What would knowledge pre-commit hooks look like?** Code pre-commit hooks validate syntax, lint, and compilation. Knowledge pre-commit hooks would need to check: does this investigation cite a Prior Work table? Does this probe reference its model? Does this model.md contradict itself? The technical challenge: knowledge validation is semantic, not syntactic.

3. **Are attention-primed attractors weaker than structural attractors, or just ungated?** Partially answered: the probe system converts attention priming to structural coupling (probes live in model directories), dropping the orphan rate from 94.7% to 52%. Remaining question: could further gating (e.g., investigations require `--model` flag) push the rate below 40%? Or is 40-50% the floor for exploratory systems regardless of coupling?

4. **What's the right threshold for claims-per-model bloat?** Code has lines-per-file thresholds (800 warning, 1,500 critical). What's the equivalent for model claims? When does a model accumulate enough claims that it needs splitting?

5. **Does adding hard knowledge gates reduce entropy or add ceremony?** Code gates (pre-commit, spawn, completion) have demonstrated ROI — they prevent measurable damage. But knowledge creation is more exploratory than code writing. Would gates on investigation creation slow down legitimate exploration? The analogy: mandatory code review slows velocity but catches coordination failures. Mandatory model-linking might slow exploration but catch orphan accumulation.

6. **Can knowledge attractors be structurally coupled?** Code attractors work through imports (structural coupling). Could knowledge attractors work through tooling — e.g., `kb create investigation` requiring a `--model` flag? This would convert attention priming into structural coupling without requiring semantic validation.

7. **What is the minimum time/volume before the physics become visible to a new user?** Dylan's system shows the physics after 1,166 investigations over months. A solo researcher generating 3-5 investigations/week might not see the dynamics (orphan accumulation, attractor pull, synthesis opportunities) for weeks. The "magic moment" — when the system's compounding value becomes obvious — may need to be accelerated through seeded examples or guided onboarding for first-time users.

8. **Can strongly-typed languages be modeled as substrate-embedded coordination?** If type systems are coordination embedded in the substrate (like CRDTs for data), then strongly-typed languages should exhibit less accretion than weakly-typed ones — a testable prediction. This would connect the coordination taxonomy to practical programming language design.

9. **~~Is the theory falsifiable?~~ ANSWERED (2026-03-10):** Yes. Systematic search across 15+ counterexamples found none that survive rigorous condition checking. The theory makes testable predictions about where accretion concentrates and what interventions reduce it. Refinement: the four conditions were expanded to five with "non-trivial composition" made explicit (preventing over-prediction in additive substrates). The theory is conditionally predictive — a qualitative causal framework, not a quantitative law.

10. **How do you build metrics that detect false coherence when the system itself produces the metrics?** Three approaches identified but not tested: (a) external ground truth injection — metrics the measured system cannot produce (e.g., human quality sampling, PR review); (b) negative channel health monitoring — flag when feedback channels (rework, abandonment) have zero activity alongside high completion volume; (c) metric mutation testing — inject known-bad completions and verify the measurement system detects them. The core constraint: any metric produced by the system being measured is subject to the same circular validation that produced false ground truth.

---

## Evolution

**2025-12-25:** System-learning-loop model created, describing gap→pattern→suggestion→improvement as a closed feedback loop. Proto-knowledge-accretion without naming it.

**2026-02-25:** Entropy-spiral model created, documenting how locally correct changes compose into globally incoherent systems in code substrates. Three spirals, 1,625 lost commits. Control plane immutability principle established.

**2026-03-07 to 2026-03-08:** Harness engineering model created, synthesizing accretion as thermodynamics, hard/soft harness taxonomy, compliance vs coordination failure. Established that codebase architecture is governance infrastructure.

**2026-03-09:** Knowledge accretion probe (orch-go-8m7w9) empirically measured knowledge dynamics: 85.5% orphan rate, three model behaviors (attractor/capstone/dormant), zero hard gates, 4 unmerged contradicts. Confirmed substrate independence of the physics. This model created to formalize the framework.

**2026-03-09:** Natural orphan baseline probe (orch-go-80rg8) decomposed the orphan rate into six categories and established natural baseline of 40-50%. Discovered pre-model era (83% of corpus) inflates the aggregate rate; model-era rate is 52%. Probes confirmed as the structural fix converting attention-primed attractors to structurally-coupled attractors. Open question #1 answered.

**2026-03-09:** First external user profile probe (orch-go-j2ziz) established adoption sequencing: Solo Technical Researcher (STR) is the first user, not R&D labs or startups. Bottom-up adoption pattern validated across ADRs (7 years individual→industry), Obsidian (1.5M individual MAU→team features later), Benchling (200K academics→1/4 biotech IPOs). Tooling gap identified: `kb create model`/`kb create probe` don't exist, `kb init` doesn't create models/ directory — the "fundamental unit" has the least tooling support. Added Failure Mode 4 (tooling inverts importance hierarchy), Adoption Sequencing section, and open question #7 (minimum time to visible physics).

**2026-03-10:** OPSEC substrate model created (`price-watch/.kb/models/opsec-substrate/model.md`). Third confirmed substrate instance. Extends the theory to adversarial substrates with three distinctive properties: binary catastrophic failure (vs gradual degradation in code/knowledge), adversarial entropy (external actor exploits signals, making "clean up later" impossible), and multiplicative signal composition (weak signals confirm each other, unlike linear code accretion or additive knowledge accretion). Evidence: OshCut detection incident — 5 signals accumulated over 5 months while all engineering metrics were green, composed into definitive attribution in one day. Gate escalation followed the same soft → hard → structural sequence as code. Updated substrate generalization table from 2 confirmed + 4 hypothesized to 3 confirmed + 4 hypothesized.

**2026-03-10:** Health score structural improvement validation (orch-go-y642j). Decomposed the 37→73 score improvement: 90% from calibration (tracking gates, scaling thresholds, adding total_source_files), 10% from actual extractions. The 10 target files were extracted (avg 1139→410 lines, 64% reduction), but the old formula was too broken to register this. New finding: **measurement-improvement bias** — fixing a broken metric appears as improvement in the measured thing. Score jumped 37→69 in one snapshot when total_source_files was added. Added entropy metric #7 (composite health score) with this caveat. Hotspot dimension (1.9/20) identified as the honest remaining structural debt signal.

**2026-03-09:** Public release readiness audit (orch-go-8zp45) of kb-cli codebase. **Contradicted** the "single tooling gap" claim from minimal-substrate probe — found 6 additional gaps beyond `--extract-models`. **Confirmed** core cycle commands are fully standalone. **Extended** substrate/orchestration separation with three-tier coupling taxonomy: hard (ask, link — break without orch), soft (reflect --create-issue, context --siblings — degrade gracefully), none (init, create, search, list — fully standalone). Code health: 57.6% test coverage, 3 failing tests, 3 direct dependencies, no go vet errors. Key blockers: missing LICENSE file, incomplete README, hardcoded orch paths.

**2026-03-10:** Ostrom framing thread resolved (orch-go-3hdyt). Added Theory Type section establishing knowledge accretion as a diagnostic framework (Ostrom-scale institutional analysis), not a natural law. Updated Summary and Critical Invariant #6 to reflect diagnostic framing. Analogues: Ostrom's commons governance design principles, Conway's Law, Brooks's Law. This shapes publication tone: "here's a predictive pattern I found and tried to break" rather than "here's a new field of science."

**2026-03-10:** Falsifiability probe (orch-go-dqv2o). Systematic counterexample search across 15+ systems in three domains (natural, engineered, human). No clean counterexamples found. Three key extensions: (1) Added condition 5 — "non-trivial composition" — to prevent over-prediction in additive substrates (logs, sensor data, coral reefs) that meet conditions 1-4 but don't degrade because composition is trivial. (2) Coordination taxonomy: explicit (type systems), substrate-embedded (CRDTs), environmental (stigmergy). Digital substrates lack environmental coordination, explaining why they require engineered gates. (3) Continuous risk formulation: `accretion_risk = f(amnesia × complexity / coordination)` — more precise than binary conditions. Theory verdict: conditionally predictive, publishable with composition refinement.

**2026-03-11:** Distribution channels probe. Extended Adoption Sequencing section with concrete channel data: r/ClaudeCode (96k members, 4.2k weekly contributors), awesome-claude-code lists (21.6k stars), Show HN as top-3 channels for reaching STR users. Active demand confirmed via GitHub Issue #28196 ("Built-in Personal Knowledge Base with Semantic RAG"). Key positioning insight: pain-point framing outperforms feature-list framing by ~100:1 on HN. Claude Code orchestrator space oversaturated (10+ tools, 1-5 HN pts each) but knowledge management methodology is undersaturated. Conversion funnel: ~0.1% top-of-funnel → regular user. Aider case validates persistence over virality (3 HN pts → 41.6k stars over 2 years). Concrete 4-phase launch sequence defined: preparation → community presence → passive distribution → active launch.

**2026-03-18:** Phase transition probe (orch-go-58923). Named the mechanical→epistemic phase transition and identified three instances of false ground truth in the measurement infrastructure. Ground truth adjustment inflates self-reported success by +7.3pp because rework_rate=0.0 (0 reworks / 817 completions) is treated as evidence rather than absent signal. Merge rate is tautological (100% because single-branch, no PR flow). Detector outcome "useful rate" is circular (self-reported completion counted as "useful"). Added Failure Mode 8 (false ground truth), entropy metric #8 (false ground truth detection), and open question #10 (metrics that detect false coherence when the system produces the metrics). Key insight: Phase 1 infrastructure detects broken builds and missing files; Phase 2 failures (false confidence, inflated metrics) are invisible to the same infrastructure.

**2026-03-10:** Blog post uncontaminated claim review (orch-go-2bdvb). Read all three blog drafts (harness engineering, knowledge accretion, coordination demo) as an external reader with no model framing. Found systematic gap between the model's self-assessment (already corrected to "WORKING HYPOTHESIS" / "overclaimed") and the publication language (still claims universality, novelty, and validated theory). 8 validation assumptions, 4 novelty assumptions, 12 overclaimed language instances identified with specific line references and recommended corrections. Key finding: hedging exists but is structurally misplaced (buried at end, not inline with claims). Publication gate assessment: 2 of 3 required artifacts exist (red-team memo + this claim-label pass); claim ledger still needed. Added Publication Readiness section to model.

**2026-03-20:** Intervention effectiveness audit. Systematically audited all 31 interventions proposed in this model against codebase implementation and measurement data. **Contradicted** the Gate Deficit table's "zero hard knowledge gates" claim — probe-to-model merge (`pkg/verify/probe_model_merge.go`) is a blocking completion gate, and model-stub precommit was already acknowledged elsewhere but missing from the table. **Extended** the model with an intervention effectiveness hierarchy: structural attractors > signaling (event→daemon) > blocking gates (100% bypass) > advisory > metrics-only. Only 4 of 31 interventions (13%) have evidence of reducing their target. Updated Gate Deficit table and added Section 3a (Intervention Effectiveness). Documented gate lifecycle arc: designed→measured→found inert/high-FP→downgraded/removed.

**2026-03-20:** Prompt context substrate probe. Extended the substrate generalization table with **prompt context** (CLAUDE.md, skill files, SPAWN_CONTEXT.md) as a confirmed accreting substrate. CLAUDE.md grew 93→753 lines in 91 days (8x), with 62% of commits by agents and 35% by automated drift-sync processes. Found 3 contradictory claims about the default model (line 267: Gemini, line 561: Opus, actual code: Sonnet) — textbook accretion from amnesiac agents updating different sections independently. Only 8.2% of CLAUDE.md content is directive (shapes agent behavior); 91.8% is passive reference consuming ~56,500 tokens/week across all agents. Unique properties vs other substrates: total read amplification (all agents read all content regardless of relevance), signal-to-noise degradation with growth, ratchet behavior after pruning events (two prunings both followed by immediate re-growth), and multiplicative cost (each line × all agents × all sessions). SPAWN_CONTEXT.md doesn't accrete per-file (regenerated fresh) but grows +36% over time because the KB it draws from grows. Worker-base skill (429 lines, injected into all workers) is a secondary accreting substrate with same properties. Total injected context per agent: ~2,400 lines before reading any task code.

---

## Publication Readiness (Blog Post Claim Audit)

**Status: NOT READY.** Uncontaminated review of all three blog drafts (harness engineering, knowledge accretion, coordination demo) found overclaimed language that contradicts the model-level corrections already applied. The model says "WORKING HYPOTHESIS" and "Status: overclaimed" — but the publications still use language that assumes validation and novelty.

**Key gaps between model self-assessment and publication language:**

1. **"Physics" in title and body.** The model's Theory Type section says "the name implies a level of generality and novelty that independent review did not support." The knowledge-accretion draft still uses the title "Knowledge Accretion" and the phrase "the physics appear to be substrate-independent" (appears in 2 of 3 posts).

2. **Universality claims from N=2.** The model's Core Claim says "Whether the pattern generalizes beyond this one system is untested." The publications say "any substrate," "regardless of what the substrate is made of," and "applicable to any substrate where amnesiac agents contribute."

3. **Internal validation presented as external.** The model says "not externally validated" and flags closed-loop risk. The knowledge-accretion draft says "kept surviving every attempt to break it" and "No clean counterexample survived" without disclosing that all testing was by agents inside the framework.

4. **The 265-trial methodology gap.** The harness engineering post cites "265 contrastive trials across 7 agent skills" as evidence for soft instruction failure. No methodology is provided — what constitutes a "trial," what was measured, how controls were defined. The number is unverifiable by readers.

5. **"Better models make it worse" is under-evidenced.** The merge experiment shows equal conflict rates for Haiku and Opus (both 100%), not worse ones. The daemon.go data doesn't isolate model capability as a variable. The claim should be "model improvement doesn't fix coordination" not "model improvement makes coordination worse."

6. **Ostrom comparison without evidence-base acknowledgment.** Ostrom studied hundreds of commons across dozens of countries over decades. This is one system, one operator, three months. The comparison is apt in type but the evidence gap is not acknowledged.

7. **Hedging is structurally misplaced.** The knowledge-accretion post buries "One system, one operator" in an "Honest Gaps" section at the end. Readers absorb confident claims in the body and skim caveats at the end.

**Specific language requiring change (by post):**

| Post | Current Language | Recommended Change |
|------|-----------------|-------------------|
| Knowledge Accretion (title) | "Knowledge Accretion" | Consider "Coordination Patterns" or scope the name explicitly |
| Knowledge Accretion (L3) | "applicable to any substrate" | "observed in two substrates in one system" |
| Knowledge Accretion (L31) | "kept surviving every attempt to break it" | "survived internal testing; not yet externally validated" |
| Knowledge Accretion (L71) | "any shared mutable substrate" | "the substrates we've tested" |
| Knowledge Accretion (L89) | "No clean counterexample survived" | Add: "though all testing was by agents within the same framework" |
| Knowledge Accretion (L143) | "Greater capability created greater divergence" | "In this trial, ..." (N=1) |
| Harness Engineering (L15) | "faster, more capable agents accrete more code per session" | "model improvement doesn't reduce accretion" |
| Harness Engineering (L195) | "language-independent" | "tested in one additional language" |
| Harness Engineering (L217) | "absent from the published literature" | "not prominent in the AI agent literature we've reviewed" |
| Harness Engineering (L297) | "The physics appear to be substrate-independent" | "We observe similar patterns in both code and knowledge substrates" |
| Coordination Demo (L214) | "The physics appear to be substrate-independent" | Same change |
| Both posts | "thermodynamic tendency" | "tendency" (remove physics metaphor or explicitly label as metaphor) |

**Publication gate assessment:** The publication gate requires claim ledger, red-team memo, and claim-label pass. This probe serves as the claim-label pass. The red-team memo exists in the Codex review (`.kb/threads/2026-03-10-closed-loop-risk-ai-agents.md`). A claim ledger has not been created. **2 of 3 gate artifacts exist; publications require language corrections above before gate is satisfied.**

---

## Observability

**What you can measure today:**

```bash
# Orphan rate (approximate — grep-based)
total_inv=$(find .kb/investigations -name "*.md" | wc -l)
connected=$(grep -rl "\.kb/investigations/" .kb/models/*/model.md | wc -l)
echo "Orphan rate: $(( (total_inv - connected) * 100 / total_inv ))%"

# Contradicts backlog
grep -rl "contradicts" .kb/models/*/probes/*.md | wc -l

# Model probe freshness
for model in .kb/models/*/; do
  latest=$(ls -t "$model"probes/*.md 2>/dev/null | head -1)
  echo "$model: $latest"
done

# Synthesis backlog
kb reflect --type synthesis

# Stale models
kb reflect --type stale
```

**What you cannot measure today:**
- Claims per model (no automated claim extraction)
- Quick entry duplication rate (no dedup tooling)
- Semantic overlap across investigations (requires NLP/embedding)
- Knowledge attractor strength (no reference tracking beyond manual grep)

---

## References

**Primary Evidence:**
- `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-accretion-accretion-attractor-gate-dynamics.md` — Full empirical measurement (1,166 investigations, 32 models, 187 probes)
- `.kb/models/knowledge-accretion/probes/2026-03-09-probe-natural-orphan-baseline-categorization.md` — Orphan taxonomy, era-adjusted rates, natural baseline (40-50%), probe displacement finding
- `.kb/models/knowledge-accretion/probes/2026-03-09-probe-minimal-kb-substrate-cycle-dependencies.md` — Minimal substrate identification: 5 components (agent + kb + git + .kb/ + skill), substrate/orchestration separation confirmed, context injection gap identified
- `.kb/models/knowledge-accretion/probes/2026-03-09-probe-first-external-user-profile-analysis.md` — First user is Solo Technical Researcher (STR), not R&D lab. Adoption sequencing validated across 4 analogous tools. Tooling gap: `kb create model`/`kb create probe` missing, `kb init` doesn't create models/
- `.kb/models/knowledge-accretion/probes/2026-03-09-probe-kb-cli-public-release-readiness-audit.md` — kb-cli public release audit: core cycle standalone, 6 gaps beyond --extract-models, three-tier coupling taxonomy (hard/soft/none), 7 blocking changes for v0.1
- `.kb/models/knowledge-accretion/probes/2026-03-10-probe-health-score-structural-improvement-validation.md` — Health score 37→73 decomposition: 90% calibration, 10% extraction. Measurement-improvement bias finding. Hotspot dimension (1.9/20) is honest remaining signal.
- `.kb/models/knowledge-accretion/probes/2026-03-10-probe-falsifiability-counterexamples.md` — Falsifiability probe: 15+ counterexamples tested, none survive. Fifth condition (non-trivial composition) identified. Coordination taxonomy (explicit/substrate-embedded/environmental). Theory is conditionally predictive.
- `.kb/models/knowledge-accretion/probes/2026-03-10-probe-blog-post-uncontaminated-claim-review.md` — Publication claim audit: 8 validation assumptions, 4 novelty assumptions, 12 overclaimed language instances across 3 blog drafts. Publications still use language the model has already corrected internally.
- `.kb/models/knowledge-accretion/probes/2026-03-11-probe-empty-model-stub-creation-vectors.md` — Model stub creation vector analysis: 3 creation paths identified (kb create model, kb create model --from, agent direct-write). Zero current empty stubs exist (behavioral compliance held). Model-stub pre-commit gate implemented to convert from behavioral to architectural enforcement.
- `.kb/models/knowledge-accretion/probes/2026-03-11-probe-distribution-channels-solo-dev-cli-tools.md` — Distribution channel analysis for first STR user: r/ClaudeCode (96k), awesome-claude-code (21.6k stars), Show HN as top channels. Active demand confirmed via GitHub Issue #28196. Pain-point framing >> feature-list framing (100:1). Orchestrator space saturated; knowledge methodology space undersaturated.
- `.kb/models/knowledge-accretion/probes/2026-03-20-probe-behavioral-accretion-runtime-cost-grows-silently.md` — Runtime behavior as 4th confirmed accretion substrate. 5 instances: events.jsonl unbounded parse (153K lines, 0.56s, projected 17s at 1yr), cross-project workspace scan (~895 workspaces), manifest lookup on every status, KB filepath.Walk (1700+ files), daemon compound effect (23+ tasks × growing N). Code didn't change — only the world it operates on grew.
- `.kb/models/knowledge-accretion/probes/2026-03-20-probe-governance-infrastructure-self-accretion.md` — KA-10 confirmed empirically in orch-go: 35% of 138K non-test Go lines are governance/measurement/KB management. Daemon is 85% governance (22/26 periodic tasks). Governance share accelerated 18%→23% in March, growing faster than core. The accretion management infrastructure itself satisfies all five accretion conditions.

**Related Models:**
- `.kb/models/harness-engineering/model.md` — Code instance of substrate physics, hard/soft harness taxonomy
- `.kb/models/system-learning-loop/model.md` — Knowledge accretion in context-gap domain without naming it
- `.kb/models/skill-content-transfer/model.md` — Attention primers vs action directives, mechanism behind knowledge attractors
- `.kb/models/entropy-spiral/model.md` — Feedback loops in code substrate, control plane immutability

**Knowledge Accretion Assessment (in system-learning-loop model):**
- Section "Knowledge Accretion Assessment (2026-03-09)" in system-learning-loop/model.md documents the mapping and empirical evidence that prompted this model's creation.

**Threads:**
- `.kb/threads/2026-03-09-knowledge-accretion-does-knowledge-have.md` — Initial question formulation
- `.kb/threads/2026-03-10-kind-theory-this-ostrom-scale.md` — Theory type resolved: diagnostic framework (Ostrom-scale), not natural law

## Auto-Linked Investigations

- .kb/investigations/2026-03-10-inv-probe-falsifiability-theory-called-knowledge.md
