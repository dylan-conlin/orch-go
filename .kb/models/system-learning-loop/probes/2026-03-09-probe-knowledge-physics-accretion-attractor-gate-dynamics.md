# Probe: Knowledge Physics — Accretion/Attractor/Gate Dynamics in Knowledge Systems

**Model:** system-learning-loop
**Date:** 2026-03-09
**Status:** Complete

---

## Question

The system-learning-loop model describes gap->pattern->suggestion->improvement as a closed feedback loop and frames it as observability. Does knowledge exhibit the same physics as code — accretion, attractors, gates, entropy? If so, the system-learning-loop model is describing knowledge physics without naming it, and harness engineering generalizes from "governing agents writing code" to "governing agents contributing to any shared mutable substrate."

Specifically testing:
1. Does knowledge accrete (orphan rate, duplication)?
2. Do models act as attractors (cluster investigations)?
3. Are knowledge gates missing or ungated paths exist?
4. Are current entropy metrics the right ones?
5. Does the system-learning-loop already describe knowledge physics?
6. Does the physics generalize to other substrates?

---

## What I Tested

### 1. Accretion Measurement

Counted investigation connections to models across the full corpus (1,166 investigations, 32 models, 187 probes).

```bash
# Forward references: model.md files citing investigations
grep -r "\.kb/investigations/" .kb/models/*/model.md | wc -l

# Reverse references: investigations citing models
grep -rl "\.kb/models/" .kb/investigations/ | wc -l

# Quick entry duplication check
wc -l .kb/quick/entries.jsonl

# kb reflect synthesis opportunities
kb reflect --type synthesis
```

### 2. Attractor Effect Measurement

Measured investigation density referencing specific domains before and after model creation for three models:

```bash
# Harness-engineering model (created 2026-03-07/08)
git log --all --oneline --follow .kb/models/harness-engineering/model.md

# Entropy-spiral model (created 2026-02-25)
git log --all --oneline --follow .kb/models/entropy-spiral/model.md

# Daemon-autonomous-operation model (created 2026-02-08)
git log --all --oneline --follow .kb/models/daemon-autonomous-operation/model.md
```

Counted investigations referencing each domain's keywords before and after model creation date.

### 3. Missing Gates Audit

Tested each knowledge transition path for blocking gates:
- Investigation creation: `kb create investigation`
- Quick entry creation: `kb quick decide`
- Probe → model update: `orch complete` pipeline
- Decision → implementation: `kb agreements check`

Checked pre-commit hooks, completion pipeline, and daemon for knowledge validation.

### 4. Entropy Measurement

```bash
kb reflect --type stale
kb reflect --type synthesis
```

Analyzed current metrics vs proposed knowledge-physics metrics.

### 5. System-Learning-Loop Mapping

Read the model.md and mapped gap->pattern->suggestion->improvement to attractor/gate/entropy framework.

---

## What I Observed

### 1. ACCRETION: Knowledge Bloat Is Real and Measurable

**Orphan rate: 85.5%** (997 of 1,166 investigations have no traceable connection to any model).

| Category | Connected | Total | Orphan Rate |
|----------|-----------|-------|-------------|
| Active investigations | 103 | 206 | 50.0% |
| Archived investigations | 57 | 890 | 93.6% |
| Synthesized investigations | 7 | 66 | 89.4% |
| **All investigations** | **169** | **1,166** | **85.5%** |

This is the knowledge equivalent of code bloat. 85.5% of investigative work product has no structural connection to the synthesized understanding (models). The archive is essentially dead weight — 890 files with 93.6% orphan rate.

**Quick entry duplication exists.** Found at least one confirmed duplicate pair in entries.jsonl (kb-69d5cf and kb-9f3964, both recording "verify.Comment uses Text field"). No automated dedup checking exists.

**Synthesis backlog: 4 clusters totaling 17 investigations** that should be models but aren't. The synthesis opportunity detection works but is retroactive — it detects bloat after it accumulates, not at contribution time.

### 2. ATTRACTOR EFFECT: Models Pull Findings, But Three Distinct Behaviors

| Model | Created | Pre-model ref rate | Post-model ref rate | Probes | Behavior |
|-------|---------|-------------------|---------------------|--------|----------|
| harness-engineering | 2026-03-08 | 2.0% | 100% | 3 | **Strong attractor** (launch burst) |
| daemon-autonomous-operation | 2026-02-08 | 12.5% | 50.0% | 34 | **Sustained attractor** |
| entropy-spiral | 2026-02-25 | 12.5% | 8.8% | 2 | **Capstone** (settled topic) |

Three model behaviors observed:
- **Attractor**: Model creation increases investigation density toward it (harness, daemon)
- **Capstone**: Model creation *decreases* investigation density — it synthesizes and settles a topic (entropy-spiral)
- **Dormant**: Model exists but generates no probes or investigations (7 models with 0 probes)

The probe count is the strongest attractor metric: daemon-autonomous-operation attracted 34 probes across 13 dates over 21 days. This is sustained gravitational pull.

**Key finding:** Attractors work differently in knowledge than in code. In code, a package attractor pulls code toward it mechanically (imports, function calls). In knowledge, a model attractor pulls attention toward it — agents spawned in the model's domain receive model claims via kb context injection, which frames their investigation. The mechanism is attention priming (same as stance transfer from the skill-content-transfer model), not structural routing.

### 3. MISSING GATES: All Knowledge Paths Are Ungated

Every knowledge transition is either ungated or advisory-only:

| Transition | Status | Gate Mechanism |
|------------|--------|---------------|
| Investigation → model | **UNGATED** | No automated model update when investigation contradicts model |
| Probe → model update | **UNGATED** | Skill template says "merge findings before completion" but no tooling validates. 4 probes contain "contradicts" verdicts with no model update |
| Quick entry → decision | **UNGATED** | No dedup checking against existing entries or decisions |
| Decision → implementation | **UNGATED** (1/56 exception) | Only 1 of 56 decisions has an `kb agreements` check |
| Investigation Prior Work | **SOFT** (52% adoption) | Template includes it, 48% of investigations skip it |
| Knowledge consistency at commit | **UNGATED** | Pre-commit hooks only run on *.go files, not .kb/ files |

**The striking parallel to code:** In the code system, the harness-engineering model documented that "every convention without a gate will eventually be violated." The knowledge system proves this claim — every knowledge convention (Prior Work tables, probe-to-model merge, dedup checking) is a convention without a gate, and each is violated at significant rates.

**4 probes contain "contradicts" verdicts** that sit unmerged in the system. These are the knowledge equivalent of known bugs left unfixed.

### 4. ENTROPY METRICS: Current Metrics Are Incomplete

**Current metrics (via `kb reflect`):**
- 47 synthesis opportunities (clusters of 3+ investigations on same topic)
- 66+ stale decisions (>7 days, zero citations)
- 1 stale model (entropy-spiral went 26 days without a probe)
- 3 recurring defect classes

**What's missing (knowledge equivalents of code metrics):**

| Code Metric | Knowledge Equivalent | Status |
|-------------|---------------------|--------|
| Lines of code per file | Claims per model | **Not tracked** |
| File bloat (>1,500 lines) | Model bloat (>N claims unprobed) | **Not tracked** |
| Fix:feat ratio | Contradiction:extension ratio | **Not tracked** |
| Duplication across files | Semantic overlap across investigations | **Not tracked** |
| Churn rate | Investigation-to-model conversion rate | **Partially** (synthesis opportunities) |
| Dead code | Dead investigations (orphaned, never cited) | **Partially** (85.5% orphan rate measured here) |

**Proposed knowledge entropy metrics:**

1. **Orphan rate** — % investigations not connected to any model (currently 85.5%, should decrease over time as models absorb findings)
2. **Claims-per-model** — number of claims in a model.md (like lines-per-file; too many → model needs splitting)
3. **Unprobed claims ratio** — % of model claims never tested by a probe (like untested code)
4. **Contradiction backlog** — count of "contradicts" verdicts sitting unmerged (currently 4)
5. **Synthesis backlog** — investigation clusters awaiting model creation (currently 4 clusters, 17 investigations)
6. **Decision enforcement rate** — % of decisions with `kb agreements` checks (currently 1/56 = 1.8%)

### 5. SYSTEM-LEARNING-LOOP IS PROTO-KNOWLEDGE-PHYSICS

The system-learning-loop model describes:
```
gap → pattern (RecurrenceThreshold=3) → suggestion → improvement → fewer gaps
```

This maps directly to the attractor/gate framework:

| System-Learning-Loop | Knowledge Physics | Code Physics |
|---------------------|-------------------|--------------|
| Gap recording | Entropy detection | Hotspot analysis |
| Pattern detection (threshold=3) | Attractor emergence | Package structure as routing |
| Suggestion generation | Gate recommendation | Pre-commit gate |
| Resolution tracking | Gate enforcement | Completion verification |
| 30-day retention | Entropy decay | Stale code pruning |

**The reframe:** The system-learning-loop is describing knowledge physics in the specific domain of "context gaps." It's a specialized instance of the general pattern:

```
Entropy (unstructured contributions)
    → Attractor formation (patterns crystallize into models/packages)
    → Gate enforcement (patterns become hard constraints)
    → Entropy reduction (future contributions route through structure)
```

The system-learning-loop's RecurrenceThreshold=3 is the knowledge equivalent of the code system's "3+ investigations trigger model creation" threshold. Both are attractor formation criteria — when enough signal accumulates, a structural attractor forms.

**What the system-learning-loop doesn't name:**
- It doesn't call gap recording "entropy measurement"
- It doesn't call pattern detection "attractor formation"
- It doesn't call suggestions "proposed gates"
- It doesn't recognize that the gap→pattern→suggestion cycle is the scientific method (observation → hypothesis → prediction)

### 6. SUBSTRATE GENERALIZATION

The physics hold for any shared mutable substrate where:
1. **Multiple agents contribute** (no single-author bottleneck)
2. **Contributions are locally correct** (each agent's work passes local validation)
3. **No persistent memory** (each agent session starts fresh)
4. **Composition isn't guaranteed** (locally correct + locally correct ≠ globally correct)

| Substrate | Accretion | Attractors | Gates | Entropy Signal |
|-----------|-----------|-----------|-------|----------------|
| Code (orch-go) | daemon.go +892 lines | pkg/ packages | Pre-commit, spawn, completion | Fix:feat ratio, hotspot analysis |
| Knowledge (.kb/) | 85.5% orphan investigations | Models pull probes/investigations | None hard (all advisory) | Stale decisions, synthesis backlog |
| Database schemas | Column bloat, unused tables | Normalized entity structure | Migration validation, FK constraints | Dead columns, orphan tables |
| Config systems | Setting sprawl | Config categories/namespaces | Schema validation | Unused settings, duplicate keys |
| API surfaces | Endpoint bloat | Resource-oriented design | Contract testing, versioning | Deprecated endpoints, inconsistent naming |
| Documentation | Doc sprawl, contradictions | Doc hierarchy/taxonomy | Link validation, freshness checks | Stale pages, orphaned docs |

**Minimal substrate properties for knowledge physics to apply:**
1. **Mutable** — agents can add/modify content
2. **Shared** — multiple agents read/write
3. **Compositional** — individual contributions must compose into a coherent whole
4. **Amnesiac** — no single agent has full context of all prior contributions

**The generalization of harness engineering:** Governing any shared mutable substrate where amnesiac agents contribute requires the same three mechanisms:
- **Attractors** — structural destinations that route contributions (packages, models, schemas, resource hierarchies)
- **Gates** — enforcement that blocks wrong paths (pre-commit, build, migration validation, contract testing)
- **Entropy measurement** — detection of when composition is failing (bloat, orphans, contradictions, staleness)

---

## Model Impact

- [x] **Confirms** invariant: The system-learning-loop's gap→pattern→suggestion→improvement cycle IS knowledge physics (attractor formation from entropy). RecurrenceThreshold=3 is an attractor formation criterion, same as the model creation threshold.

- [x] **Extends** model with:

  1. **The system-learning-loop describes one specialized instance of knowledge physics** (context gaps), but the same dynamics apply to all knowledge artifacts (investigations, models, decisions, quick entries). The model should be reframed as an instance of the general knowledge physics pattern.

  2. **Knowledge physics differs from code physics in one key way:** Code attractors work through structural coupling (imports, compilation). Knowledge attractors work through attention priming (kb context injection frames agent investigation). This maps to the skill-content-transfer model's distinction between structural enforcement (hard harness) and attention primers (stance).

  3. **The knowledge system has zero hard gates.** While the code system has pre-commit hooks, spawn gates, and completion verification, every knowledge gate is advisory. This is the most significant finding: the knowledge substrate is governed entirely by soft harness, which the harness-engineering model has shown degrades under pressure.

  4. **Six proposed knowledge entropy metrics** that would make knowledge physics measurable: orphan rate, claims-per-model, unprobed claims ratio, contradiction backlog, synthesis backlog, decision enforcement rate.

  5. **Three model behaviors discovered** — attractor (pulls future work), capstone (settles topic), dormant (no ongoing engagement). The system-learning-loop only describes the attractor pattern.

  6. **Substrate generalization holds.** The physics apply to any shared mutable substrate with multiple amnesiac agents. Minimal properties: mutable, shared, compositional, amnesiac. This means harness engineering generalizes from code governance to substrate governance.

---

## Notes

**Recommendation:** Create `.kb/models/knowledge-physics/model.md` to formalize this framework. The system-learning-loop model should be reframed as an instance (specialized to context gaps) rather than the general theory.

**Relationship to harness-engineering model:** Knowledge physics provides the theoretical grounding for why harness engineering generalizes. Harness engineering is the discipline; knowledge physics is the theory explaining why the discipline works across substrates.

**Key evidence to verify in future probes:**
- Does adding hard gates to the knowledge system (e.g., blocking commits that add .kb/ files contradicting existing models) actually reduce knowledge entropy?
- Does the 85.5% orphan rate decrease over time as more models are created, or does investigation volume outpace model synthesis?
- Do capstone models (like entropy-spiral) represent a healthy lifecycle stage, or a sign that the model has become stale?
