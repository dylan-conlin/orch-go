# Session Synthesis

**Agent:** og-inv-audit-rebuild-change-28jan-7cd4
**Issue:** orch-go-20986
**Duration:** 2026-01-28 19:15 → 2026-01-28 19:50
**Outcome:** success

---

## TLDR

Audited rebuild-on-change mechanisms across all 10 ecosystem repos and identified three distinct mechanisms (post-commit hooks in 40%, orch complete auto-rebuild, manual make install) with critical gaps in staleness detection (glass and agentlog lack version commands entirely).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-28-inv-audit-rebuild-change-mechanisms-across.md` - Comprehensive audit of rebuild mechanisms with detailed matrix

### Files Modified
None (investigation only, no code changes)

### Commits
- `26e271ae` - inv: audit rebuild-on-change mechanisms - initial checkpoint
- `e177d0ea` - inv: complete rebuild mechanism audit - found 3 mechanisms, 40% auto-rebuild coverage

---

## Evidence (What Was Observed)

- Post-commit hooks exist in 4 of 10 repos: orch-go, kb-cli, agentlog, kn (tested with `test -f .git/hooks/post-commit`)
- Version commands with git hash exist in 4 of 6 CLIs: orch, kb, bd, skillc (tested via `<cli> version`)
- glass has NO version command (errors with "unknown command: version")
- agentlog has NO version command (errors with "unknown command")
- orch complete has `rebuildGoProjectsIfNeeded()` function (found in 2026-01-23 investigation)
- skillc has auto-rebuild mechanism that can race (observed "rebuild already in progress" error)

### Tests Run
```bash
# Batch audit of all repos
/tmp/check_rebuild_mechanisms.sh

# Version command testing
orch version  # Shows: eedc8991
kb version    # Shows: 5e52def-dirty
bd version    # Shows: 0.41.0 (629441ad)
skillc version  # Shows: a8b8b25-dirty with auto-rebuild race warning
glass version  # Error: unknown command
agentlog version  # Error: unknown command

# Hook examination
cat ~/Documents/personal/orch-go/.git/hooks/post-commit
cat ~/Documents/personal/kb-cli/.git/hooks/post-commit
cat ~/Documents/personal/agentlog/.git/hooks/post-commit
cat ~/Documents/personal/kn/.git/hooks/post-commit
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-28-inv-audit-rebuild-change-mechanisms-across.md` - Complete matrix of rebuild mechanisms across ecosystem with gap analysis and prioritized recommendations

### Decisions Made
- Decision 1: Prioritize staleness detection (version commands) over auto-rebuild (hooks) because detection enables both manual verification AND automated tooling
- Decision 2: Created detailed matrix showing all repos to make gaps visible and actionable

### Constraints Discovered
- No single rebuild mechanism covers all scenarios (hooks miss 60%, orch complete only in agent workflows, manual is error-prone)
- Glass and agentlog have ZERO staleness detection capability (no version command)
- skillc has custom auto-rebuild with potential race conditions

### Externalized via `kb quick`
- Captured finding about three distinct rebuild mechanisms with no comprehensive coverage

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has `**Phase:** Complete`
- [x] Detailed matrix with recommendations created
- [x] Ready for `orch complete orch-go-20986`

**Follow-up work identified (create separate issues):**
1. Add version command to glass (high priority - enables staleness detection)
2. Add version command to agentlog (high priority - enables staleness detection)
3. Add post-commit hook to skillc (medium priority - auto-rebuild)
4. Add post-commit hook to glass (medium priority - auto-rebuild)
5. Document skillc auto-rebuild mechanism and race condition (low priority)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What is skillc's auto-rebuild mechanism exactly? (Observed race condition but didn't examine implementation)
- Why is kn binary not in PATH? (Hook exists and looks correct but binary not found during testing)
- Could post-commit hooks be centrally managed for consistency? (All 4 hooks have slightly different implementations)

**Areas worth exploring further:**
- Automated staleness checks for ALL ecosystem binaries (expand `orch doctor` beyond just orch)
- Watch mechanism for orch-knowledge skills (auto-run `skillc deploy` when skill sources change)
- Central hook management system for consistency

**What remains unclear:**
- skillc race condition severity (single observation, not reproduced)
- Whether beads needs hooks (upstream OSS, Dylan rarely modifies)
- Best approach for beads-ui-svelte rebuild (Vite handles dev, production builds are infrequent)

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-5
**Workspace:** `.orch/workspace/og-inv-audit-rebuild-change-28jan-7cd4/`
**Investigation:** `.kb/investigations/2026-01-28-inv-audit-rebuild-change-mechanisms-across.md`
**Beads:** `bd show orch-go-20986`
