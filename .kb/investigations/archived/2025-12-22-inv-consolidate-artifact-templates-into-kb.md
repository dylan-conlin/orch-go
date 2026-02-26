## Summary (D.E.K.N.)

**Delta:** Consolidated artifact templates into kb-cli by updating hardcoded templates with D.E.K.N. + structured uncertainty format and adding `kb create research` command.

**Evidence:** kb-cli tests pass, new templates verified with D.E.K.N. format, skill templates/ directories removed.

**Knowledge:** kb-cli owns artifact templates (investigation, decision, guide, research); orch owns spawn-time templates (SYNTHESIS, SPAWN_CONTEXT).

**Next:** Close issue - consolidation complete.

---

# Investigation: Consolidate Artifact Templates Into KB

**Question:** How to consolidate the fragmented template systems into kb-cli as the single source of truth for artifact templates?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** og-feat-consolidate-artifact-templates-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: kb-cli hardcoded templates were outdated

**Evidence:** investigationTemplate was 25 lines, ~/.kb/templates/INVESTIGATION.md was 234 lines with D.E.K.N. summary and structured uncertainty sections.

**Source:** kb-cli/cmd/kb/create.go:14-39 vs ~/.kb/templates/INVESTIGATION.md

**Significance:** User-level templates had evolved separately, kb-cli defaults were stale.

---

### Finding 2: Four skill templates/ directories existed with duplicates

**Evidence:** Found templates at:
- ~/.claude/skills/worker/research/templates/
- ~/.claude/skills/worker/investigation/templates/
- ~/.claude/skills/investigation/templates/
- ~/.claude/skills/research/templates/

**Source:** `find ~/.claude/skills -type d -name "templates"`

**Significance:** Multiple sources of truth caused divergence - some had D.E.K.N., some didn't.

---

### Finding 3: `kb create research` command was missing

**Evidence:** Research was valid investigation type but had no dedicated command.

**Source:** kb-cli/cmd/kb/create.go - only investigation/decision/guide commands existed

**Significance:** Added research command for consistency with skill types.

---

## Synthesis

**Key Insights:**

1. **Domain-based ownership is correct** - kb-cli owns artifact templates (investigation, decision, guide, research), orch owns spawn-time templates (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT)

2. **Skill-level templates are now obsolete** - All templates should come from kb-cli hardcoded defaults or ~/.kb/templates/ overrides

3. **D.E.K.N. + Structured Uncertainty is the canonical format** - All artifact templates now include these sections at the top

**Answer to Investigation Question:**

Consolidated by: (1) updating kb-cli create.go with D.E.K.N. templates, (2) adding research command, (3) removing skill templates/ directories, (4) updating ~/.kb/templates/ overrides. kb-cli is now the single source of truth for artifact templates.

---

## Structured Uncertainty

**What's tested:**
- ✅ kb-cli tests pass (50+ tests)
- ✅ `kb create research` works correctly
- ✅ New templates include D.E.K.N. format

**What's untested:**
- ⚠️ Whether other systems still reference removed skill templates
- ⚠️ Full workflow with spawned agents using new templates

**What would change this:**
- Discovery of other systems depending on skill templates/ directories

---

## References

**Files Modified:**
- `kb-cli/cmd/kb/create.go` - Updated templates, added research command
- `~/.kb/templates/DECISION.md` - Updated to D.E.K.N. format
- `~/.kb/templates/RESEARCH.md` - Created

**Files Removed:**
- `~/.claude/skills/*/templates/` directories

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md` - Prior investigation identifying fragmentation

---

## Investigation History

**2025-12-22 12:00:** Investigation started
- Initial question: How to consolidate templates into kb-cli?
- Context: Prior investigation identified 6 template systems with divergence

**2025-12-22 12:30:** Implementation complete
- Updated kb-cli templates with D.E.K.N.
- Added `kb create research` command
- Removed skill templates/ directories

**2025-12-22 12:45:** Investigation completed
- Status: Complete
- Key outcome: kb-cli is now single source of truth for artifact templates
