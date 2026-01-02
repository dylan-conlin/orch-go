# Session Synthesis

**Agent:** og-inv-what-knowledge-context-22dec
**Issue:** orch-go-untracked-1766438634 (ad-hoc spawn)
**Duration:** 2025-12-22 ~11:00 → 2025-12-22 ~11:45
**Outcome:** success

---

## TLDR

Investigated what knowledge lives in completed workspace directories that isn't captured elsewhere. Found that SPAWN_CONTEXT.md is 100% redundant (safe to delete), but SYNTHESIS.md contains unique session metadata (duration, model, commits, unexplored questions) not preserved in .kb/ investigations or beads.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-what-knowledge-context-lives-completed.md` - Full investigation with findings

### Files Modified
- None

### Commits
- (pending) feat: investigate workspace knowledge preservation

---

## Evidence (What Was Observed)

- 225 total workspace directories, 125 with SYNTHESIS.md (55% completion rate)
- SPAWN_CONTEXT.md is generated from: beads issue + kb context query + skill template + spawn template
- SYNTHESIS.md uniquely captures: duration, model used, skill invoked, commit SHAs, file changes, unexplored questions
- .kb/ investigations have MORE technical detail but LACK session metadata
- Beads issues have only title/description, no execution details
- Git commits preserved regardless but lose session-to-commit linkage without SYNTHESIS.md

### Tests Run
```bash
# Artifact comparison test
# Compared og-debug-orch-send-fails-21dec across 4 layers:
# - SYNTHESIS.md (73 lines) - has unique session metadata
# - .kb/investigations/2025-12-21-debug-orch-send-fails-silently-tmux.md (215 lines) - technical detail
# - bd show orch-go-kszt - only title/description
# - git log --grep="kszt" - code changes preserved

# Result: SYNTHESIS.md fills genuine gap for session-level context
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-what-knowledge-context-lives-completed.md` - Knowledge preservation analysis

### Decisions Made
- Decision: SPAWN_CONTEXT.md is 100% redundant because it's generated from existing sources (beads + kb + skill + template)
- Decision: SYNTHESIS.md has unique value - session metadata not captured elsewhere

### Constraints Discovered
- Knowledge is distributed across 4 layers with different completeness: .kb/ (100% technical), git (100% code), beads (25% description), workspace (unique metadata)
- 45% of workspaces have no SYNTHESIS.md - unclear what session context is lost for those

### Externalized via `kn`
- None (findings documented in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for orchestrator review

### Recommended Implementation
If `orch clean` is enhanced:
1. Delete SPAWN_CONTEXT.md immediately (100% safe)
2. Extract session metadata from SYNTHESIS.md to compact JSON archive
3. Delete SYNTHESIS.md only if .kb/ investigation exists
4. Preserve ad-hoc artifacts or archive separately

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should SYNTHESIS.md content be merged into .kb/ investigations? (Would reduce artifact count)
- Should `orch complete` extract metadata before workspace cleanup? (Could automate)
- What happens to the 45% of workspaces without SYNTHESIS.md? (Incomplete sessions?)

**Areas worth exploring further:**
- Programmatic analysis of all 125 SYNTHESIS.md files for unique content patterns
- Whether "unexplored questions" section provides long-term value

**What remains unclear:**
- Exact disk savings from deleting SPAWN_CONTEXT.md vs full cleanup
- Whether ad-hoc artifacts (test results, transcripts) are ever referenced later

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-inv-what-knowledge-context-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-what-knowledge-context-lives-completed.md`
**Beads:** `bd show orch-go-untracked-1766438634` (ad-hoc spawn, issue not found)
