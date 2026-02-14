## Summary (D.E.K.N.)

**Delta:** .kb/models/README.md said 3+ investigations triggers model creation, contradicting the lifecycle guide's 15+ threshold and four-factor test.

**Evidence:** README.md line 12 said "3+ investigations"; lifecycle guide lines 179-184 say 15+ with four-factor test; lifecycle guide line 394 explicitly calls 3→model an anti-pattern.

**Knowledge:** The README was written before the lifecycle guide codified the thresholds. The 3+ signal is useful as a watch indicator but not a creation trigger.

**Next:** None - fix applied, README now aligned with lifecycle guide.

**Authority:** implementation - Documentation fix within existing patterns, no architectural impact.

---

# Investigation: Reconcile Model Creation Threshold Contradiction

**Question:** How should .kb/models/README.md be updated to align with the lifecycle guide's model creation thresholds?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker (orch-go-7ii)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Findings

### Finding 1: README stated 3+ as creation signal

**Evidence:** `.kb/models/README.md` line 12: "3+ investigations on same topic converge to understanding" listed as first signal under "When to Create a Model".

**Source:** `.kb/models/README.md:12`

**Significance:** This directly contradicts the lifecycle guide and could lead orchestrators to create premature models.

---

### Finding 2: Lifecycle guide establishes 15+ threshold with four-factor test

**Evidence:** `.kb/guides/understanding-artifact-lifecycle.md` lines 179-184 specify: creation trigger is 15+ investigations, with four-factor test (HOT, COMPLEX, OWNED, STRATEGIC_VALUE) all required. Hard floor of 10 at lines 399-400.

**Source:** `.kb/guides/understanding-artifact-lifecycle.md:179-184, 392-400`

**Significance:** The lifecycle guide is the authoritative source, written later with more operational experience.

---

### Finding 3: Lifecycle guide explicitly calls 3→model an anti-pattern

**Evidence:** `.kb/guides/understanding-artifact-lifecycle.md` line 394: "3 investigations on topic → 'let's create a model'" listed as anti-pattern with explanation: "Models are synthesis artifacts. 3 investigations don't provide enough perspective."

**Source:** `.kb/guides/understanding-artifact-lifecycle.md:392-398`

**Significance:** The two documents were in direct contradiction - one recommended what the other prohibited.

---

## Synthesis

**Answer:** Updated README.md to:
1. Set creation threshold to 15+ with four-factor test (matching lifecycle guide)
2. Added hard floor of 10 (matching lifecycle guide)
3. Kept 3+ as a "watch signal" (topic to monitor, not a creation trigger)
4. Added explicit anti-pattern warning against premature model creation
5. Updated success criteria to reference 15+ threshold

## References

**Files Examined:**
- `.kb/models/README.md` - The file with the contradictory threshold
- `.kb/guides/understanding-artifact-lifecycle.md` - The authoritative lifecycle guide
