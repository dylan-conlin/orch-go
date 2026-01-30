---
status: active
---

# Decision: Recommendation Authority Classification and Question Generation

**Date:** 2026-01-30
**Status:** Active
**Decision:** Investigation and architect recommendations must be classified by authority level, and architects must produce explicit Question Generation output for unresolved forks.

## Context

Investigation recommendations were sitting unactioned because:
1. No classification of recommendation authority (worker/orchestrator/Dylan)
2. Architects complete without surfacing gate-level decisions explicitly
3. No "what questions should we be asking" phase

This created a failure mode where 237 investigations had actionable recommendations, but many remained stranded - no beads issue created, no implementation done, no explicit deferral.

## Standards

### Standard 1: Authority Classification

All investigation and architect recommendations must specify authority level using labels:

| Authority Level | Label | Who Decides | Criteria |
|-----------------|-------|-------------|----------|
| **Implementation** | `authority:implementation` | Worker (within scope) | Reversible, single-scope, clear criteria, no cross-boundary impact |
| **Architectural** | `authority:architectural` | Orchestrator | Cross-boundary, multiple valid approaches, requires synthesis across contexts |
| **Strategic** | `authority:strategic` | Dylan | Irreversible, resource commitment, value judgment, premise-level question |

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → `authority:implementation`
- Reaches to other components/agents → `authority:architectural`  
- Reaches to values/direction/irreversibility → `authority:strategic`

### Standard 2: Authority × Subtype Matrix

Authority is orthogonal to question subtype. Use both dimensions:

| | `subtype:factual` | `subtype:judgment` | `subtype:framing` |
|---|---|---|---|
| **`authority:implementation`** | "What's the current timeout value?" (worker checks code) | "Should timeout be 30s or 60s?" (worker decides within scope) | N/A - framing implies strategic |
| **`authority:architectural`** | "How does auth flow across services?" (spawn investigation) | "Should we use JWT or sessions?" (orchestrator synthesizes tradeoffs) | "Is auth the right abstraction?" (orchestrator may escalate) |
| **`authority:strategic`** | N/A - factual doesn't require Dylan | "Accept eventual consistency tradeoff?" (Dylan judges value) | "Should we build this at all?" (Dylan reframes direction) |

**Note:** `authority:strategic + subtype:framing` is what the decidability graph calls a "Gate" - but we don't create a separate entity type. Use `--type question` with these labels; gate-ness is inferred.

### Standard 3: Question Generation Output

Architects must produce explicit Question Generation output when forks cannot be navigated with available substrate.

**Format:**

```markdown
## Blocking Questions

> **Hard cap: 3-7 questions maximum.** If you have more, you're either bikeshedding or the scope is too large.

### Q1: [Question text]
- **Authority:** implementation | architectural | strategic
- **Subtype:** factual | judgment | framing
- **What changes based on answer:** [How this affects the design]

### Q2: ...
```

**Triggering criteria:** A fork is "unnavigable" when:
- Substrate consultation (principles, models, decisions) doesn't provide enough context
- Multiple valid approaches exist with unclear tradeoffs
- The question is about premise, not implementation

**Outputs:**
- Questions create beads entities: `bd create --type question "[text]" -l authority:X -l subtype:Y`
- Architect documents blocking questions in investigation file
- Orchestrator wires dependencies (architects create nodes, orchestrators create edges)

### Standard 4: Artifact Complete ≠ Work Unblocked

An architect artifact can be `Status: Complete` even if it outputs unresolved questions.

- **Artifact status:** Reflects whether the architect finished their work (investigation + question surfacing)
- **Work graph status:** Reflects whether dependent work can proceed (blocked by unresolved questions)

This prevents the ambiguity where architects can't complete because questions are pending. The architect's job is to surface the questions clearly; resolution is orchestrator/Dylan's job.

## What This Enables

1. **Routing by authority** - Orchestrator can filter `authority:strategic` for Dylan review, handle `authority:architectural` directly, and let workers resolve `authority:implementation`

2. **Explicit uncertainty** - Instead of "it depends" hedges buried in prose, architects produce structured blocking questions

3. **Unblocking visibility** - `bd list -l authority:strategic --status open` shows what Dylan needs to decide

4. **Completion clarity** - Architects can complete even when questions are pending; the work graph tracks blocking, not the artifact

## What This Constrains

1. **Classification burden** - Workers/architects must classify recommendations (mitigated: clear criteria above)

2. **Question cap** - 3-7 max prevents comprehensive but paralyzing question lists

3. **No "Gate" entity type** - Use labels on questions, not new beads type (prevents taxonomy drift)

## Implementation

**Immediate (this decision):**
- Authority labeling standard is active
- Question Generation output format is required for architect sessions

**Follow-on work (separate issues):**
- Update investigation template with `Recommendation Authority:` field
- Update architect skill with Question Generation phase guidance
- Enhance `kb reflect` to surface recommendations by authority level

## References

- `.kb/models/decidability-graph.md` - Authority levels for graph traversal
- `.kb/decisions/2026-01-19-worker-authority-boundaries.md` - Workers create nodes, orchestrators create edges
- `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md` - Subtype label pattern
- `.kb/investigations/2026-01-30-inv-design-decision-authority-flow-question.md` - Design investigation
- `.kb/investigations/2026-01-30-inv-scope-unactioned-investigation-recommendations.md` - Evidence of gap
