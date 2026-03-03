# Operationalize Models in Orchestrator Skill

**Date:** 2026-01-12
**Status:** Active
**Impact:** Medium - Changes orchestrator workflow when encountering investigation clusters

---

## Context

After creating 11 models (Jan 12, 2026) that synthesized 133+ investigations into queryable understanding, we validated the pattern empirically:

- ✅ 100% structural consistency (6-section pattern across all 11)
- ✅ Enable/constrain query works (Dylan's questions answered in <60s vs hours)
- ✅ Four-factor emergence pattern explains what got modeled (HOT × COMPLEX × OWNED × STRATEGIC_VALUE)
- ✅ Models compose (orchestrator tier query used 3 models)

**The question:** Are models ready to operationalize in orchestrator skill, or still too experimental?

---

## Decision

**Operationalize conservatively** - Add models guidance to orchestrator skill with explicit "NEW" status and 30-day validation window.

**What gets added:**
1. When to create models (four-factor test)
2. When to consult models (before spawning, for strategic questions)
3. Model template structure (6 sections)
4. Tracking expectations (30-day validation)
5. Anti-patterns (what NOT to model)

**What does NOT get added:**
- ❌ World model validation metrics (unproven)
- ❌ Meta-models (speculative)
- ❌ Automatic staleness detection (unknown behavior)

---

## Rationale

**Why conservative approach:**
- Pattern is proven at N=11 but boundaries unknown (at what N do diminishing returns hit?)
- Staleness behavior untested (how often do models need updating?)
- Meta-model hypothesis interesting but no evidence yet

**Why operationalize now:**
- Enough empirical validation (N=11, structural consistency, query effectiveness)
- Next orchestrator shouldn't re-investigate spawn (87 investigations → model)
- Four-factor test prevents over-modeling (gates creation on all factors)
- 30-day tracking built in (will surface if assumptions wrong)

**The conservative framing:**
- Explicit "NEW - Jan 2026" status
- "Still establishing boundaries" language
- Tracking expectation with end date (Feb 12, 2026)
- Failure signals enumerated (know when to reevaluate)

---

## Implementation

**File modified:** `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template`

**Section added:** "Models as Understanding Artifacts (NEW - Jan 2026)" after "Synthesis is Orchestrator Work"

**Fast Path entry added:** "Investigation cluster (15+) → Check four factors → spawn architect to create model"

**Deployment:** `skillc deploy --target ~/.claude/skills/` (Jan 12, 2026)

---

## Validation Criteria (30 Days)

**Success signals:**
- Investigation spawn rate drops 30-50% on modeled topics
- Models referenced in new investigations
- Dylan's strategic questions answered quickly from models
- Model Evolution sections grow (system living, not static)

**Failure signals:**
- Investigation rate unchanged
- Models never referenced
- Evolution sections empty after 60+ days

**Review date:** Feb 12, 2026

If failure signals appear, options:
1. Tighten four-factor test (raise thresholds)
2. Adjust template (current structure doesn't work)
3. Abandon pattern (doesn't scale as hypothesized)

---

## Trade-offs

**Accepted:**
- Risk of over-modeling (mitigated by four-factor gate)
- Uncertainty about staleness (will learn empirically)
- Unknown scale limits (tracking will surface)

**Rejected:**
- Waiting for more evidence before operationalizing (next orchestrator needs this now)
- Adding unvalidated metrics (world model parallels interesting but untested)
- Automating model creation (synthesis requires orchestrator engagement)

---

## Alternative Considered: Wait Until N=20+

**Argument:** More data before operationalizing

**Why rejected:**
- N=11 already shows structural consistency
- Four-factor pattern retrospectively explains all 11
- Next orchestrator shouldn't re-investigate what's already modeled
- 30-day tracking will catch issues faster than waiting for N=20

---

## Relationship to Principles

**Understanding Through Engagement:** Models are synthesis outputs, not spawnable work. Orchestrator must create them directly.

**Session Amnesia:** Models externalize understanding so next orchestrator doesn't re-investigate.

**Friction is Signal:** Investigation clusters (friction) signal missing models.

**Pressure Over Compensation:** Don't manually answer strategic questions repeatedly - externalize to model.

**Evolve by Distinction:** Models are distinct from guides (process), decisions (choices), investigations (point-in-time).

---

## References

- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Artifact taxonomy
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Synthesis ownership
- `orch-go/.kb/models/PHASE4_REVIEW.md` - N=11 validation review
- Session summary (Jan 12, 2026) - 11 models created, pattern validated
