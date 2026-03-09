# Model: Knowledge Physics

**Domain:** Multi-Agent Knowledge Systems / Substrate Governance / Entropy Management
**Last Updated:** 2026-03-09
**Synthesized From:**
- `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-physics-accretion-attractor-gate-dynamics.md` — Empirical measurement across 1,166 investigations, 32 models, 187 probes
- `.kb/models/harness-engineering/model.md` — Hard/soft harness taxonomy, accretion as thermodynamics, compliance vs coordination failure
- `.kb/models/system-learning-loop/model.md` — Gap→pattern→suggestion→improvement as proto-knowledge-physics
- `.kb/models/skill-content-transfer/model.md` — Attention primers vs action directives, three-type vocabulary
- `.kb/models/entropy-spiral/model.md` — Feedback loops, control plane immutability, 1,625 lost commits

---

## Summary (30 seconds)

Knowledge exhibits the same physics as code when multiple amnesiac agents contribute to a shared mutable substrate. Accretion, attractors, gates, and entropy are substrate-independent dynamics — they emerge from system properties (multiple writers, no persistent memory, local correctness without global coordination), not from properties of the substrate itself. Empirical measurement across orch-go's knowledge corpus: 85.5% orphan rate (997/1,166 investigations unconnected to any model), three model behaviors (attractor/capstone/dormant), zero hard knowledge gates (all transitions advisory), and 4 unmerged contradiction verdicts. Code and knowledge are two instances of the same physics. Others include database schemas, config systems, API surfaces, and documentation. The generalization: harness engineering is not code governance — it is substrate governance. Any shared mutable substrate where amnesiac agents contribute requires attractors (structural destinations), gates (enforcement blocking wrong paths), and entropy measurement (detection of compositional failure).

---

## Core Claim

Knowledge exhibits the same physics as code when multiple amnesiac agents contribute to a shared mutable substrate.

The dynamics — accretion, attractors, gates, entropy — are properties of the system configuration, not the substrate. They emerge whenever four conditions hold:

1. **Multiple agents write** to the substrate
2. **Agents are amnesiac** — no cross-session memory
3. **Contributions are locally correct** — each passes local validation
4. **No structural coordination mechanism exists** — locally correct + locally correct ≠ globally correct

Code was the first substrate where we observed these dynamics (daemon.go +892 lines from 30 correct commits). Knowledge is the second (85.5% orphan rate, 997 unconnected investigations). The physics are identical; only the substrate-specific manifestations differ.

---

## Core Mechanism

### 1. Accretion Dynamics

Accretion is entropy: individually correct contributions compose into structural degradation when shared infrastructure is missing. In code, this manifests as file bloat and duplication. In knowledge, it manifests as orphan investigations and semantic overlap.

**Empirical measurement (2026-03-09):**

| Category | Connected | Total | Orphan Rate |
|----------|-----------|-------|-------------|
| Active investigations | 103 | 206 | 50.0% |
| Archived investigations | 57 | 890 | 93.6% |
| Synthesized investigations | 7 | 66 | 89.4% |
| **All investigations** | **169** | **1,166** | **85.5%** |

**Knowledge accretion signatures:**
- **Orphan investigations** — work product with no structural connection to synthesized understanding (models). 85.5% overall, 93.6% in the archive.
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

### 3. Gate Deficit

Every knowledge transition is either ungated or advisory-only:

| Transition | Status | Gate Mechanism |
|------------|--------|---------------|
| Investigation → model | **UNGATED** | No automated model update when investigation contradicts model |
| Probe → model update | **UNGATED** | Skill template says "merge findings before completion" but no tooling validates. 4 probes with "contradicts" verdicts sit unmerged |
| Quick entry → decision | **UNGATED** | No dedup checking against existing entries or decisions |
| Decision → implementation | **UNGATED** (1/56 exception) | Only 1 of 56 decisions has a `kb agreements` check |
| Investigation prior work | **SOFT** (52% adoption) | Template includes it, 48% of investigations skip it |
| Knowledge consistency at commit | **UNGATED** | Pre-commit hooks only run on `*.go` files, not `.kb/` files |

**The parallel to code is exact.** The harness-engineering model documented that "every convention without a gate will eventually be violated." The knowledge system proves this claim across every transition:

- Probe-to-model merge is a convention → 4 contradicts verdicts sit unmerged
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

3. **Contradicts backlog** — count of "contradicts" probe verdicts sitting unmerged (baseline: 4). These are the knowledge equivalent of known bugs left unfixed. Direct measure of model accuracy debt.

4. **Synthesis debt** — investigation clusters without models (baseline: 4 clusters, 17 investigations). Measures how much investigative work product remains unsynthesized.

5. **Claim density** — claims per model (equivalent of lines per file). Too many claims → model needs splitting. Models should be focused enough that each claim can be probed.

6. **Quick entry duplication rate** — duplicate entries in entries.jsonl. Measures whether the "quick capture" path is creating noise instead of signal. No automated dedup exists.

---

## Substrate Generalization

The physics hold for any shared mutable substrate where the four conditions are met (multiple agents, amnesiac, locally correct, no structural coordination).

| Substrate | Accretion | Attractors | Gates | Entropy Signal |
|-----------|-----------|-----------|-------|----------------|
| **Code** (orch-go) | daemon.go +892 lines, 6 cross-cutting dupes | pkg/ packages (structural coupling) | Pre-commit, spawn, completion, `go build` | Fix:feat ratio, hotspot analysis |
| **Knowledge** (.kb/) | 85.5% orphan investigations | Models pull probes/investigations (attention priming) | None hard (all advisory) | Stale decisions, synthesis backlog |
| **Database schemas** | Column bloat, unused tables | Normalized entity structure | Migration validation, FK constraints | Dead columns, orphan tables |
| **Config systems** | Setting sprawl | Config categories/namespaces | Schema validation | Unused settings, duplicate keys |
| **API surfaces** | Endpoint bloat | Resource-oriented design | Contract testing, versioning | Deprecated endpoints, inconsistent naming |
| **Documentation** | Doc sprawl, contradictions | Doc hierarchy/taxonomy | Link validation, freshness checks | Stale pages, orphaned docs |

**Minimal substrate properties:**

1. **Mutable** — agents can add/modify content
2. **Shared** — multiple agents read/write
3. **Compositional** — individual contributions must compose into a coherent whole
4. **Amnesiac** — no single agent has full context of all prior contributions

**The generalization of harness engineering:** Governing any shared mutable substrate where amnesiac agents contribute requires the same three mechanisms:

- **Attractors** — structural destinations that route contributions (packages, models, schemas, resource hierarchies)
- **Gates** — enforcement that blocks wrong paths (pre-commit, build, migration validation, contract testing)
- **Entropy measurement** — detection of when composition is failing (bloat, orphans, contradictions, staleness)

---

## Critical Invariants

1. **Every convention without a gate will eventually be violated — in knowledge too.** The knowledge system has zero hard gates. Every knowledge convention (Prior Work tables, probe-to-model merge, dedup checking) is violated at significant rates. This is the same invariant from harness-engineering, empirically confirmed in a second substrate.

2. **Models are the fundamental unit of knowledge organization.** Without models, knowledge is homeless. The 85.5% orphan rate is not just a metric — it means 997 investigations exist as isolated artifacts that don't contribute to synthesized understanding. Models provide the gravitational centers that organize investigative work product.

3. **Attention-primed attractors (knowledge) may be fundamentally weaker than structurally-coupled attractors (code).** Code attractors work through compilation — imports route code mechanically. Knowledge attractors work through context injection — kb context frames agent investigation. The attention mechanism is the same as stance transfer (skill-content-transfer model), which is known to be scenario-specific and not a universal amplifier. Whether this weakness is inherent (attention priming is fundamentally softer) or contingent (knowledge attractors are just ungated) is an open question.

4. **The orphan rate is partially natural but 85.5% signals systemic under-synthesis.** Not all investigations need models — exploratory work, one-off debugging, and negative results are legitimately orphaned. But 85.5% (and 93.6% in the archive) far exceeds the natural baseline. The system produces investigations faster than it synthesizes them into models. This is the knowledge equivalent of code accreting faster than it's extracted.

5. **Accretion, attractors, gates, and entropy are substrate-independent.** They emerge from system properties (multiple writers, no persistent memory, local correctness, no structural coordination), not substrate properties. Code and knowledge are two confirmed instances. The same dynamics should appear in any substrate meeting the four conditions.

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

Every knowledge gate is advisory. The "merge findings before completion" instruction in probe templates is a convention — 4 contradicts verdicts sit unmerged. Prior Work tables are a convention — 48% skip them. Without hard enforcement, conventions degrade under time pressure. This is the same dynamic as daemon.go growing past the stated 1,500-line convention in CLAUDE.md.

### Failure Mode 3: Attention-Primed Attractors Lose Under Pressure

When kb context injection includes a relevant model, agents are primed to engage with it. But when the task is urgent or the context window is crowded, the attention primer competes with task pressure. Code attractors (package structure) don't compete — they're enforced by the compiler. Knowledge attractors can be ignored.

### Failure Mode 4: No Contradiction Resolution Mechanism

When a probe contradicts a model claim, the finding is recorded in the probe file but no mechanism forces the model to be updated. 4 contradicts verdicts are currently unmerged. Over time, models accumulate stale or contradicted claims that new agents receive as authoritative via kb context injection — creating a knowledge equivalent of stale cache invalidation.

---

## Open Questions

1. **Is the 85.5% orphan rate a problem or a natural property of exploratory systems?** Not all investigations need models. What's the natural baseline for orphaned work product in an exploratory system? Is there an analogy to "dead code" — code that exists but serves no current purpose, and that's fine?

2. **What would knowledge pre-commit hooks look like?** Code pre-commit hooks validate syntax, lint, and compilation. Knowledge pre-commit hooks would need to check: does this investigation cite a Prior Work table? Does this probe reference its model? Does this model.md contradict itself? The technical challenge: knowledge validation is semantic, not syntactic.

3. **Are attention-primed attractors weaker than structural attractors, or just ungated?** If knowledge models had hard gates (e.g., investigations blocked from closing without citing a model), would the orphan rate approach code's structural coupling rates? Or is attention priming inherently weaker than compilation-enforced coupling?

4. **What's the right threshold for claims-per-model bloat?** Code has lines-per-file thresholds (800 warning, 1,500 critical). What's the equivalent for model claims? When does a model accumulate enough claims that it needs splitting?

5. **Does adding hard knowledge gates reduce entropy or add ceremony?** Code gates (pre-commit, spawn, completion) have demonstrated ROI — they prevent measurable damage. But knowledge creation is more exploratory than code writing. Would gates on investigation creation slow down legitimate exploration? The analogy: mandatory code review slows velocity but catches coordination failures. Mandatory model-linking might slow exploration but catch orphan accumulation.

6. **Can knowledge attractors be structurally coupled?** Code attractors work through imports (structural coupling). Could knowledge attractors work through tooling — e.g., `kb create investigation` requiring a `--model` flag? This would convert attention priming into structural coupling without requiring semantic validation.

---

## Evolution

**2025-12-25:** System-learning-loop model created, describing gap→pattern→suggestion→improvement as a closed feedback loop. Proto-knowledge-physics without naming it.

**2026-02-25:** Entropy-spiral model created, documenting how locally correct changes compose into globally incoherent systems in code substrates. Three spirals, 1,625 lost commits. Control plane immutability principle established.

**2026-03-07 to 2026-03-08:** Harness engineering model created, synthesizing accretion as thermodynamics, hard/soft harness taxonomy, compliance vs coordination failure. Established that codebase architecture is governance infrastructure.

**2026-03-09:** Knowledge physics probe (orch-go-8m7w9) empirically measured knowledge dynamics: 85.5% orphan rate, three model behaviors (attractor/capstone/dormant), zero hard gates, 4 unmerged contradicts. Confirmed substrate independence of the physics. This model created to formalize the framework.

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

**Related Models:**
- `.kb/models/harness-engineering/model.md` — Code instance of substrate physics, hard/soft harness taxonomy
- `.kb/models/system-learning-loop/model.md` — Knowledge physics in context-gap domain without naming it
- `.kb/models/skill-content-transfer/model.md` — Attention primers vs action directives, mechanism behind knowledge attractors
- `.kb/models/entropy-spiral/model.md` — Feedback loops in code substrate, control plane immutability

**Knowledge Physics Assessment (in system-learning-loop model):**
- Section "Knowledge Physics Assessment (2026-03-09)" in system-learning-loop/model.md documents the mapping and empirical evidence that prompted this model's creation.

**Thread:**
- `.kb/threads/2026-03-09-knowledge-physics-does-knowledge-have.md` — Initial question formulation
