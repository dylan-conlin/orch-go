# Models

**Purpose:** Synthesized understanding of how system components work, why they fail, and what constraints exist.

Models are where orchestrators externalize the mental models they build through direct engagement with investigations. Models create **surface area for questions** by making implicit constraints explicit.

---

## When to Create a Model

**Creation threshold:** 3+ investigations on a single topic.

Models are "building blocks of understanding" — composable context that orchestrators and architects combine to frame design work. The threshold is intentionally low because models gain value through use, not through waiting.

**Three-factor test (all required):**
1. **CONVERGING** - 3+ investigations on the topic show a coherent mechanism emerging
2. **COMPLEX** - Has failure modes, constraints, state transitions (not just a single fact)
3. **REUSABLE** - You'd point an orchestrator or architect at this to set up a design conversation

**The test:** Can you explain the mechanism in 1-2 paragraphs, and would combining this with 1-2 other models give someone enough context to make design decisions?

**Anti-pattern:** Creating a model from a single investigation. One investigation is a finding. Three investigations showing the same mechanism from different angles is understanding worth externalizing.

---

## Boundary Tests

Models answer **"How does X work?"** and **"Why does X fail?"**

| Question | Artifact Type |
|----------|---------------|
| "How does dashboard status work?" | **Model** (here) |
| "How do I debug completion?" | **Guide** (`.kb/guides/`) |
| "Why did we choose Priority Cascade?" | **Decision** (`.kb/decisions/`) |
| "What work remains?" | **Epic** (beads) |
| "Does X behave this way?" | **Investigation** (`.kb/investigations/`) |

**Key distinction:**
- Models are **descriptive** (how system IS)
- Guides are **prescriptive** (how to DO)
- Decisions are **choices** (what we CHOSE)

---

## Lifecycle

```
Multiple investigations (probes)
    ↓
Orchestrator synthesizes (direct engagement)
    ↓
Model created (understanding externalized)
    ↓
Decisions/Guides/Epics reference model
    ↓
Model evolves (as system changes)
```

**Models are living documents** - update them when understanding changes, don't create new versions.

---

## Directory Convention

Each model is a directory containing `model.md` and optionally `probes/`:

```
.kb/models/
  {domain}-{component}/
    model.md              ← the model content
    probes/               ← evidence gathered against this model
      2026-02-25-probe-description.md
```

**Examples:**
- `dashboard-architecture/model.md` - How dashboard works
- `spawn-architecture/model.md` - How spawn process works
- `completion-verification/model.md` - How completion gates work

**Why directories:** Navigable with `ls`, co-locates model with its evidence (probes), consistent structure across all models.

---

## Template

Use `TEMPLATE.md` in this directory as a starting point. To create a new model:

```bash
mkdir -p .kb/models/{name}/probes
cp .kb/models/TEMPLATE.md .kb/models/{name}/model.md
```

**Required sections:**
- Summary (30-second overview)
- Core Mechanism (how it works)
- Why This Fails (failure modes)
- Constraints (limits and boundaries)
- References (provenance chain)

**Why constraints matter:** Making constraints explicit creates surface area for strategic questions.

Example: Model states "OpenCode doesn't expose session state via HTTP API" → enables question "Should we add that endpoint?"

---

## Provenance Chain

Models are **nodes in provenance chains, not endpoints.**

```
Primary evidence (code, tests, behavior)
    ↓ (referenced in)
Investigations (probe findings)
    ↓ (synthesized into)
Models (understanding)
    ↓ (inform)
Decisions (choices)
    ↓ (create)
Guides, Epics (downstream work)
```

**Critical:** Models must reference investigations (via "Synthesized From" section). Investigations must reference code. Chain terminates in observable reality.

Models without provenance are closed loops (violates Provenance principle).

---

## Relationship to Principles

### Understanding Through Engagement

Models are the artifact type that "Understanding Through Engagement" principle produces.

**The principle:** You can spawn work to gather facts (investigations), but synthesis into coherent models requires the cross-agent context that only orchestrator has.

**The artifact:** Models are where that synthesis lives.

**Why not spawnable:** Synthesis requires seeing patterns across multiple investigations. Only orchestrator has that vantage point.

### Evidence Hierarchy

Models are secondary evidence (like investigations and decisions). They must trace to primary evidence (code).

Trust code over models. When model and code conflict, code wins. Update the model.

---

## Success Criteria

**How we know models work:**

1. ✅ Orchestrators create models after synthesizing 3+ investigations
2. ✅ Dylan asks sharper questions because constraints are explicit
3. ✅ Decisions reference models for context
4. ✅ Duplicate investigations decrease (model answers the question)
5. ✅ Epic readiness increases (model = understanding achieved)

**If models don't get created:** Process isn't working, need to revisit.

**If models get created but not referenced:** Discoverability problem, need to fix `kb context` integration.

---

## Related Artifacts

**Decision:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Why models exist
**Principle:** `~/.kb/principles.md` (symlink → `~/orch-knowledge/kb/principles.md`) - Understanding Through Engagement
**Decisions:** `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Synthesis is orchestrator work
