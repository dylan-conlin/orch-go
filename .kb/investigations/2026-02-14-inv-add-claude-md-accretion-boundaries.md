<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added concise Accretion Boundaries section to CLAUDE.md using progressive disclosure pattern (4 lines vs initial 20-line draft).

**Evidence:** CLAUDE.md now contains accretion rule (>1,500 lines = extraction first), pointers to `orch hotspot` and extraction guide, enforcement summary with link to full architecture.

**Knowledge:** CLAUDE.md is loaded into every agent session, so content must be minimal with progressive disclosure to detailed .kb/ documentation; guarded file reminder caught excessive detail in first draft.

**Next:** Close after commit - section complete and follows CLAUDE.md guidelines for conciseness.

**Authority:** implementation - Tactical documentation addition within established patterns, no architectural impact.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Add Claude Md Accretion Boundaries

**Question:** How do we document accretion boundaries in CLAUDE.md without adding excessive context burden?

**Defect-Class:** unbounded-growth

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Worker agent (orch-go-241)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** Phase 1 of `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md`

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` | extends | Yes - verified Phase 1 recommendation | None - implements declarative boundaries layer as designed |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Initial Draft Violated CLAUDE.md Conciseness Guidelines

**Evidence:** First draft added ~20 lines with redundant details: specific file list (available via `orch hotspot`), repeated explanations of accretion pattern, lengthy workflow steps. Guarded file reminder caught excessive detail.

**Source:** CLAUDE.md edit attempt; guarded file system intervention

**Significance:** CLAUDE.md is loaded into every agent session - excessive detail wastes context tokens for agents that won't modify code. Need progressive disclosure: minimal in CLAUDE.md, detailed in .kb/.

---

### Finding 2: Progressive Disclosure Pattern Applied Successfully

**Evidence:** Final version reduced to 4 lines: rule statement, tooling pointer (`orch hotspot`), guide reference (`.kb/guides/code-extraction-patterns.md`), enforcement summary with link to full architecture.

**Source:** CLAUDE.md:111-114 (final section)

**Significance:** Achieves architect's "declarative boundaries" goal (agents know rule without hitting gates) while respecting CLAUDE.md conciseness constraint. Agents get awareness without context bloat.

---

## Synthesis

**Key Insights:**

1. **Declarative Boundaries vs Context Cost Tradeoff** - Architect investigation recommended CLAUDE.md placement for "agents see constraints before starting work," but guarded file system correctly enforces conciseness. Progressive disclosure resolves the tension: rule + pointers in CLAUDE.md, details in .kb/.

2. **Four-Layer Defense Implementation Sequence** - This completes Phase 1 "CLAUDE.md Boundaries" (zero cost, immediate effect). Remaining layers (spawn gates, completion gates, coaching plugin) build on this declarative foundation but have implementation/testing overhead.

**Answer to Investigation Question:**

Document accretion boundaries in CLAUDE.md using progressive disclosure: state the rule (>1,500 lines = extraction first), point to tooling (`orch hotspot`), link to detailed guide (`.kb/guides/code-extraction-patterns.md`), and reference enforcement architecture. This provides declarative prevention without context bloat, costing 4 lines instead of 20+.

---

## Structured Uncertainty

**What's tested:**

- ✅ CLAUDE.md section added - verified file edit successful, section appears at lines 111-114
- ✅ Progressive disclosure applied - verified final version is 4 lines vs 20-line initial draft
- ✅ Content follows architect recommendation - verified rule statement, tooling pointer, guide link, enforcement reference all present

**What's untested:**

- ⚠️ Will agents actually read and follow this section? (Hypothesis: declarative boundaries prevent violations, but not validated with real agent sessions)
- ⚠️ Is 4 lines the right balance? (Could be even more concise with just 2 lines: rule + link)
- ⚠️ Does this reduce accretion violations? (Won't know until spawn gates and completion gates are implemented and we see if agents tried to bypass)

**What would change this:**

- Finding would be wrong if agents ignore CLAUDE.md section and still attempt to add features to bloated files (suggests declarative boundaries alone insufficient)
- Finding would be wrong if 4 lines still causes context complaints (too much detail for every session)
- Finding would be wrong if agents can't find detailed guidance (progressive disclosure links broken or guides missing)

---

## Implementation Recommendations

**Status:** Implementation complete - this investigation was the implementation work itself (adding CLAUDE.md section).

### What Was Implemented

**Progressive Disclosure Pattern** - Added 4-line Accretion Boundaries section to CLAUDE.md with rule statement and pointers to detailed documentation.

**Why this approach:**
- Provides declarative boundaries (agents aware of rule before hitting gates)
- Minimizes context cost (4 lines vs 20+ line detailed version)
- Follows CLAUDE.md conciseness guidelines (caught by guarded file system)

**Success criteria:**
- ✅ CLAUDE.md section exists at lines 111-114
- ✅ Contains rule, tooling pointer, guide link, enforcement reference
- ✅ Follows progressive disclosure pattern (minimal in CLAUDE.md, detailed in .kb/)
- ✅ Completes Phase 1 of architect's four-layer enforcement design

---

## References

**Files Examined:**
- `CLAUDE.md:1-335` - Project context document; verified existing structure and added Accretion Boundaries section at line 111
- `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` - Source architect investigation; Phase 1 recommendation for CLAUDE.md boundaries
- `SPAWN_CONTEXT.md` - Spawn context for this task; contained task description and architect investigation context

**Files Modified:**
- `CLAUDE.md` - Added 4-line Accretion Boundaries section after "Architectural Principle: Pain as Signal" and before "Key References"
- `.kb/investigations/2026-02-14-inv-add-claude-md-accretion-boundaries.md` - This investigation file

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md` - Parent investigation with four-layer enforcement architecture
- **Guide:** `.kb/guides/code-extraction-patterns.md` - Referenced in CLAUDE.md section for extraction workflow
- **Issue:** orch-go-241 - Beads issue this work was spawned from

---

## Investigation History

**2026-02-14:** Investigation started (worker agent orch-go-241)
- Initial question: How do we document accretion boundaries in CLAUDE.md without adding excessive context burden?
- Context: Phase 1 of architect's four-layer accretion enforcement design; CLAUDE.md boundaries provide "zero cost, immediate effect" declarative prevention

**2026-02-14:** Guarded file system intervention
- First draft added ~20 lines with redundant details (file lists, lengthy explanations)
- Guarded file reminder enforced CLAUDE.md conciseness guidelines
- Revised to progressive disclosure pattern: 4 lines with pointers to detailed docs

**2026-02-14:** Investigation completed
- Status: Complete
- Key outcome: Added 4-line Accretion Boundaries section to CLAUDE.md using progressive disclosure, completing Phase 1 of accretion enforcement architecture
