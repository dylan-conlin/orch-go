# Decision: Probe as Universal Evidence-Gathering Primitive

**Status:** Proposed
**Date:** 2026-02-14
**Deciders:** Architect agent (proposed), Dylan (approval required)
**Context:** Meta-orchestrator identified "probe vs investigation?" decision tax as friction. Core insight: probe forces specificity, investigation allows vagueness. Every investigation is a probe without a named target.

**blocks:** probe, investigation, evidence-gathering, artifact taxonomy, understanding work

---

## Summary

**Probes become the universal evidence-gathering artifact type.** Investigations retire as a concept. Every evidence-gathering task becomes a probe with a named target — model claim, decision rationale, code hypothesis, or assumption. The "investigation" skill renames to "probe". The `.kb/investigations/` directory freezes as a read-only archive. New evidence-gathering work goes to `.kb/probes/`.

---

## Decision

### The Taxonomy Change

| Before | After |
|--------|-------|
| "Investigation" = default evidence-gathering | "Probe" = universal evidence-gathering |
| Probes only target model claims | Probes target any named claim |
| Orchestrator routes: "probe or investigation?" | Orchestrator always spawns probe |
| Two templates (INVESTIGATION.md + PROBE.md) | One merged template |
| `.kb/investigations/` = active directory | `.kb/investigations/` = read-only archive |
| `.kb/models/{name}/probes/` = model-scoped | `.kb/probes/` = universal home |

### What Probe Targets

| Target Type | Example | Replaces |
|---|---|---|
| **Model claim** | "completion-verification model claims three gates catch real defects" | Current model probe |
| **Decision rationale** | "minimal artifact taxonomy says investigations are essential" | Decision-patching investigation |
| **Code hypothesis** | "spawn_cmd.go handles concurrent spawns without race conditions" | Code-focused investigation |
| **Assumption** | "I assume the OpenCode API returns paginated sessions" | Novel investigation |

**The forcing function:** Every probe has a named target. Even "novel exploration" requires naming an assumption. "I want to understand X" becomes "I assume Y about X — let me test that." The target IS the specificity.

### Merged Probe Template

The new template absorbs investigation rigor into probe leanness:

```markdown
# Probe: {title}

**Target:** {what you're testing — model claim, decision rationale, code hypothesis, or assumption}
**Source:** {path to model/decision/code, or "Novel — no prior artifact"}
**Date:** {date}
**Status:** Active

---

## D.E.K.N. Summary

**Delta:** [What was discovered]
**Evidence:** [Primary evidence]
**Knowledge:** [What was learned]
**Next:** [Recommended action]

---

## Prior Work

| Probe/Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| [path or "N/A — novel probe"] | [extends/confirms/contradicts/deepens] | [yes/pending] | [description] |

---

## Question

[What specific claim, hypothesis, or assumption are you testing?]

---

## What I Tested

[Command run, code examined, or experiment performed — not just code review]

## What I Observed

[Actual output, behavior, or evidence gathered]

---

## Impact

- [ ] **Confirms:** [what claim/hypothesis]
- [ ] **Contradicts:** [what claim/hypothesis] — [what's actually true]
- [ ] **Extends:** [new finding not covered by existing knowledge]

---

## Structured Uncertainty

**What's tested:**
- [Claim with evidence]

**What's untested:**
- [Hypothesis without validation]

**What would change this:**
- [Falsifiability criteria]
```

**Template design rationale:**
- **Absorbed from investigations:** D.E.K.N. (handoff), Prior Work (knowledge accumulation), Structured Uncertainty (epistemic rigor)
- **Kept from probes:** Target (forcing function), What I Tested / What I Observed (lean structure), Impact (claim connection)
- **Dropped from investigations:** Multi-finding structure (Finding 1/2/3), Implementation Recommendations, Investigation History — bulk without proportional value

### File Location

- **New probes:** `.kb/probes/` (created by `kb create probe {slug}`)
- **Archived investigations:** `.kb/investigations/` (read-only, still searchable by `kb context`)
- **Legacy model probes:** `.kb/models/{name}/probes/` (frozen, new model-targeted probes go to `.kb/probes/` with `**Source:**` linking to the model)

### Updated Provenance Chain

```
Primary evidence (code, tests, behavior)
    ↓ (referenced in)
Probes (evidence-gathering findings)          ← renamed from "investigations"
    ↓ (synthesized into)
Models (understanding)
    ↓ (inform)
Decisions (choices)
    ↓ (create)
Guides, Epics (downstream work)
```

### Updated Knowledge Placement

| You have... | Put it in... | Trigger |
|-------------|--------------|---------|
| Evidence-gathering | `.kb/probes/` | "Test whether X is true" / "Understand how X works" |
| Quick decision | `kb quick decide` | "We chose X because Y" |
| Significant decision | `.kb/decisions/` | Architectural, cross-project |
| Rule/constraint | `kb quick constrain` | "Never do X" / "Always do Y" |
| Failed approach | `kb quick tried` | "X didn't work because Y" |
| Reusable framework | `.kb/guides/` | "How should I approach X?" |

### Updated Boundary Tests

| Question | Artifact Type |
|----------|---------------|
| "Does X behave this way?" | **Probe** (`.kb/probes/`) |
| "How does X work?" | **Model** (`.kb/models/`) |
| "How do I do X?" | **Guide** (`.kb/guides/`) |
| "What did we choose?" | **Decision** (`.kb/decisions/`) |
| "What work remains?" | **Epic** (beads) |

---

## Context

### The Problem: Decision Tax

Dylan (meta-orchestrator) identified a recurring friction:

> "Should I request an investigation or a probe? Is this new or do we have a model yet?" — this decision tax should not exist.
> "I should be thinking about what I want to understand, not which artifact shape to use."

The orchestrator skill has a "Probe vs Investigation Boundary" table and "Spawn-Time Probe vs Investigation Routing (Required)" section. This routing forces the orchestrator to determine whether a model exists before spawning evidence-gathering work. The routing is correct but creates cognitive overhead.

### The Insight: Probe Forces Specificity

> "Investigations don't offer that granularity."
> "I genuinely value the probe concept because it helps me think in terms of probing against model claims or asking what probes do we need answered before we're ready to decide."

The core insight: "probe" demands a target ("probe WHAT?"), while "investigation" allows vagueness ("look into X"). Every investigation is really a probe without a named target. Making the target explicit improves thinking quality.

### The Evolution: Already Happening

The models-as-understanding decision (2026-01-12) already says: "Investigations = probes (temporal, narrow questions)." The models README provenance chain already says: "Investigations (probe findings)." The investigation skill already has dual-mode detection (probe mode vs investigation mode). This decision aligns naming with what the system already recognizes.

---

## Options Considered

### Option A: Probe as Universal Primitive (Chosen) ⭐

**Every evidence-gathering task is a probe.** Investigations retire. Targets expand from model claims to any named claim.

**Pros:**
- Eliminates decision tax completely
- Forces specificity through named targets
- Simplifies orchestrator routing to "always spawn probe"
- Aligns naming with what system already recognizes
- Lean template preserves probe's focus-forcing quality

**Cons:**
- Two directories to search (`.kb/probes/` + `.kb/investigations/` archive)
- Skill rename cascades through multiple files
- 935 existing investigations don't retroactively gain targets
- Agents must learn to name targets for novel exploration

### Option B: Keep Investigations, Make Probes Sub-Type

**Investigations remain. Probes are a specialized sub-type when models exist.**

**Pros:** No migration, backward compatible

**Cons:** Preserves the decision tax, doesn't solve Dylan's core problem, perpetuates conflation

**Rejected:** Doesn't address the stated problem.

### Option C: Rename .kb/investigations/ to .kb/probes/ In-Place

**Mass rename all existing files.**

**Pros:** Single clean directory

**Cons:** Breaks 935+ cross-references in models, decisions, guides, and other investigations. Massive git churn.

**Rejected:** Catastrophic risk with no proportional benefit.

### Option D: All Probes Under Models (Model-Centric)

**Every probe lives under `.kb/models/{name}/probes/`.**

**Pros:** Probes feel connected to models (addresses Dylan's "one-offs" complaint)

**Cons:** Non-model probes have no home. Reintroduces routing ("which model does this target?"). Novel exploration probes don't have a model yet.

**Rejected:** Reintroduces routing decision tax in different form.

---

## Consequences

### Positive

- Orchestrator thinks in probes, not artifact types — improved cognitive flow
- Every evidence-gathering artifact has a named target — improved specificity
- Provenance chain strengthened (probes always reference what they target)
- Routing simplified (orchestrator skill loses "Probe vs Investigation" routing)
- Skill system cleaner (one skill, one name, one template)

### Negative

- Two-directory search (`.kb/probes/` + `.kb/investigations/`) until old investigations naturally become irrelevant
- One-time migration cost for skill rename across orch-knowledge, orchestrator, CLAUDE.md
- 935 old investigations lack Target field — legacy artifacts won't have the forcing function
- New concept needs testing: "probe against an assumption" is new territory for agents

### Risks

- **Template bloat risk:** If the merged probe template grows beyond its current lean design, the forcing function weakens. Mitigation: template governance — any addition requires justification.
- **Target quality risk:** Agents might produce vague targets ("I assume something about X"). Mitigation: self-review checklist includes "Is the target specific enough to be testable?"
- **Discovery risk:** `kb context` must search both `.kb/probes/` and `.kb/investigations/`. If it doesn't, archived investigations become invisible. Mitigation: verify kb-cli search paths.

---

## Implementation Plan

### Phase 1: Foundation (kb-cli)
1. Create `~/.kb/templates/PROBE.md` with merged template
2. Add `kb create probe {slug}` command to kb-cli
3. Ensure `kb context` searches `.kb/probes/` directory
4. Create `.kb/probes/` directory in each active project

### Phase 2: Skill System (orch-knowledge)
5. Rename `investigation` skill to `probe` in `.skillc` sources
6. Update skill content to use probe terminology and merged template
7. Remove probe-mode vs investigation-mode branching (everything is probe)
8. Update all 7 skills with probe routing to use simplified logic
9. `skillc deploy` to push changes

### Phase 3: Orchestrator Routing (orch-knowledge + orch-go)
10. Remove "Probe vs Investigation Boundary" table from orchestrator SKILL.md
11. Remove "Spawn-Time Probe vs Investigation Routing" section
12. Replace with simple "always spawn probe" guidance
13. Update quick decision tree: `UNDERSTAND → probe (any target)`

### Phase 4: Documentation (global + project CLAUDE.md)
14. Update global CLAUDE.md Knowledge Placement table
15. Update project CLAUDE.md boundary tests
16. Update models README provenance chain (remove "investigations" parenthetical)
17. Document `.kb/investigations/` as read-only archive

### Phase 5: Cleanup
18. Deprecate `.orch/templates/PROBE.md` (replaced by `~/.kb/templates/PROBE.md`)
19. Deprecate `.kb/models/{name}/probes/` as a write destination (existing files stay)
20. Update `orch spawn` to use "probe" terminology in output messages

---

## Decision Gate Guidance

**Add `blocks:` frontmatter when promoting:**

This decision resolves recurring confusion about probe vs investigation routing. It establishes constraints future agents might violate (trying to create "investigations" in the old style).

**Suggested blocks keywords:**
- `probe`, `investigation`, `evidence-gathering`
- `artifact taxonomy`, `understanding work`

---

## Related

- **Patches:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` — Investigation evolves from essential to archived
- **Extends:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` — Completes the "investigations = probes" recognition
- **Investigation:** `.kb/investigations/2026-02-14-inv-design-artifact-taxonomy-evolution-probe.md` — Full design analysis
- **Principle:** Evolve by Distinction — "What are we conflating?" → investigation = probe + target
- **Principle:** Premise Before Solution — Validated "should we retire investigations?" before "how"
- **Prior work:** `.kb/investigations/2026-02-13-inv-disambiguate-probe-terminology-across-skills.md` — Probe/spike/scouting disambiguation
- **Prior work:** `.kb/investigations/2026-02-13-inv-expand-model-probe-awareness-beyond.md` — Probe routing expanded to 7 skills
