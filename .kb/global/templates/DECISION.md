---
status: draft
# Optional: Gate future spawns that would conflict with this decision
# Add blocks: when this decision resolves recurring issues (3+ prior investigations) or
# establishes constraints future agents might violate. Keywords should match how someone
# would describe the problem in a spawn task.
# blocks:
#   - keywords:
#       - [keyword that should trigger this decision]
#       - [another keyword]
#     patterns:
#       - "**/path/pattern*"
---

## Summary (D.E.K.N.)

**Delta:** [What was decided - the choice made in one sentence]

**Evidence:** [What informed this decision - investigation findings, prior experience]

**Knowledge:** [Key insight that drove the choice]

**Next:** [Implementation steps or follow-up]

---

# Decision: {{title}}

**Date:** {{date}}
**Status:** Draft

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Context

[What prompted this decision?]

---

## Options Considered

### Option A: [Name]
- **Pros:** 
- **Cons:** 

### Option B: [Name]
- **Pros:** 
- **Cons:** 

---

## Decision

**Chosen:** [Option name]

**Rationale:** [Why this option?]

**Trade-offs accepted:**
- [What we're giving up or deferring]

---

## Structured Uncertainty

**What's tested:**
- ✅ [Fact that informed this decision with source]

**What's untested:**
- ⚠️ [Assumption that might not hold]

**What would change this:**
- [Condition under which we'd revisit this decision]

---

## Consequences

**Positive:**
- [Benefit 1]

**Risks:**
- [Risk 1]
