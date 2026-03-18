# Model: Model Relationships

**Domain:** Knowledge Management / Meta-Modeling
**Last Updated:** 2026-03-18
**Synthesized From:** Decidability-graph recovery probe (2026-03-01), observation that losing the decidability graph made session-lifecycle claims feel arbitrary, discussion of what "foundational" means

---

## Summary (30 seconds)

Models serve three distinct functions: **structural** (why something is the way it is), **mechanistic** (how something works), and **taxonomic** (what kinds of things exist). These aren't model types — a single model can serve multiple functions. The decidability graph is structural (why authority exists), taxonomic (Work/Question/Gate vocabulary), and partially mechanistic (graph dynamics). Dependencies between functions explain why losing certain models has outsized impact: when a structural model disappears, mechanistic models that depend on it become rules without reasons.

---

## Core Mechanism

### Three Functions of Models

| Function | Question Answered | Example | What It Provides |
|----------|-------------------|---------|------------------|
| **Structural** | Why is it this way? | Decidability graph: authority exists because of context-scoping | Premises that other models assume |
| **Mechanistic** | How does it work? | Agent lifecycle: state flows through 4 layers | State machines, flows, failure modes |
| **Taxonomic** | What kinds exist? | Drift taxonomy: 4 drift domains | Vocabulary that other models use to be precise |

**Key distinction:** These are functions, not types. A model can serve multiple functions simultaneously. The decidability graph is the clearest example — it's structural (explains why the authority hierarchy exists), taxonomic (provides Work/Question/Gate classification), and partially mechanistic (describes graph dynamics and frontier behavior).

### Dependency Pattern

```
Structural (why)
    │ provides premises for
    ▼
Mechanistic (how)
    │ uses vocabulary from
    ▼
Taxonomic (what kinds)
```

Not strictly layered — taxonomic models can be foundational (drift taxonomy shapes reliability thinking), and some models serve all three functions. But the general flow holds: structural premises ground mechanistic descriptions, which use taxonomic vocabulary to be precise.

### How to Identify Function

**Structural test:** Remove this model. Do other models' claims now feel arbitrary? "Orchestrators comprehend" without the decidability graph is a rule you follow because it's written down. With the decidability graph, it's a consequence of context-scoping being irreducible.

**Mechanistic test:** Does this model describe state transitions, flows, or failure modes you can trace through? Agent lifecycle's four-layer state derivation is mechanistic — you can follow a specific agent through the layers.

**Taxonomic test:** Does this model provide distinctions that other models reference? Drift taxonomy's four domains (state, schema, behavioral, semantic) give the reliability model precise vocabulary.

### Multi-Function Models

| Model | Structural | Mechanistic | Taxonomic |
|-------|-----------|-------------|-----------|
| Decidability graph | Why authority hierarchy exists | Graph dynamics, frontier | Work/Question/Gate |
| Agent lifecycle | — | 4-layer state derivation | Active/idle/dead/completed |
| Drift taxonomy | Why drift is inevitable (duplicated state) | — | 4 drift domains |
| Session lifecycle | — | Session boundaries, checkpoints | Worker/orchestrator/cross-session |

Models serving multiple functions are higher-impact when lost and harder to reconstruct, because the loss is felt in multiple places simultaneously.

---

## Why This Fails

### 1. Structural Model Lost — Rules Without Reasons

**What happens:** A structural model disappears. Mechanistic models that depended on it keep working but feel arbitrary. People follow rules without understanding why.

**Evidence:** The decidability graph was removed during entropy-spiral cleanup. The session lifecycle model continued to say "orchestrators comprehend, workers implement" — but no model explained WHY. The claim became a convention rather than a structural consequence.

**Detection:** Claims in mechanistic models that start with "the rule is..." or "by convention..." without linking to structural reasoning. Also: agents violating authority boundaries more often, because the reason for the boundary isn't in context.

### 2. Taxonomic Model Lost — Blurred Distinctions

**What happens:** A taxonomic model disappears. Other models lose precision — they blur distinctions that matter operationally.

**Evidence:** Without Work/Question/Gate taxonomy, all backlog items are "issues." The daemon treats them uniformly. Questions get spawned as investigations without recognizing they need synthesis, not just evidence gathering.

**Detection:** Conversations where people say "it's kind of like X but different" — the vocabulary for the distinction is missing.

### 3. Model Serves Multiple Functions — Outsized Impact

**What happens:** A multi-function model is lost. The damage is multiplicative because multiple dependency chains break simultaneously.

**Evidence:** Losing the decidability graph broke structural understanding (why authority exists), taxonomic precision (Work/Question/Gate), AND mechanistic description (graph dynamics). The probe identified 8 concepts with zero coverage on master.

**Detection:** When losing one model creates gaps in 3+ other models, it was multi-function.

---

## Constraints

### Why Not Merge Structural Models Into Mechanistic Ones?

**Constraint:** Structural models should remain separate even when they "explain" a mechanistic model.

**Why:** Structural reasoning is reusable across mechanisms. The context-scoping insight from the decidability graph applies to session lifecycle, spawn architecture, AND daemon operation. Embedding it in one mechanistic model makes it invisible to the others.

**This enables:** One structural model grounding multiple mechanistic models
**This constrains:** More models to maintain; cross-references required

### Why Not Make All Models Serve All Functions?

**Constraint:** Models should serve the functions they naturally serve, not be forced into all three.

**Why:** Agent lifecycle is mechanistic. Trying to make it structural ("why do agents have states?") or taxonomic ("here's a classification of all agent types") would bloat it without adding insight. The natural function is the right scope.

**This enables:** Focused, maintainable models
**This constrains:** Must maintain multiple models and their cross-references

---

## Integration Points

### With kb context

`kb context` returns models alongside decisions and investigations. Understanding model function helps evaluate relevance — a structural model is relevant whenever its premises are in play, even if the specific mechanism isn't.

### With Model Recovery

When deciding what to restore from a branch cleanup, prioritize by function:
1. **Multi-function models** — highest impact (decidability graph)
2. **Structural models** — other models depend on them silently
3. **Taxonomic models** — vocabulary loss is subtle but cumulative
4. **Mechanistic models** — most self-contained, lowest restoration urgency

### With Probes

Probes test specific claims within a model. The function tells you what kind of claim you're testing:
- Structural probe: "Is context-scoping really irreducible?" (tests a premise)
- Mechanistic probe: "Does the 4-layer derivation actually resolve state disagreements?" (tests a flow)
- Taxonomic probe: "Is Work/Question/Gate exhaustive, or is there a 4th type?" (tests a classification)

---

## Representative Model Inventory (by function)

The project has 30+ models (run `ls .kb/models/` for current list). Representative examples by function:

| Model | Functions | Dependencies |
|-------|-----------|--------------|
| `decidability-graph` | Structural + Taxonomic + Mechanistic | Foundational — grounds authority assumptions in session-lifecycle, agent-lifecycle, spawn-architecture |
| `orchestrator-session-lifecycle` | Mechanistic | Depends on decidability-graph for authority premises |
| `agent-lifecycle-state-model` | Mechanistic + Taxonomic | Independent (4-layer state derivation is self-contained) |
| `drift-taxonomy` | Structural + Taxonomic | Foundational — grounds reliability reasoning |
| `spawn-architecture` | Mechanistic | Depends on decidability-graph for authority routing |
| `defect-class-taxonomy` | Taxonomic | Independent — classifies defect types |
| `coaching-plugin` | Mechanistic | Independent — describes pain-injection mechanism |
| `knowledge-accretion` | Structural + Mechanistic | Foundational — explains why knowledge compounds |

**Note:** Not all models have been classified. The three-function framework is a useful lens but has not been universally adopted as shared vocabulary across all models.

---

## Evolution

**2026-03-01:** Initial model created. Emerged from the decidability-graph recovery probe, which revealed that losing a multi-function model had outsized impact. The question "what does 'foundational' mean?" led to identifying three model functions and their dependency patterns.

**2026-03-18:** Knowledge decay probe. Core framework confirmed (structural/mechanistic/taxonomic distinctions hold). Inventory updated from 7 to representative sample of 30+ models. Noted low adoption of framework vocabulary across KB — valid lens but not widely applied. Added newer model examples (coaching-plugin, defect-class-taxonomy, knowledge-accretion).

---

## Probes

| Probe | Date | Verdict |
|-------|------|---------|
| `probes/2026-03-18-probe-knowledge-decay-verification.md` | 2026-03-18 | Core framework confirmed; inventory severely stale (7→30+ models); low adoption of framework vocabulary |

---

## References

- `.kb/models/decidability-graph/model.md` — Primary example of multi-function model
- `.kb/models/orchestrator-session-lifecycle/model.md` — Example of mechanistic model depending on structural premises
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-01-probe-decidability-graph-knowledge-recovery-assessment.md` — The probe that surfaced this pattern
