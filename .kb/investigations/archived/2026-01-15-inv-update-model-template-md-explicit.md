# Investigation: Update Model Template Md Explicit

## Summary (D.E.K.N.)

**Delta:** Updated TEMPLATE.md Constraints section with explicit enable/constrain pattern; decided `kb create model` tooling not needed.

**Evidence:** Template now has structured pattern (Constraint/Implication/This enables/This constrains). Models are infrequent synthesized artifacts - tooling overhead exceeds benefit.

**Knowledge:** Models emerge from investigation synthesis, not scratch creation. Template structure is sufficient to enforce consistency.

**Next:** Close - task complete.

**Promote to Decision:** recommend-no (tactical template fix, not architectural)

---

**Question:** Should TEMPLATE.md Constraints section be updated with explicit enable/constrain pattern? Does `kb create model` tooling add value?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Vague Constraints Section Caused Inconsistent Usage

**Evidence:** Issue description notes morning agents (N=4) independently discovered enable/constrain pattern; afternoon agents (N=6) didn't use it. Current template just says "**Why {constraint}:**" without structure.

**Source:** `.kb/models/TEMPLATE.md:39-44`, issue orch-go-kpdg2 description

**Significance:** Template-as-documentation only works when patterns are explicit. Implicit conventions get dropped.

---

### Finding 2: Models Are Infrequently Created Synthesized Artifacts

**Evidence:** Models domain in `.kb/models/` contains understanding artifacts that emerge from multiple investigations. They're not created from scratch like investigations or decisions - they're synthesized.

**Source:** kb decisions `2026-01-12-models-as-understanding-artifacts.md`, template ownership decisions in prior knowledge

**Significance:** Tooling (`kb create model`) adds friction for artifacts that are infrequently created. Template structure provides sufficient guidance.

---

## Decision on `kb create model` Tooling

**Recommendation:** Template update sufficient - tooling not needed.

**Reasoning:**
1. Models are synthesized from investigations, not created from scratch
2. Low creation frequency (investigations/decisions are much more common)
3. `kb create investigation` pattern exists for frequent artifacts
4. Template structure provides explicit format - agents will follow it
5. Tooling maintenance overhead exceeds benefit

---

## Implementation

**Updated:** `.kb/models/TEMPLATE.md` Constraints section now has:

```markdown
### Why {Constraint Question}?

**Constraint:** {What's the limitation}

**Implication:** {What this means in practice}

**This enables:** {What becomes possible}
**This constrains:** {What's no longer allowed/possible}
```

This matches the enable/constrain pattern discovered by morning agents.

---

## References

**Files Modified:**
- `.kb/models/TEMPLATE.md:39-50` - Updated Constraints section

**Issue:**
- orch-go-kpdg2 - Source of exact format specification
