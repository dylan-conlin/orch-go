# Session Synthesis

**Agent:** og-arch-too-many-knowledge-04jan
**Issue:** orch-go-45qd
**Duration:** 2026-01-04 12:15 → 2026-01-04 13:15
**Outcome:** success

---

## TLDR

Analyzed knowledge artifact taxonomy and found we have 4 distinct types (Investigation, Decision, Guide, Quick) but 2 unused templates (RESEARCH.md, KNOWLEDGE.md). The key insight is that Guides are the synthesis output when `kb reflect` detects investigation clusters - the feedback loop isn't broken, it just wasn't documented.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-04-design-too-many-knowledge-artifact-types.md` - Architecture investigation on artifact taxonomy

### Files Modified
- None - this is a design investigation, no code changes

### Commits
- (pending) - architect: knowledge artifact taxonomy design

---

## Evidence (What Was Observed)

- RESEARCH.md template: 94 lines, 0 files using it (verified: `find .kb -name "*research*.md"` returns nothing)
- KNOWLEDGE.md template: 327 lines, 0 files using it (verified: no files with "knowledge" pattern in kb)
- Guides in orch-go: 7 files averaging 200 lines each, manually created with no template
- Guide headers all say "Single authoritative reference for X" - they ARE the synthesis output
- Prior decision (2025-12-21-minimal-artifact-taxonomy.md) defined 5+3 artifacts but omitted Guides
- Quick entries (.kb/quick/entries.jsonl): 10 entries, JSON lines format, operational memory

### Key Finding Pattern

From agent-lifecycle.md header:
> "Created after spending 1 hour debugging a problem that was already documented in kn. Synthesized from 20+ investigations about sessions/completion/lifecycle."

This IS the synthesis lifecycle: many investigations → Guide (not more investigations).

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-design-too-many-knowledge-artifact-types.md` - Clarifies the 4-type taxonomy

### Decisions Made
- RESEARCH.md and KNOWLEDGE.md should be deprecated (0 usage, purposes served by Investigation + Guide)
- Guides are the synthesis output type for `kb reflect` investigation clusters
- The lifecycle is: Investigation (exploratory) → Decision (if accepted) OR → Guide (if cluster synthesized)

### Constraints Discovered
- Source dimension (internal vs external) doesn't need separate templates - Investigation handles both
- Lifecycle dimension (point-in-time vs evolved) does need distinction - Investigation vs Guide

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file has findings, synthesis, recommendations)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-45qd`

### Follow-up Work (Not Blocking)

The investigation recommends 4 implementation steps - these should become beads issues:

1. **Deprecate unused templates** - Add notices to RESEARCH.md and KNOWLEDGE.md
2. **Create GUIDE.md template** - Formalize the guide format in `~/.kb/templates/`
3. **Update kb reflect** - Output "Create guide: {topic}" for investigation clusters
4. **Update CLAUDE.md** - Add Guide to knowledge placement table

These are feature-impl tasks, not blocking this architect session.

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How does kb-cli's `kb reflect` currently work? (Need to understand before updating it)
- Are there other projects outside orch-go using RESEARCH.md or KNOWLEDGE.md?

**Areas worth exploring further:**
- Guide template structure - should it have TLDR? D.E.K.N.? Different format?
- Whether kn entries should auto-suggest promotion when count exceeds threshold

**What remains unclear:**
- Whether deprecating templates affects agent training/expectations

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-too-many-knowledge-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-design-too-many-knowledge-artifact-types.md`
**Beads:** `bd show orch-go-45qd`
