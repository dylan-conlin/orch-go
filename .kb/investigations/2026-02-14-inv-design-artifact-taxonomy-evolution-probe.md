## Summary (D.E.K.N.)

**Delta:** Probes should become the universal evidence-gathering primitive, retiring "investigation" as a concept. Every evidence-gathering task becomes a probe with a named target (model claim, decision rationale, code hypothesis, or assumption). This eliminates the orchestrator's "probe vs investigation?" decision tax while preserving all rigor from the investigation workflow.

**Evidence:** Analysis of 935 investigations, 11 model-scoped probes, the dual-mode investigation skill, orchestrator routing logic, probe disambiguation work (Feb 13), and Dylan's meta-orchestrator framing. The investigation skill already detected probe mode via SPAWN_CONTEXT markers — the infrastructure for universal probes partially exists.

**Knowledge:** The probe concept forces specificity ("test THIS claim") while investigations allow vagueness ("look into THIS area"). Every investigation is implicitly a probe without a named target. Making the target explicit is what improves thinking quality. The primary design constraint is Dylan's ability to THINK in the artifact names — "probe" is the better thinking tool.

**Next:** Promote to decision record. Implementation requires: kb-cli template (PROBE.md), skill rename (investigation → probe), orchestrator routing simplification, `kb create probe` command support.

**Authority:** strategic - This is an irreversible taxonomy change affecting all evidence-gathering workflows across the system. Requires Dylan's judgment on the thinking-tool tradeoff.

---

# Investigation: Design Artifact Taxonomy Evolution — Probe as Universal Primitive

**Question:** Should probes become the ONLY evidence-gathering artifact type? What taxonomy, template, location, and migration strategy best serves Dylan's ability to think in artifact names?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** Promote to decision if accepted
**Status:** Complete

**Patches-Decision:** .kb/decisions/2025-12-21-minimal-artifact-taxonomy.md (evolves Investigation from essential to archived)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/decisions/2025-12-21-minimal-artifact-taxonomy.md | extends | yes | Investigation listed as essential — this proposes archival |
| .kb/decisions/2026-01-12-models-as-understanding-artifacts.md | extends | yes | Already says "investigations = probes (temporal, narrow questions)" |
| .kb/investigations/2026-02-13-inv-disambiguate-probe-terminology-across-skills.md | extends | yes | Disambiguated probe/spike/scouting — this builds on that |
| .kb/investigations/2026-02-13-inv-expand-model-probe-awareness-beyond.md | extends | yes | Expanded probe routing to 7 skills — this goes further |
| .kb/models/PHASE4_REVIEW.md | confirms | yes | N=11 model pattern validated; probes already positioned as primary evidence input |
| .kb/models/README.md | confirms | yes | Provenance chain already says "investigations (probes)" |

---

## Findings

### Finding 1: The investigation skill already has dual-mode detection that validates the split

**Evidence:** The investigation skill (`orch-knowledge/skills/src/worker/investigation/.skillc/SKILL.md`) detects probe mode by checking SPAWN_CONTEXT for `### Models (synthesized understanding)` markers. If found → "Probe Mode" (confirmatory test against model claims). If absent → "Investigation Mode" (novel exploration). Seven skills now have this routing.

**Source:** Investigation skill source, orchestrator SKILL.md lines 263-322

**Significance:** The infrastructure for probe routing already exists. The skill isn't one thing — it's two things sharing a name. This is exactly the "conflation" that Evolve by Distinction warns about.

---

### Finding 2: The "models-as-understanding" decision already equates investigations with probes

**Evidence:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` contains: "Investigations = probes (temporal, narrow questions)". The models README provenance chain says: "Investigations (probe findings) → synthesized into → Models". The concept already exists — the naming just hasn't caught up.

**Source:** Models decision lines 16-17, README.md line 105

**Significance:** The system already treats investigations as probes in everything but name. The taxonomy evolution is recognizing what already happened.

---

### Finding 3: Dylan's framing identifies "decision tax" as the core problem

**Evidence:** Dylan's meta-orchestrator framing:
- "Should I request an investigation or a probe? Is this new or do we have a model yet?" — this decision tax should not exist
- "I should be thinking about what I want to understand, not which artifact shape to use"
- "Investigations being separate from .kb/models feels off, like they're one-offs"

**Source:** SPAWN_CONTEXT design input

**Significance:** The primary design constraint is cognitive: Dylan wants to think about WHAT to understand, not WHICH artifact to use. "Probe" forces specificity (name the target). "Investigation" allows vagueness (look into an area). The artifact name IS the thinking tool.

---

### Finding 4: Probes currently require model targets — this is the key constraint to relax

**Evidence:** Current probe template (`.orch/templates/PROBE.md`) has `**Model:** {model-name}` as a required field. Probes live in `.kb/models/{name}/probes/`. This means probes can only target model claims.

But Dylan's insight is broader: "probing against model claims OR asking what probes do we need answered before we're ready to decide." Probes should target ANY named claim — not just model claims.

**Source:** PROBE.md template, SPAWN_CONTEXT design input

**Significance:** Expanding the target space from "model claims only" to "any named claim" is what makes probes universal. The forcing function isn't "test a model claim" — it's "name what you're testing."

---

### Finding 5: Investigation template has valuable rigor that probes should absorb

**Evidence:** The investigation template (`~/.kb/templates/INVESTIGATION.md`) includes:
- D.E.K.N. summary (30-second handoff)
- Prior Work table (builds on existing knowledge)
- Structured Uncertainty (tested vs untested vs falsifiable)
- Self-review checklist

The probe template (`PROBE.md`) has only: Question, What I Tested, What I Observed, Model Impact.

**Source:** Both templates read in full

**Significance:** The probe template's leanness is a feature (forces focus), but it needs D.E.K.N. for handoff and Prior Work for knowledge accumulation. The merged template should be "probe-lean with investigation-rigor essentials."

---

### Finding 6: 935 investigations cannot and should not be migrated

**Evidence:** `.kb/investigations/` contains 935 files dating from late 2025 through today. These are referenced by models, decisions, guides, and each other via file paths. Mass renaming would break thousands of cross-references.

**Source:** `ls .kb/investigations/ | wc -l` → 935

**Significance:** The migration strategy must be coexistence, not mass rename. `.kb/investigations/` freezes as a read-only archive. New evidence-gathering work goes to `.kb/probes/`. `kb context` searches both.

---

## Synthesis

**Key Insights:**

1. **The conflation is investigation = two things** — Evolve by Distinction says: when problems recur, ask "what are we conflating?" Investigations conflate novel exploration (no target) with confirmatory testing (model target). But Dylan's deeper insight is that even "novel exploration" implicitly probes assumptions — you just haven't named the target yet. The fix: make the target naming explicit and universal.

2. **"Probe" is the better thinking tool** — The artifact name shapes how the orchestrator thinks about work. "Spawn an investigation" invites open-ended exploration. "Spawn a probe" demands: "probe WHAT?" The name itself is the forcing function for specificity. This is the primary design constraint.

3. **The provenance chain doesn't change, just the vocabulary** — Currently: investigations → models → decisions. New: probes → models → decisions. Same chain, better name. The chain already says "investigations (probe findings)" — the rename aligns naming with reality.

4. **Target flexibility eliminates decision tax** — When probes can target model claims, decisions, code hypotheses, OR assumptions, the orchestrator never needs to ask "is this a probe or investigation?" Everything is a probe. The target type provides the specificity that the probe/investigation routing used to provide.

**Answer to Investigation Question:**

YES — probes should become the universal evidence-gathering primitive, with investigations retired as a concept. The key design move is expanding probe targets from "model claims only" to "any named claim" (model, decision, code hypothesis, assumption). This eliminates the orchestrator's decision tax while preserving rigor through a merged template. Migration is coexistence: `.kb/probes/` for new work, `.kb/investigations/` frozen as archive.

---

## Structured Uncertainty

**What's tested:**

- ✅ Investigation skill already has dual-mode detection (probe mode vs investigation mode) — infrastructure exists
- ✅ 7 skills already have probe routing via SPAWN_CONTEXT markers — pattern is portable
- ✅ Probe terminology disambiguation completed (probe/spike/scouting) — no conflation remaining
- ✅ Models decision already equates investigations with probes — conceptual alignment exists
- ✅ D.E.K.N. + Prior Work + Structured Uncertainty are valuable additions to probe template — investigation rigor demonstrated over 935 uses

**What's untested:**

- ⚠️ Whether agents can produce good probes against non-model targets (assumptions, hypotheses) — current probes only target models
- ⚠️ Whether the merged template stays lean enough to force focus — risk of template bloat
- ⚠️ Whether `kb context` can efficiently search both `.kb/probes/` and `.kb/investigations/` — needs kb-cli verification
- ⚠️ Whether the skill rename (investigation → probe) cascades cleanly through skillc, SPAWN_CONTEXT, orchestrator routing

**What would change this:**

- If agents produce vague probes against vague targets ("I assume something about X"), the forcing function fails — would need stronger target validation
- If the merged template grows to investigation size, the leanness benefit is lost — would need template governance
- If Dylan finds "probe" doesn't map to how he thinks about novel exploration — would need to reconsider the name

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Retire "investigation" as concept, make "probe" universal | strategic | Irreversible taxonomy change affecting all evidence-gathering workflows |
| Merged probe template design | architectural | Cross-component (kb-cli template, orch-go spawn, skill system) |
| Coexistence migration strategy | implementation | Minimal-disruption approach within existing patterns |
| Skill rename (investigation → probe) | architectural | Cross-skill change affecting orchestrator routing |

### Recommended Approach ⭐

**Probe as Universal Evidence-Gathering Primitive** — Every evidence-gathering task becomes a probe with a named target. Investigations retire as a concept. Coexistence migration.

**Why this approach:**
- Eliminates decision tax (orchestrator always spawns "probe")
- Forces specificity through named targets (the thinking-tool benefit)
- Preserves rigor by absorbing investigation essentials into probe template
- Minimal disruption through coexistence (no mass migration)

**Trade-offs accepted:**
- Two directories to search (`.kb/probes/` + `.kb/investigations/` archive) — acceptable because `kb context` handles discovery
- Skill rename cascades through multiple files — acceptable as one-time migration
- Existing 11 model-scoped probes in `.kb/models/{name}/probes/` become legacy — acceptable as they're a small set

**Implementation sequence:**

1. **Decision record** — Capture this design as accepted decision (this document → `.kb/decisions/`)
2. **Merged probe template** — Create `~/.kb/templates/PROBE.md` in kb-cli with Target + D.E.K.N. + Prior Work + Structured Uncertainty
3. **`kb create probe` command** — Add to kb-cli alongside existing `kb create investigation`
4. **Create `.kb/probes/` directory** — New home for all evidence-gathering work
5. **Skill rename** — `investigation` → `probe` in orch-knowledge skill sources, update skillc
6. **Orchestrator routing simplification** — Remove "Probe vs Investigation Boundary" table, replace with "always spawn probe"
7. **Update global CLAUDE.md** — Knowledge Placement table: change "Exploration/analysis → .kb/investigations/" to "Evidence-gathering → .kb/probes/"
8. **Freeze `.kb/investigations/`** — Document as read-only archive, update `kb context` to search both

### Alternative Approaches Considered

**Option B: Keep investigations, make probes a sub-type**
- **Pros:** No migration needed, backward compatible
- **Cons:** Preserves the decision tax ("is this an investigation or a probe?"), doesn't solve Dylan's core problem
- **When to use instead:** If the thinking-tool benefit of "probe" turns out to be illusory

**Option C: Rename .kb/investigations/ to .kb/probes/ in-place**
- **Pros:** Single directory, clean namespace
- **Cons:** Breaks 935+ cross-references, massive git churn, high risk
- **When to use instead:** Never — the cross-reference breakage is unacceptable

**Option D: All probes under .kb/models/{name}/probes/ (model-centric)**
- **Pros:** Probes feel connected to models (Dylan's complaint about "one-offs")
- **Cons:** Non-model probes have no home, reintroduces routing ("which model?")
- **When to use instead:** If probes end up being exclusively model-targeted in practice

**Rationale for recommendation:** Option A uniquely eliminates decision tax while preserving rigor. The coexistence strategy avoids the catastrophic risk of Options C/D while achieving the cognitive benefit of the "probe" name.

---

### Implementation Details

**What to implement first:**
- Decision record (this document) → `.kb/decisions/`
- Merged probe template in kb-cli

**Things to watch out for:**
- ⚠️ The probe template must stay LEAN — if it grows to investigation size, the forcing function weakens
- ⚠️ `kb context` must be updated to search `.kb/probes/` in addition to `.kb/investigations/`
- ⚠️ The skill rename cascades through: investigation source (.skillc), deployed SKILL.md, orchestrator SKILL.md, CLAUDE.md (both global and project), spawn prompts, all 7 skills with probe routing

**Areas needing further investigation:**
- Whether `kb create probe` should generate the Target field interactively or from CLI args
- Whether existing `.kb/models/{name}/probes/` directories should be deprecated or kept as aliases
- Whether the Prior Work table in probe template should reference both probes AND old investigations

**Success criteria:**
- ✅ Orchestrator never faces "probe or investigation?" decision
- ✅ Every new evidence-gathering artifact has a named target
- ✅ `kb context` returns both probes and archived investigations
- ✅ Investigation skill renamed to probe, all skills updated
- ✅ Dylan reports improved thinking clarity when requesting evidence-gathering work

---

## Design: Merged Probe Template

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
- ✅ [Claim with evidence]

**What's untested:**
- ⚠️ [Hypothesis without validation]

**What would change this:**
- [Falsifiability criteria]
```

### Template Design Rationale

**Absorbed from investigations:** D.E.K.N. (handoff), Prior Work (knowledge accumulation), Structured Uncertainty (epistemic rigor)

**Kept from probes:** Target (forcing function), What I Tested / What I Observed (lean structure), Impact (model connection)

**Dropped from investigations:** Multi-finding structure (Finding 1/2/3 with Evidence/Source/Significance), Implementation Recommendations section, Investigation History section — these add bulk without proportional value for probe-scoped work. Agents producing complex probes can add findings sections ad hoc.

### Probe Target Types

| Target Type | Example | When to Use |
|---|---|---|
| **Model claim** | "completion-verification model claims three gates catch real defects" | Model exists, testing a specific claim |
| **Decision rationale** | "minimal artifact taxonomy says investigations are essential" | Decision exists, testing whether rationale still holds |
| **Code hypothesis** | "spawn_cmd.go handles concurrent spawns without race conditions" | Testing specific code behavior |
| **Assumption** | "I assume the OpenCode API returns paginated sessions" | Novel exploration — naming the implicit assumption |

**The forcing function:** Even "novel exploration" requires naming an assumption. "I want to understand X" becomes "I assume Y about X — let me test that." The target IS the specificity.

---

## References

**Files Examined:**
- `.orch/templates/PROBE.md` — Current probe template (lightweight)
- `~/.kb/templates/INVESTIGATION.md` — Current investigation template (heavy)
- `orch-knowledge/skills/src/worker/investigation/.skillc/SKILL.md` — Investigation skill (dual-mode)
- `~/.claude/skills/meta/orchestrator/SKILL.md:263-322` — Probe vs investigation routing
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` — Lists investigation as essential
- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` — Already equates investigations with probes
- `.kb/models/README.md` — Provenance chain
- `.kb/models/PHASE4_REVIEW.md` — N=11 model pattern validation
- `~/.kb/principles.md` — Evolve by Distinction, Premise Before Solution
- `.kb/investigations/2026-02-13-inv-disambiguate-probe-terminology-across-skills.md` — Probe/spike/scouting
- `.kb/investigations/2026-02-13-inv-expand-model-probe-awareness-beyond.md` — Probe routing expansion

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` — This decision patches it
- **Decision:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` — Conceptual predecessor
- **Model:** `.kb/models/README.md` — Provenance chain documentation
