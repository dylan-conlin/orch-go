# Model: Knowledge Physics

**Domain:** Multi-Agent Knowledge Systems / Substrate Governance / Entropy Management
**Last Updated:** 2026-03-10
**Synthesized From:**
- `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-physics-accretion-attractor-gate-dynamics.md` — Empirical measurement across 1,166 investigations, 32 models, 187 probes
- `.kb/models/harness-engineering/model.md` — Hard/soft harness taxonomy, accretion as thermodynamics, compliance vs coordination failure
- `.kb/models/system-learning-loop/model.md` — Gap→pattern→suggestion→improvement as proto-knowledge-physics
- `.kb/models/skill-content-transfer/model.md` — Attention primers vs action directives, three-type vocabulary
- `.kb/models/entropy-spiral/model.md` — Feedback loops, control plane immutability, 1,625 lost commits

---

## Summary (30 seconds)

Knowledge exhibits the same physics as code when multiple amnesiac agents contribute to a shared mutable substrate. Accretion, attractors, gates, and entropy are substrate-independent dynamics — they emerge from system properties (multiple writers, no persistent memory, local correctness, non-trivial compositional requirements, absent coordination), not from properties of the substrate itself. Empirical measurement across orch-go's knowledge corpus: 85.5% orphan rate (997/1,166 investigations unconnected to any model), three model behaviors (attractor/capstone/dormant), zero hard knowledge gates (all transitions advisory), and 4 unmerged contradiction verdicts. Code, knowledge, and OPSEC are three confirmed instances of the same physics. Hypothesized additional substrates include database schemas, config systems, API surfaces, and documentation. The generalization: harness engineering is not code governance — it is substrate governance. Any shared mutable substrate where amnesiac agents contribute requires attractors (structural destinations), gates (enforcement blocking wrong paths), and entropy measurement (detection of compositional failure). OPSEC extends the theory to adversarial substrates, revealing that entropy can be invisible to internal measurement (only the adversary observes it), failure can be binary/catastrophic (no gradual degradation), and signal composition can be multiplicative rather than linear or additive.

---

## Core Claim

Knowledge exhibits the same physics as code when multiple amnesiac agents contribute to a shared mutable substrate.

The dynamics — accretion, attractors, gates, entropy — are properties of the system configuration, not the substrate. They emerge whenever five conditions hold:

1. **Multiple agents write** to the substrate
2. **Agents are amnesiac** — no cross-session memory
3. **Contributions are locally correct** — each passes local validation
4. **Contributions must compose non-trivially** — coherence between contributions is required and not automatic
5. **No structural coordination mechanism exists** — locally correct + locally correct ≠ globally correct

Conditions 1-3 are context-setters (common in most modern systems). Condition 4 distinguishes compositional substrates (code, knowledge bases, schemas) from additive substrates (append-only logs, sensor data, votes) where contributions are independent and cannot compose incorrectly. Condition 5 is the lever — the presence or absence of coordination mechanisms determines whether accretion occurs or is managed.

**Continuous formulation:** These conditions exist on spectrums, not as binary values. The theory is most precisely expressed as a risk model: `accretion_risk = f(amnesia_level × compositional_complexity / coordination_strength)`. This explains partial accretion in partially-coordinated systems and avoids debates about where binary thresholds fall.

Code was the first substrate where we observed these dynamics (daemon.go +892 lines from 30 correct commits). Knowledge is the second (85.5% orphan rate, 997 unconnected investigations). OPSEC is the third (5 detection signals accumulating invisibly over 5 months). The physics are identical; only the substrate-specific manifestations differ.

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

**Orphan taxonomy (from 35-file sample):**

| Category | Rate | Natural? |
|----------|------|----------|
| Implementation-as-investigation | 30-45% | Yes — wrong skill routing |
| Audit/design | 25-33% | Yes — point-in-time snapshots |
| Exploratory | 15-20% | Yes — one-off questions |
| Genuinely lost | ~20% of orphans | **No — knowledge loss** |
| Negative results | 5-7% | Yes — valuable to record |
| Superseded | 3-5% | Yes — natural lifecycle |

**Natural baseline: 40-50% orphan rate is healthy.** The actionable signal is the "genuinely lost" rate (~10% of total investigations), not the raw orphan rate.

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

Every knowledge transition is either ungated or advisory-only:

| Transition | Status | Gate Mechanism |
|------------|--------|---------------|
| Investigation → model | **UNGATED** | No automated model update when investigation contradicts model |
| Probe → model update | **UNGATED** | Skill template says "merge findings before completion" but no tooling validates. Historically 4 probes had unmerged "contradicts" verdicts (resolved 2026-03-09); pattern recurrence expected without hard gates |
| Quick entry → decision | **UNGATED** | No dedup checking against existing entries or decisions |
| Decision → implementation | **UNGATED** (1/56 exception) | Only 1 of 56 decisions has a `kb agreements` check |
| Investigation prior work | **SOFT** (52% adoption) | Template includes it, 48% of investigations skip it |
| Knowledge consistency at commit | **UNGATED** | Pre-commit hooks only run on `*.go` files, not `.kb/` files |

**The parallel to code is exact.** The harness-engineering model documented that "every convention without a gate will eventually be violated." The knowledge system proves this claim across every transition:

- Probe-to-model merge is a convention → contradicts verdicts historically accumulated before resolution (4 at baseline, resolved 2026-03-09)
- Prior Work tables are a convention → 48% of investigations skip them
- Quick entry uniqueness is a convention → confirmed duplicates exist
- Decision enforcement is a convention → 1.8% enforcement rate

Code has pre-commit hooks, spawn gates, and completion verification. Knowledge has zero hard gates. The knowledge substrate operates entirely on soft harness, which the harness-engineering model has shown degrades under pressure.

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

7. **Composite health score** — `orch health` implements a 5-dimension 0-100 score: gate coverage, accretion control, fix:feat balance, hotspot control, bloat percentage. First concrete implementation of entropy measurement for code substrate. Current score: 73/100 (C). Key finding: **measurement-improvement bias** — when a broken metric is fixed, the improvement in the metric appears as improvement in the thing being measured. The health score jumped 37→69 in a single snapshot when `total_source_files` tracking was added (a pure measurement fix, not structural change). Systems tracking their own health need to distinguish "we got healthier" from "we got better at measuring." This has a knowledge-physics analogue: if we track orphan rate but change the counting method, an apparent rate drop could be measurement improvement rather than actual synthesis.

---

## Substrate Generalization

The physics hold for any shared mutable substrate where the four conditions are met (multiple agents, amnesiac, locally correct, no structural coordination).

| Substrate | Accretion | Attractors | Gates | Entropy Signal | Status |
|-----------|-----------|-----------|-------|----------------|--------|
| **Code** (orch-go) | daemon.go +892 lines, 6 cross-cutting dupes | pkg/ packages (structural coupling) | Pre-commit, spawn, completion, `go build` | Fix:feat ratio, hotspot analysis | **Confirmed** |
| **Knowledge** (.kb/) | 87.6% orphan investigations (52% model-era) | Models pull probes (structural coupling via directory) | None hard (all advisory) | Orphan rate, synthesis backlog | **Confirmed** |
| **OPSEC** (price-watch) | 5 detection signals over 5 months, invisible until catastrophic failure | Safe patterns contaminate unsafe contexts; middleware routes to proxy | Convention → middleware → startup → network isolation | **None pre-detection** — adversary IS the measurement | **Confirmed** — see `price-watch/.kb/models/opsec-substrate/model.md` |
| **Database schemas** | Column bloat, unused tables | Normalized entity structure | Migration validation, FK constraints | Dead columns, orphan tables | Hypothesized |
| **Config systems** | Setting sprawl | Config categories/namespaces | Schema validation | Unused settings, duplicate keys | Hypothesized |
| **API surfaces** | Endpoint bloat | Resource-oriented design | Contract testing, versioning | Deprecated endpoints, inconsistent naming | Hypothesized |
| **Documentation** | Doc sprawl, contradictions | Doc hierarchy/taxonomy | Link validation, freshness checks | Stale pages, orphaned docs | Hypothesized |

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

1. **Every convention without a gate will eventually be violated — in knowledge too.** The knowledge system has zero hard gates. Every knowledge convention (Prior Work tables, probe-to-model merge, dedup checking) is violated at significant rates. This is the same invariant from harness-engineering, empirically confirmed in a second substrate.

2. **Models are the fundamental unit of knowledge organization.** Without models, knowledge is homeless. Pre-model era's 94.7% orphan rate vs model era's 52.0% demonstrates that models (and their probe system) provide the gravitational centers that organize investigative work product. The probe system is particularly effective — it structurally couples findings to models via directory placement.

3. **Attention-primed attractors become structurally-coupled via the probe system.** Code attractors work through compilation — imports route code mechanically. Knowledge attractors initially work through context injection — kb context frames agent investigation. But the probe system converts this to structural coupling: probes live in `.kb/models/{name}/probes/`, creating a directory-level connection. This explains the orphan rate drop from 94.7% (pre-probe) to 52.0% (post-probe). The probe system is the knowledge equivalent of Go's import system — it makes the connection structural rather than attention-dependent. Open question remains whether further gating could approach code's coupling rates.

4. **The orphan rate decomposes into six categories; the natural baseline is 40-50%.** The raw 85.5% (now 87.6% by strict measurement) is inflated by pre-model era artifacts (83% of corpus, 94.7% orphan rate). The model-era rate is **52.0%** — within the healthy range for an exploratory system. Orphans decompose into: implementation-as-investigation (~30-45%), audit/design (~25-33%), exploratory (~15-20%), genuinely lost (~20% of orphans, ~10% of total), negative results (~5-7%), and superseded (~3-5%). The actionable signal is the "genuinely lost" rate (~10% of investigations), not the raw orphan rate. Analogous to dead code: 5-15% dead code is healthy in code; 40-50% orphan rate is healthy in knowledge due to inherently higher exploration rates.

5. **Accretion, attractors, gates, and entropy are substrate-independent.** They emerge from system properties (multiple writers, no persistent memory, local correctness, non-trivial composition, no structural coordination), not substrate properties. Code, knowledge, and OPSEC are three confirmed instances. OPSEC extends the evidence to adversarial substrates where entropy is invisible to internal metrics and failure is binary/catastrophic. The same dynamics should appear in any substrate meeting the five conditions.

6. **The theory is falsifiable and conditionally predictive.** Systematic search across 15+ candidate counterexamples (natural systems: ant colonies, coral reefs, immune systems; engineered: CRDTs, blockchains, event stores; human: Wikipedia, scientific literature, shared drives) found no clean counterexamples. Every system that resists accretion does so through coordination (explicit, substrate-embedded, or environmental). The theory makes testable predictions: (a) where accretion will concentrate (at coordination gaps), (b) what interventions will reduce it (gates at compositional boundaries), (c) that removing coordination will introduce accretion. It does NOT predict accretion form, rate, or threshold — these are substrate-specific. The theory is a qualitative causal framework with quantitative evidence, not a quantitative physical law.

---

## Relationship to Existing Models

### Harness Engineering (Code Instance of Substrate Physics)

Knowledge physics provides the theoretical grounding for why harness engineering generalizes. Harness engineering describes the *discipline* — making wrong paths mechanically impossible for AI agents. Knowledge physics explains the *theory* — why the discipline works across substrates.

The mapping:
- Harness engineering's hard/soft taxonomy → applies to knowledge (all knowledge harness is currently soft)
- Harness engineering's "every convention without a gate" invariant → confirmed in knowledge substrate
- Harness engineering's accretion as thermodynamics → knowledge accretion follows the same thermodynamic pattern
- Harness engineering's compliance vs coordination distinction → knowledge failures are coordination failures (agents each investigate correctly but collectively produce 85.5% orphan rate)

Harness engineering is the code-specific instance of substrate governance. Knowledge physics is the general theory.

### System Learning Loop (Knowledge Physics in One Domain)

The system-learning-loop model describes knowledge physics without naming it. Its gap→pattern→suggestion→improvement cycle maps directly to the attractor/gate framework:

| System-Learning-Loop | Knowledge Physics | Code Physics |
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

---

## Evolution

**2025-12-25:** System-learning-loop model created, describing gap→pattern→suggestion→improvement as a closed feedback loop. Proto-knowledge-physics without naming it.

**2026-02-25:** Entropy-spiral model created, documenting how locally correct changes compose into globally incoherent systems in code substrates. Three spirals, 1,625 lost commits. Control plane immutability principle established.

**2026-03-07 to 2026-03-08:** Harness engineering model created, synthesizing accretion as thermodynamics, hard/soft harness taxonomy, compliance vs coordination failure. Established that codebase architecture is governance infrastructure.

**2026-03-09:** Knowledge physics probe (orch-go-8m7w9) empirically measured knowledge dynamics: 85.5% orphan rate, three model behaviors (attractor/capstone/dormant), zero hard gates, 4 unmerged contradicts. Confirmed substrate independence of the physics. This model created to formalize the framework.

**2026-03-09:** Natural orphan baseline probe (orch-go-80rg8) decomposed the orphan rate into six categories and established natural baseline of 40-50%. Discovered pre-model era (83% of corpus) inflates the aggregate rate; model-era rate is 52%. Probes confirmed as the structural fix converting attention-primed attractors to structurally-coupled attractors. Open question #1 answered.

**2026-03-09:** First external user profile probe (orch-go-j2ziz) established adoption sequencing: Solo Technical Researcher (STR) is the first user, not R&D labs or startups. Bottom-up adoption pattern validated across ADRs (7 years individual→industry), Obsidian (1.5M individual MAU→team features later), Benchling (200K academics→1/4 biotech IPOs). Tooling gap identified: `kb create model`/`kb create probe` don't exist, `kb init` doesn't create models/ directory — the "fundamental unit" has the least tooling support. Added Failure Mode 4 (tooling inverts importance hierarchy), Adoption Sequencing section, and open question #7 (minimum time to visible physics).

**2026-03-10:** OPSEC substrate model created (`price-watch/.kb/models/opsec-substrate/model.md`). Third confirmed substrate instance. Extends the theory to adversarial substrates with three distinctive properties: binary catastrophic failure (vs gradual degradation in code/knowledge), adversarial entropy (external actor exploits signals, making "clean up later" impossible), and multiplicative signal composition (weak signals confirm each other, unlike linear code accretion or additive knowledge accretion). Evidence: OshCut detection incident — 5 signals accumulated over 5 months while all engineering metrics were green, composed into definitive attribution in one day. Gate escalation followed the same soft → hard → structural sequence as code. Updated substrate generalization table from 2 confirmed + 4 hypothesized to 3 confirmed + 4 hypothesized.

**2026-03-10:** Health score structural improvement validation (orch-go-y642j). Decomposed the 37→73 score improvement: 90% from calibration (tracking gates, scaling thresholds, adding total_source_files), 10% from actual extractions. The 10 target files were extracted (avg 1139→410 lines, 64% reduction), but the old formula was too broken to register this. New finding: **measurement-improvement bias** — fixing a broken metric appears as improvement in the measured thing. Score jumped 37→69 in one snapshot when total_source_files was added. Added entropy metric #7 (composite health score) with this caveat. Hotspot dimension (1.9/20) identified as the honest remaining structural debt signal.

**2026-03-09:** Public release readiness audit (orch-go-8zp45) of kb-cli codebase. **Contradicted** the "single tooling gap" claim from minimal-substrate probe — found 6 additional gaps beyond `--extract-models`. **Confirmed** core cycle commands are fully standalone. **Extended** substrate/orchestration separation with three-tier coupling taxonomy: hard (ask, link — break without orch), soft (reflect --create-issue, context --siblings — degrade gracefully), none (init, create, search, list — fully standalone). Code health: 57.6% test coverage, 3 failing tests, 3 direct dependencies, no go vet errors. Key blockers: missing LICENSE file, incomplete README, hardcoded orch paths.

**2026-03-10:** Falsifiability probe (orch-go-dqv2o). Systematic counterexample search across 15+ systems in three domains (natural, engineered, human). No clean counterexamples found. Three key extensions: (1) Added condition 5 — "non-trivial composition" — to prevent over-prediction in additive substrates (logs, sensor data, coral reefs) that meet conditions 1-4 but don't degrade because composition is trivial. (2) Coordination taxonomy: explicit (type systems), substrate-embedded (CRDTs), environmental (stigmergy). Digital substrates lack environmental coordination, explaining why they require engineered gates. (3) Continuous risk formulation: `accretion_risk = f(amnesia × complexity / coordination)` — more precise than binary conditions. Theory verdict: conditionally predictive, publishable with composition refinement.

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
- `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-physics-accretion-attractor-gate-dynamics.md` — Full empirical measurement (1,166 investigations, 32 models, 187 probes)
- `.kb/models/knowledge-physics/probes/2026-03-09-probe-natural-orphan-baseline-categorization.md` — Orphan taxonomy, era-adjusted rates, natural baseline (40-50%), probe displacement finding
- `.kb/models/knowledge-physics/probes/2026-03-09-probe-minimal-kb-substrate-cycle-dependencies.md` — Minimal substrate identification: 5 components (agent + kb + git + .kb/ + skill), substrate/orchestration separation confirmed, context injection gap identified
- `.kb/models/knowledge-physics/probes/2026-03-09-probe-first-external-user-profile-analysis.md` — First user is Solo Technical Researcher (STR), not R&D lab. Adoption sequencing validated across 4 analogous tools. Tooling gap: `kb create model`/`kb create probe` missing, `kb init` doesn't create models/
- `.kb/models/knowledge-physics/probes/2026-03-09-probe-kb-cli-public-release-readiness-audit.md` — kb-cli public release audit: core cycle standalone, 6 gaps beyond --extract-models, three-tier coupling taxonomy (hard/soft/none), 7 blocking changes for v0.1
- `.kb/models/knowledge-physics/probes/2026-03-10-probe-health-score-structural-improvement-validation.md` — Health score 37→73 decomposition: 90% calibration, 10% extraction. Measurement-improvement bias finding. Hotspot dimension (1.9/20) is honest remaining signal.
- `.kb/models/knowledge-physics/probes/2026-03-10-probe-falsifiability-counterexamples.md` — Falsifiability probe: 15+ counterexamples tested, none survive. Fifth condition (non-trivial composition) identified. Coordination taxonomy (explicit/substrate-embedded/environmental). Theory is conditionally predictive.

**Related Models:**
- `.kb/models/harness-engineering/model.md` — Code instance of substrate physics, hard/soft harness taxonomy
- `.kb/models/system-learning-loop/model.md` — Knowledge physics in context-gap domain without naming it
- `.kb/models/skill-content-transfer/model.md` — Attention primers vs action directives, mechanism behind knowledge attractors
- `.kb/models/entropy-spiral/model.md` — Feedback loops in code substrate, control plane immutability

**Knowledge Physics Assessment (in system-learning-loop model):**
- Section "Knowledge Physics Assessment (2026-03-09)" in system-learning-loop/model.md documents the mapping and empirical evidence that prompted this model's creation.

**Thread:**
- `.kb/threads/2026-03-09-knowledge-physics-does-knowledge-have.md` — Initial question formulation
