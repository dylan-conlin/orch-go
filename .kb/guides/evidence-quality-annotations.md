# Evidence Quality Annotations

**Purpose:** Standardize the 4-level evidence quality taxonomy for annotating claims in model files. This is Layer 1 of the three-layer confidence propagation design.

**When to use:** Every time you write or update a claim in a `model.md` file, annotate it with an `**Evidence quality:**` line. When merging probe findings into a model (probe-to-model merge), update the annotation to reflect the new evidence.

---

## The 4-Level Taxonomy

| Level | Definition | When to Use | Example Annotation |
|-------|-----------|-------------|-------------------|
| **Replicated** | Confirmed across 2+ independent sources, experiments, or contexts | Claim verified by multiple investigations, probes, or real-world observations | `**Evidence quality:** Replicated (3 sources: behavioral compliance probe, testing baseline, simplification investigation)` |
| **Single-source** | Measured or observed, but only in one experiment/context | Test results, line counts, command outputs from a single investigation or probe | `**Evidence quality:** Single-source measured (behavioral compliance probe, Feb 24)` |
| **Analytical** | Logical deduction from measured evidence, not directly observed | Inferences, design rationale derived from empirical findings | `**Evidence quality:** Analytical (deduced from knowledge-vs-behavioral transfer asymmetry)` |
| **Assumed** | Stated without direct evidence; plausible but untested | Hypotheses, extrapolations, claims carried forward without verification | `**Evidence quality:** Assumed (no direct measurement; based on analogy with other systems)` |

---

## Annotation Format

Place the annotation immediately after the claim paragraph in a model file:

```markdown
### Claim N: [Title]

[Claim description paragraph.]

**Evidence quality:** [Level] ([brief provenance — sources, dates, or reasoning]).
```

The provenance parenthetical should be specific enough that a reader can trace the evidence:
- **Replicated:** Name the sources. E.g., `(3 sources: probe X, investigation Y, production observation Z)`
- **Single-source:** Name the source and date. E.g., `(behavioral compliance probe, Feb 24)`
- **Analytical:** Name what it's deduced from. E.g., `(deduced from Claims 1 and 2)`
- **Assumed:** State why it's assumed. E.g., `(no direct measurement; extrapolated from similar system)`

---

## Level Transitions

Evidence quality changes over time. Common transitions:

| Transition | Trigger |
|-----------|---------|
| Assumed → Single-source | A probe or investigation measures the claim |
| Assumed → Analytical | Reasoning is formalized from other measured claims |
| Single-source → Replicated | A second independent source confirms the finding |
| Single-source → Single-source (caveated) | Replication attempt fails — add caveat |
| Replicated → Single-source (caveated) | Evidence is retracted or contradicted |
| Any → Assumed | All supporting evidence is invalidated |

When a probe **contradicts** a claim, downgrade the annotation and note what changed:

```markdown
**Evidence quality:** Measured but caveated (single-source, replication failed, N=3).
```

When a probe **confirms** a claim with new evidence, upgrade if warranted:

```markdown
**Evidence quality:** Replicated (4 sources: [original], [new source]).
```

---

## Mechanical Detection

The `kb audit provenance` command scans for these annotations. To ensure your annotations are detectable:

1. Use the exact prefix: `**Evidence quality:**`
2. Start with one of the four level keywords: `Replicated`, `Single-source`, `Analytical`, `Assumed`
3. Variant forms are acceptable after the keyword: `Measured but caveated`, `Highest-confidence`, `Multi-source analytical` — the scanner matches on the leading keyword

**Gap detection:** Claims without an `**Evidence quality:**` line are flagged as unannotated. The scanner counts these per model to produce a coverage percentage.

---

## Examples from Practice

These examples are drawn from the orchestrator-skill model, where this taxonomy emerged organically:

**Replicated:**
> **Evidence quality:** Highest-confidence finding in the cluster. Confirmed across 4 independent sources: behavioral compliance probe (Feb 24), testing baseline (Mar 1), simplification investigation (Mar 4), grammar-first investigation (Mar 4).

**Single-source (caveated):**
> **Evidence quality:** Measured but caveated (single-source, replication failed, N=3).

**Analytical (multi-source):**
> **Evidence quality:** Multi-source analytical (2 investigations). Supported by agent framework landscape survey (Mar 1, 2026).

**Analytical (highest-replication):**
> **Evidence quality:** Highest-replication analytical finding (5 sources across 5 investigations).

---

## Integration with Probe-to-Model Merge

When merging probe findings into a model (per the worker-base probe-to-model merge protocol):

1. **Read** the existing `**Evidence quality:**` annotation on affected claims
2. **Update** the level based on what the probe found:
   - Probe confirms → strengthen (e.g., single-source → replicated)
   - Probe contradicts → weaken and caveat (e.g., single-source → caveated)
   - Probe extends → add the probe as an additional source
3. **Add** annotations to any new claims introduced by the merge
4. **Never** leave a new claim unannotated

---

## Common Mistakes

| Mistake | Why It's Wrong | Fix |
|---------|---------------|-----|
| Omitting the annotation entirely | Downstream consumers treat unannotated claims as established fact | Always annotate, even if `Assumed` |
| Citing probe count without naming sources | "3 probes" doesn't tell the reader which probes or whether they're independent | Name the specific sources |
| Treating analytical findings as replicated | Deduction from evidence ≠ independent measurement | Use `Analytical` unless multiple independent measurements exist |
| Not downgrading after replication failure | Stale high-confidence ratings propagate false certainty | Add caveat immediately when replication fails |
| Upgrading based on citations, not evidence | 4 artifacts citing a claim doesn't replicate it — it just propagates | Only upgrade when new independent evidence exists |

---

## Reference

- **Origin:** Evidence inventory probe (67 claims, Mar 12, 2026) — taxonomy emerged from classifying claims across the orchestrator-skill investigation cluster
- **Design:** `.kb/investigations/2026-03-12-inv-design-confidence-propagation-knowledge-system.md`
- **Exemplar model:** `.kb/models/orchestrator-skill/model.md` (6 claims, 100% annotated)
- **Enforcement:** Layer 2 (probe-to-model merge gate) and Layer 3 (`kb audit provenance` command)
