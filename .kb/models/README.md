# Models

**Purpose:** Synthesized understanding of how system components work, why they fail, and what constraints exist.

Models are where orchestrators externalize the mental models they build through direct engagement with investigations. Models create **surface area for questions** by making implicit constraints explicit.

---

## When to Create a Model

**Signals:**
- 3+ investigations on same topic converge to understanding
- You can explain "how X works" coherently (not just "how to do X")
- Multiple downstream decisions/epics will reference this understanding
- Same confusion recurs (scattered understanding needs consolidation)

**The test:** Can you explain the mechanism in 1-2 paragraphs? If yes, you understand it enough to model it.

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

## Naming Convention

`{domain}-{component}.md`

**Examples:**
- `dashboard-agent-status.md` - How dashboard calculates agent status
- `agent-lifecycle.md` - Agent state transitions and completion
- `spawn-lifecycle.md` - How spawn process works
- `beads-integration.md` - How orch ↔ beads interaction works

---

## Template

Use `TEMPLATE.md` in this directory as starting point.

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
**Principle:** `~/.kb/principles.md` - Understanding Through Engagement
**Decisions:** `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Synthesis is orchestrator work
