# Session Synthesis

**Agent:** og-inv-audit-skill-documentation-13feb-21da
**Issue:** orch-go-aw1
**Duration:** 2026-02-13
**Outcome:** success

---

## TLDR

Audited all skill documentation and CLAUDE.md files for claims about the model/probe/investigation system. Found 6 critical inconsistencies: missing PROBE.md template, two conflicting deployed investigation skill versions, contradictory model creation thresholds (3+ vs 15+), overloaded "probe" terminology, mismatched routing mechanisms, and low worker skill coverage.

---

## Delta (What Changed)

### Files Modified
- `.kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md` - Complete investigation with 8 findings, canonical decision tree, and implementation recommendations

### Commits
- Investigation file with comprehensive audit findings

---

## Evidence (What Was Observed)

- `.orch/templates/PROBE.md` does not exist — referenced in 7+ locations across orchestrator and investigation skills
- Two deployed investigation skills: `~/.claude/skills/src/worker/investigation/SKILL.md` (old, no probe mode, checksum 91b0e65cab3c) and `~/.claude/skills/worker/investigation/SKILL.md` (new, has probe mode, checksum 1cf402739ec0)
- This investigation was spawned with the OLD version (91b0e65cab3c per SPAWN_CONTEXT header), meaning probe-aware routing was specified but probe execution was not possible
- `.kb/models/README.md` says "3+ investigations" triggers model creation; `.kb/guides/understanding-artifact-lifecycle.md` says "15+" and explicitly calls 3 an anti-pattern
- "probe" is used with 3 distinct meanings across skills: model-scoped probes (investigation/orchestrator), decision-navigation probes (experiment/prototype), and epic model probes (tracking understanding)
- Only 3 of 15+ skills have substantive model/probe awareness

### Tests Run
```bash
# Verified PROBE.md missing
Read .orch/templates/PROBE.md → File does not exist

# Verified two skill versions
Read ~/.claude/skills/src/worker/investigation/SKILL.md → checksum 91b0e65cab3c (no probe)
Read ~/.claude/skills/worker/investigation/SKILL.md → checksum 1cf402739ec0 (has probe)

# Searched all skill sources
Grep "probe|\.kb/models/" ~/orch-knowledge/skills/src/ → 80+ matches
Grep "probe|PROBE" ~/.claude/skills/ → 9 files
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md` - Complete audit with canonical decision tree

### Decisions Made
- The orchestrator skill's routing table is the most authoritative source for probe-vs-investigation routing
- The model creation threshold should be reconciled to the lifecycle guide's 15+ with four-factor test

### Constraints Discovered
- PROBE.md template missing blocks the entire probe creation pipeline for workers
- Two deployed investigation skill versions means worker behavior depends on which path gets loaded
- CLAUDE.md intent-based routing conflicts with skill marker-based detection
- "probe" semantic overload creates confusion across decision-navigation and investigation contexts

---

## Verification Contract

**Verification specification:** See investigation file for structured uncertainty section.

**Key outcomes:**
1. 8 findings documented with evidence and significance
2. Canonical decision tree constructed from all sources
3. 6 implementation recommendations with authority classification
4. Coverage table for all 15+ skills

---

## Next (What Should Happen)

**Recommendation:** close (investigation complete, findings ready for orchestrator review)

### If Close
- [x] All deliverables complete (investigation file fully populated)
- [x] Investigation file has `**Phase:** Complete` and `**Status:** Complete`
- [x] Ready for `orch complete orch-go-aw1`

### Recommended Follow-Up Issues
1. **Create PROBE.md template** — `bd create "Create .orch/templates/PROBE.md with 4 required sections" --type task`
2. **Clean up old investigation skill** — `bd create "Remove ~/.claude/skills/src/worker/investigation/SKILL.md (superseded)" --type task`
3. **Reconcile model creation threshold** — `bd create "Update README.md to match lifecycle guide 15+ threshold" --type task`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why do two deployment paths exist for skills (`src/worker/` vs `worker/`)? Is this a skillc deploy bug or intentional?
- Are existing probes in `.kb/models/*/probes/` following the 4-section structure despite no template?
- Should kb-reflect have model creation guidance (it handles synthesis of 3+ investigations but doesn't mention models)?

**What remains unclear:**
- Root cause of three different investigation skill checksums (source, old deployed, new deployed)
- Whether capture-knowledge skill exists as a source (not found at expected path)

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-audit-skill-documentation-13feb-21da/`
**Investigation:** `.kb/investigations/2026-02-13-inv-audit-model-probe-investigation-claims.md`
**Beads:** `bd show orch-go-aw1`
