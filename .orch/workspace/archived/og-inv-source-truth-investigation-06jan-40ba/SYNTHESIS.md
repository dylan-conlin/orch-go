# Session Synthesis

**Agent:** og-inv-source-truth-investigation-06jan-40ba
**Issue:** orch-go-jpvs0
**Duration:** 2026-01-06 → Complete
**Outcome:** success

---

## TLDR

Traced investigation template code paths to determine source of truth: kb-cli's `loadTemplate()` checks `~/.kb/templates/INVESTIGATION.md` first (user template), falls back to embedded constant. The investigation skill defers to `kb create investigation` for file creation. To add 'promote-to-decision' as a Next option, update the template's Next field guidance text.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-06-inv-source-truth-investigation-file-creation.md` - Complete investigation documenting template precedence and format variations

### Files Modified
- None (investigation-only task)

### Commits
- To be committed after Phase: Complete report

---

## Evidence (What Was Observed)

- kb-cli `loadTemplate()` function at `create.go:560-574` implements clear precedence: user template first, embedded fallback
- `~/.kb/templates/INVESTIGATION.md` exists (221 lines) and IS used when present
- Investigation skill contains simplified template (30 lines) as documentation example, with explicit instruction: "Use `kb create investigation {slug}` to create"
- 501/559 investigation files in orch-go have D.E.K.N. format (current standard)
- 58 files without D.E.K.N. are either pre-Dec-2025 (older template) or manually created by agents
- Current Next field guidance: `[close, implement, investigate further, or escalate]` - no promote-to-decision option

### Tests Run
```bash
# Verified template precedence
TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/.kb/investigations"
cd "$TEST_DIR" && kb create investigation test-template-check
# Result: Output matches ~/.kb/templates/INVESTIGATION.md exactly

# Counted format variations
grep -l "D.E.K.N." .kb/investigations/*.md | wc -l  # 501
grep -L "D.E.K.N." .kb/investigations/*.md | wc -l  # 58
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-06-inv-source-truth-investigation-file-creation.md` - Complete tracing of template ownership

### Decisions Made
- None required - this was pure investigation

### Constraints Discovered
- kb-cli embedded template requires recompile to update (user template takes immediate effect)
- Skill template is documentation only - not a generator

### Externalized via `kn`
- Not applicable - findings documented in investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file fully populated)
- [x] Tests performed (traced code paths, verified template precedence)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-jpvs0`

**Optional Follow-up Actions (not blocking):**
1. Update `~/.kb/templates/INVESTIGATION.md` line 14 to add "promote-to-decision" to Next options
2. Update kb-cli embedded template for consistency (requires recompile)
3. Update investigation skill documentation to mention promote-to-decision option

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `kb promote` command be invoked automatically based on Next field value?
- Should there be template validation to ensure created files match expected format?

**What remains unclear:**
- Whether agents actually read/follow the Next field guidance (behavioral observation not performed)

*(Straightforward investigation with clear answers)*

---

## Session Metadata

**Skill:** investigation
**Model:** Claude
**Workspace:** `.orch/workspace/og-inv-source-truth-investigation-06jan-40ba/`
**Investigation:** `.kb/investigations/2026-01-06-inv-source-truth-investigation-file-creation.md`
**Beads:** `bd show orch-go-jpvs0`
