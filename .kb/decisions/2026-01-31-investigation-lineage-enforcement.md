---
status: active
patches: 2025-12-21-minimal-artifact-taxonomy.md
---

# Decision: Investigation Lineage Enforcement

**Status:** Accepted
**Date:** 2026-01-31
**Deciders:** Dylan + Orchestrator
**Context:** Case Files design session revealed the real gap wasn't a missing artifact type but unstructured investigation chaining

---

## Summary

Investigations are relational, not independent. Require structured lineage metadata with Evidence Hierarchy verification at cite time. This captures most of what Case Files were trying to solve without adding a new artifact type.

---

## Decision

### Replace Supersedes with Prior-Work

The current `Supersedes:` field assumes investigations replace each other. In practice, only 0.86% of investigations supersede prior work. Most relationships are extensions, confirmations, or contradictions.

**New field structure:**

```markdown
## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-01-26-inv-X.md | extends | yes | None |
| .kb/investigations/2026-01-23-inv-Y.md | contradicts | yes | "Prior claimed all restarts kill agents; only OpenCode restarts matter" |
```

**Relationship vocabulary:**
- **Extends:** Adds to prior findings (most common)
- **Confirms:** Validates prior hypothesis with new evidence
- **Contradicts:** Disproves or refines prior conclusion
- **Deepens:** Explores same question at greater depth

### Evidence Hierarchy at Cite Time

When citing prior work, require:
1. **Verified:** Did you check claims against primary sources (code, test output, observed behavior)?
2. **Conflicts:** What contradictions did you find between prior claims and current evidence?

This applies the Evidence Hierarchy principle ("artifacts are claims to verify") at the moment of citation, not as an afterthought.

### Surfacing Prior Work at Spawn Time

`orch spawn investigation` should:
1. Run `kb context --topic` to find prior investigations on same topic
2. Inject results into SPAWN_CONTEXT.md
3. Require acknowledgment (gate, not reminder)

### Case Files Become Optional

With structured lineage capturing contradictions inline, Case Files are only needed for:
- Complex sagas (10+ investigations)
- Human evidence not captured in any investigation
- When the diagnostic narrative itself is valuable to preserve

This is an edge case, not a new artifact type to maintain.

---

## Context

### The Problem We Thought We Had

Case Files design session (Jan 31, 2026) proposed a new artifact type for multi-investigation failures. The coaching plugin saga (19 investigations, contradictory conclusions) seemed to justify this.

### The Problem We Actually Had

Investigation spawned to check assumptions found:
- 44.8% of investigations already cite prior work via in-text references
- Only 0.86% use formal `Supersedes:` metadata
- The server crash saga (11 investigations) chains correctly via informal citations

**The gap was metadata structure, not citation behavior.** Agents naturally chain investigations; tooling just can't parse it.

### Why Not Case Files?

The architect review correctly pushed back on Case Files as a new type (citing chronicle precedent: "views not types"). But the deeper reason is simpler: structured lineage with Evidence Hierarchy verification captures most of what Case Files were trying to capture.

---

## Options Considered

### Option A: Case Files as New Artifact Type

**Add `.kb/case-files/` with forensic diagnosis structure.**

**Pros:** Captures multi-investigation synthesis, evolution, contradiction resolution

**Cons:** New artifact type to maintain, adoption uncertain, doesn't fix root lineage problem

**Rejected:** Solves symptom (no synthesis artifact) not cause (unstructured lineage)

### Option B: Investigation Lineage Enforcement (Chosen) ⭐

**Require structured Prior-Work with verification.**

**Pros:**
- Addresses root cause (citations exist but aren't structured)
- Builds on existing behavior (agents already cite informally)
- Enables `kb reflect` to trace lineage programmatically
- No new artifact type

**Cons:**
- Adds friction to investigation creation
- Requires template and skill updates

### Option C: Both (Lineage + Case Files)

**Implement lineage now, Case Files later.**

**Deferred:** If structured lineage proves insufficient for complex sagas, Case Files can be added. But wait for evidence of need.

---

## Consequences

### Positive

- Investigations become queryable chains, not isolated documents
- Evidence Hierarchy verification happens at cite time (gate, not reminder)
- Contradictions captured in metadata, visible to tooling
- `kb reflect --lineage` can visualize investigation chains
- No taxonomy expansion (minimal artifact principle preserved)

### Negative

- Friction added to investigation creation
- Template migration for existing investigations (or leave as-is)
- May still need Case Files for extreme cases (10+ investigation sagas)

### Implementation Required

1. **Investigation template update** - Replace `Supersedes:` with `Prior-Work:` table
2. **`kb context --topic` command** - Surface prior investigations on topic
3. **Investigation skill update** - Require Prior-Work acknowledgment
4. **`kb reflect --lineage`** - Visualize investigation chains (later)

---

## Principles Applied

| Principle | Application |
|-----------|-------------|
| **Evidence Hierarchy** | Artifacts are claims to verify; verification required at cite time |
| **Session Amnesia** | Lineage metadata survives across sessions |
| **Coherence Over Patches** | Structured lineage enables synthesis detection |
| **Gate Over Remind** | Verification at cite time is enforced, not suggested |
| **Minimal Artifact Taxonomy** | No new type added; existing type enhanced |

---

## Related

- **Patches:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` (enhances Investigation artifact)
- **Investigation:** `.kb/investigations/2026-01-31-inv-investigation-churn-patterns-lineage.md`
- **Investigation:** `.kb/investigations/2026-01-31-inv-architect-synthesis-case-files.md`
- **Principle:** Evidence Hierarchy ("artifacts are claims to verify")
