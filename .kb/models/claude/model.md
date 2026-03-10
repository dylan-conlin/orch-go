# Model: Claude

**Created:** 2026-03-09
**Status:** Active
**Source:** Synthesized from 3 investigation(s)

## What This Is

[What phenomenon or pattern does this model describe? What makes it a coherent concept worth naming?]

---

## Core Claims (Testable)

### Claim 1: [Concise claim statement]

[Explanation of the claim. What would you observe if it's true? What would falsify it?]

**Test:** [How to test this claim]

**Status:** Hypothesis

### Claim 2: [Concise claim statement]

[Explanation of the claim.]

**Test:** [How to test this claim]

**Status:** Hypothesis

---

## Implications

[What follows from these claims? How should this model change behavior, design, or decision-making?]

---

## Boundaries

**What this model covers:**
- [Scope item 1]

**What this model does NOT cover:**
- [Exclusion 1]

---

## Evidence

| Date | Source | Finding |
|------|--------|---------|
| 2026-03-09 | Model creation | Initial synthesis from source investigations |

---

## Open Questions

- [Question that further investigation could answer]
- [Question about model boundaries or edge cases]

## Source Investigations

### 2026-02-14-inv-add-claude-md-accretion-boundaries.md

**Delta:** Added concise Accretion Boundaries section to CLAUDE.md using progressive disclosure pattern (4 lines vs initial 20-line draft).
**Evidence:** CLAUDE.md now contains accretion rule (>1,500 lines = extraction first), pointers to `orch hotspot` and extraction guide, enforcement summary with link to full architecture.
**Knowledge:** CLAUDE.md is loaded into every agent session, so content must be minimal with progressive disclosure to detailed .kb/ documentation; guarded file reminder caught excessive detail in first draft.
**Next:** Close after commit - section complete and follows CLAUDE.md guidelines for conciseness.

---

### 2026-02-14-inv-fix-claude-md-remove-deleted.md

**Delta:** CLAUDE.md contained references to deleted pkg/registry/, incorrect cmd/orch/ listing, and duplicated model section content.
**Evidence:** File system check showed no pkg/registry/ directory, cmd/orch/ has 100+ files not 4, model section lines 164-169 repeated identical content 3 times.
**Knowledge:** Documentation drift occurs when code structure changes but documentation isn't updated; systematic verification needed.
**Next:** Changes committed; close issue after verification.
