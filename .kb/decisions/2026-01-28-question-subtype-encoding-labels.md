## Summary (D.E.K.N.)

**Delta:** Question subtypes (factual/judgment/framing) are encoded using labels with convention `subtype:{factual|judgment|framing}` - zero schema changes, leverages existing beads infrastructure.

**Evidence:** Beads schema has `Labels []string` field; daemon already filters by labels via `issue.HasLabel()`; `bd ready --label subtype:factual` works today.

**Knowledge:** Labels provide the flexibility needed since questions can evolve (factual -> framing) during resolution. Convention over enforcement matches beads design philosophy.

**Next:** Document convention in CLAUDE.md, update decidability-graph model, optionally extend daemon to auto-spawn factual questions.

---

# Decision: Question Subtype Encoding via Labels

**Date:** 2026-01-28
**Status:** Accepted

**Extracted-From:** `.kb/investigations/2026-01-19-inv-evaluate-encoding-options-question-subtypes.md`
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Context

The decidability graph model defines three question subtypes with different authority requirements:
- **Factual:** "How does X work?" - Can be resolved by investigation (daemon-spawnable)
- **Judgment:** "Should we use X or Y?" - Requires orchestrator synthesis
- **Framing:** "Is X even the right question?" - Requires Dylan to reframe

To enable authority-aware daemon behavior (auto-spawning investigations for factual questions while deferring judgment/framing to humans), we need a way to encode question subtypes in beads.

---

## Options Considered

### Option A: Labels with Convention
- **Pros:** Zero schema changes, works today, follows existing patterns (`triage:ready`), daemon infrastructure already supports it, labels can change as question evolves
- **Cons:** No schema validation (freeform), requires manual labeling, could have multiple subtypes

### Option B: Dedicated Field in Beads Schema
- **Pros:** Type-safe enum, single value enforcement, clear semantics
- **Cons:** Requires upstream beads changes, higher implementation cost, couples beads to orch-go concepts

### Option C: Inference from Question Content
- **Pros:** Automatic, no manual labeling
- **Cons:** Unreliable classification, requires AI/heuristics, questions change subtype during resolution

---

## Decision

**Chosen:** Option A - Labels with Convention

**Convention:**
```
subtype:factual   - "How does X work?" - Daemon can spawn investigation
subtype:judgment  - "Should we use X or Y?" - Orchestrator synthesizes
subtype:framing   - "Is X even the right question?" - Dylan reframes
```

**Usage:**
```bash
# Create a factual question
bd create "How does the escalation model work?" --type question -l subtype:factual

# Query factual questions ready for daemon
bd ready --type question --label subtype:factual
```

**Rationale:** Labels match the existing beads pattern, require no schema changes, and provide the flexibility needed for question subtypes that may evolve during resolution. The daemon already has label filtering infrastructure via `HasLabel()`.

**Trade-offs accepted:**
- No schema validation (freeform labels) - mitigated by convention documentation
- Could have multiple subtypes - convention: single subtype per question
- Requires manual labeling at question creation - acceptable for low-volume question creation

---

## Structured Uncertainty

**What's tested:**
- ✅ Labels field exists in beads Issue struct (pkg/beads/types.go:148)
- ✅ Daemon filters by labels via HasLabel() (pkg/daemon/daemon.go:331)
- ✅ BD CLI supports label filtering (bd ready --label flag)
- ✅ Question type works with daemon (maps to investigation skill)

**What's untested:**
- ⚠️ Actual daemon behavior with `subtype:factual` questions (no questions have this label yet)
- ⚠️ Whether questions actually evolve subtypes in practice

**What would change this:**
- If beads adds a dedicated `QuestionSubtype` field, Option B becomes viable
- If label proliferation causes confusion, stricter encoding might be needed
- If subtype detection can be reliably automated, Option C becomes attractive

---

## Consequences

**Positive:**
- Questions can be triaged by resolvability (factual vs judgment vs framing)
- Daemon can optionally auto-spawn investigations for factual questions
- Dashboard can group questions by subtype for better visibility
- Zero implementation required - works with current beads

**Risks:**
- Convention drift if not documented and enforced by habit
- May create false sense of certainty (subtype labels imply clear categories, reality is fuzzy)

---

## Implementation

1. **Document convention** - Add to CLAUDE.md and decidability-graph model
2. **Question creation guidance** - Add `--labels subtype:factual` examples to workflows
3. **Daemon extension (optional)** - Add flag to spawn factual questions as investigations

---

## References

- **Investigation:** `.kb/investigations/2026-01-19-inv-evaluate-encoding-options-question-subtypes.md`
- **Model:** `.kb/models/decidability-graph.md` - Question subtypes and authority levels
- **Decision:** `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` - Question bead type
