# Evidence-Tier Annotation Standard

**Purpose:** Per-claim evidence grading for KB model files. Prevents compression-before-validation — where local experimental results get packaged as general claims without the evidence base to support them.

**Motivation:** Epistemic audit (2026-03-22) found the coordination model and other KB models suffer from overclaiming. Each synthesis layer adds generality without scope-restricting. This standard makes the gap between evidence and language visible and auditable.

---

## Evidence Tiers

Every claim in a model.md file must include an `**Evidence quality:**` annotation with one of these tiers:

| Tier | Meaning | Example |
|------|---------|---------|
| **assumed** | No direct evidence; inference or convention | "Models are the natural unit of knowledge" |
| **working-hypothesis** | Plausible synthesis from indirect evidence, not directly tested | "Align is the highest-leverage primitive" |
| **observed** | Single experiment or observation, one context | "In N=10 same-file trials, placement prevented conflicts" |
| **replicated** | Reproduced across >1 independent context (different repos, task types, models) | "Placement succeeded across simple and complex tasks (N=80)" |
| **validated** | Replicated + external confirmation or formal test | "6 external frameworks confirm gate/attractor correlation" |

**Legacy mappings** (existing annotations map automatically):
- `Single-source measured` → observed
- `Multi-source analytical` → replicated
- `Replicated (N sources)` → replicated

---

## Annotation Format

Place after the claim, before the next claim:

```markdown
### Claim 1: Communication is insufficient for coordination

In the tested same-file parallel-edit scenarios, messaging and context-sharing
did not reduce merge conflicts.

**Evidence quality:** Observed (single experiment, N=80, one repo, one task family).
```

The annotation is one line: `**Evidence quality:** <tier> (<qualifying details>).`

The qualifying details in parentheses are optional but recommended — they make the scope explicit.

---

## Language Calibration by Tier

The declared tier constrains what language the claim prose can use. The provenance audit (`orch kb provenance`) flags claims where prose exceeds the declared tier.

### assumed / working-hypothesis

**Allowed:** "may", "suggests", "appears to", "plausible that", "working hypothesis"
**Flagged:** "fundamentally", "universally", "always", "never", "proven", "impossible"

Example (good):
> Align may be the highest-leverage primitive — agents communicating perfectly while maintaining divergent models of correctness appears to be a common failure pattern.

Example (overclaimed):
> Align is fundamentally the most important primitive and universally determines coordination success.

### observed

**Allowed:** "in tested scenarios", "we observed", "the experiment showed", scope qualifiers
**Flagged:** "fundamentally", "universally", "general to any", "all systems", "proven"

Example (good):
> In the tested same-file parallel-edit scenarios, messaging did not reduce merge conflicts, while explicit structural placement eliminated them.

Example (overclaimed):
> Communication does not produce coordination. Frameworks relying on messaging are fundamentally flawed.

### replicated

**Allowed:** "consistently", "across tested contexts", "reliably", stronger claims within scope
**Flagged:** "fundamentally", "universally", "general to any", "all systems", "proven"

Example (good):
> Structural placement consistently prevents merge conflicts across both simple and complex task types (N=80, 2 task families).

### validated

**Allowed:** Strongest language, including generality claims, if external evidence supports them
**Still requires scope:** Even validated claims should state what was validated and by whom

Example (good):
> Gate-based coordination fails and attractor-based coordination works — confirmed across 6 independent external frameworks plus 100 controlled trials.

---

## Overclaim Indicators

The provenance audit flags these patterns when they appear below the tier that warrants them:

| Pattern | Minimum tier | Why |
|---------|-------------|-----|
| "fundamentally" | validated | Implies deep/universal truth |
| "universally" | validated | Implies no exceptions |
| "always" / "never" | replicated | Absolute claims need multiple contexts |
| "proven" | validated | Scientific standard |
| "completely" / "100% reliability" | replicated | Absolute outcomes need replication |
| "general to any" / "all systems" | validated | Generalization needs external evidence |
| "inherently" / "impossible" | validated | Implies logical necessity |

---

## Integration with Provenance Audit

`orch kb provenance` (backed by `pkg/kbmetrics/provenance.go`) automatically:

1. **Detects unannotated claims** — claims missing `**Evidence quality:**`
2. **Classifies tier** — maps annotation text to the tier hierarchy
3. **Detects drift** — flags claims where prose uses language stronger than declared tier
4. **Finds orphan contradictions** — probes that contradict claims but model wasn't updated

Drift flags appear in the provenance report as:
```
  Drift: 2 claims exceed declared tier
    L10  [observed] Multi-agent frameworks that rely on messaging are fund... (triggers: fundamentally)
    L25  [hypothesis] This pattern universally applies to all coordinatio... (triggers: universally, all systems/frameworks)
```

---

## When Writing New Claims

1. Write the finding at its actual evidence scope — don't generalize
2. Add `**Evidence quality:** <tier> (<details>).` immediately after
3. Use language calibrated to the tier (see table above)
4. Run `orch kb provenance` to check for drift before committing

## When Reviewing Existing Claims

If a claim's prose exceeds its declared tier, you have two options:
1. **Scope the language down** to match the tier (preferred — most claims are overclaimed, not under-evidenced)
2. **Upgrade the tier** if new evidence justifies it (must cite the evidence)

Do NOT upgrade the tier to match existing overclaimed language. The tier follows the evidence, not the prose.
