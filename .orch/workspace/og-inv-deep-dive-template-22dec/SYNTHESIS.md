# Session Synthesis

**Agent:** og-inv-deep-dive-template-22dec
**Issue:** orch-go-xrdc
**Duration:** 2025-12-22 ~07:00 → ~08:00
**Outcome:** success

---

## TLDR

Deep dive into template system fragmentation found 6 distinct template systems across the orch ecosystem with significant content divergence. Recommends domain-based ownership: kb-cli owns artifact templates (investigation/decision/guide), orch-go owns spawn-time templates (SYNTHESIS/SPAWN_CONTEXT/FAILURE_REPORT). skillc should NOT expand to artifact templates.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md` - Complete investigation documenting all 6 template systems

### Files Modified
- None (investigation only)

### Commits
- (pending - will commit investigation file)

---

## Evidence (What Was Observed)

- 6 distinct template systems identified across orch-go, kb-cli, ~/.kb/templates/, .orch/templates/, orch-knowledge skills, and skillc
- `diff` between ~/.kb/templates/INVESTIGATION.md and orch-knowledge version shows 110+ lines of divergence
- ~/.kb/templates/INVESTIGATION.md has D.E.K.N. summary (234 lines), orch-knowledge version lacks it (124 lines)
- ~/.orch/templates/SYNTHESIS.md is **DEPRECATED** (20 lines), but orch-go embeds a full 122-line version
- skillc only handles CLAUDE.md compilation from skill sources - does NOT touch artifact templates

### Tests Run
```bash
# Compared template files
diff -u ~/.kb/templates/INVESTIGATION.md orch-knowledge/.../investigation.md
# Result: 110+ lines different, structural divergence confirmed

diff -u ~/.orch/templates/SYNTHESIS.md orch-go/.orch/templates/SYNTHESIS.md
# Result: 100+ lines different, one is deprecated warning, other is full template
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md` - Full investigation

### Decisions Made
- Decision 1: skillc should NOT own artifact templates - its purpose (CLAUDE.md compilation) is distinct
- Decision 2: Domain-based ownership recommended (kb-cli for artifacts, orch-go for spawn-time)

### Constraints Discovered
- Templates evolved independently without synchronization mechanism
- No single source of truth exists - each system has its own fallback chain
- User overrides split between ~/.kb/templates/ and ~/.orch/templates/

### Externalized via `kn`
- (none yet - recommend orchestrator record constraint about template ownership)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (diffs run, evidence gathered)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-xrdc`

### Discovered Follow-up Issues
1. **Sync ~/.kb/templates/INVESTIGATION.md to kb-cli hardcoded fallback** - const investigationTemplate in create.go is 25 lines, ~/.kb/templates/ version is 234 lines
2. **Retire orch-knowledge investigation template** - skills/src/worker/investigation/templates/investigation.md is stale
3. **Verify orch-go embedded SYNTHESIS matches .orch/templates/** - May have diverged during development

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should ~/.kb/templates/ and ~/.orch/templates/ be unified into a single location?
- How did the templates diverge - was it intentional feature evolution or accidental drift?

**Areas worth exploring further:**
- Runtime template usage analysis - which templates are actually loaded vs. which are stale
- Template versioning strategy to prevent future drift

**What remains unclear:**
- Dylan's original intent for template organization
- Whether orch-knowledge skill-embedded templates were ever intended to be canonical

---

## Session Metadata

**Skill:** investigation
**Model:** opus
**Workspace:** `.orch/workspace/og-inv-deep-dive-template-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md`
**Beads:** `bd show orch-go-xrdc`
