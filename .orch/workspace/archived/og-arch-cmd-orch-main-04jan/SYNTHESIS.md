# Session Synthesis

**Agent:** og-arch-cmd-orch-main-04jan
**Issue:** orch-go-ii10
**Duration:** 2026-01-04 10:00 → 2026-01-04 11:30
**Outcome:** success

---

## TLDR

Designed a 4-phase refactoring plan to split cmd/orch/main.go (2705 lines, 49 fix commits) into 8 focused subcommand files following the established spawn_cmd.go/status_cmd.go pattern, reducing main.go to ~300 lines.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-04-inv-cmd-orch-main-go-49.md` - Full investigation with analysis, findings, and implementation recommendations
- `.orch/workspace/og-arch-cmd-orch-main-04jan/SYNTHESIS.md` - This synthesis file

### Files Modified
- None (design-only session)

### Commits
- No commits yet (investigation artifact only)

---

## Evidence (What Was Observed)

- main.go is 2705 lines with 22 cobra command definitions (`grep -n "var.*Cmd = &cobra.Command"`)
- Prior refactoring created spawn_cmd.go (1500 lines), status_cmd.go (902 lines), shared.go (301 lines)
- Commits `c129bf15`, `529b2e86`, `5ce7f2f8` show the extraction pattern working
- Git log shows 49+ commits to main.go in 28 days confirming "god file" churn
- daemon.go demonstrates Pattern A: parent command with sub-commands in same file
- serve_*.go demonstrates Pattern B: handler domain splitting (not needed for CLI commands)

### Analysis Performed
```bash
wc -l cmd/orch/main.go
# Output: 2705

grep -n "var.*Cmd = &cobra.Command" cmd/orch/main.go
# Found: 22 command definitions still in main.go

wc -l cmd/orch/*.go | sort -n
# Shows distribution of existing extractions
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-04-inv-cmd-orch-main-go-49.md` - Complete design for main.go refactoring

### Decisions Made
- Decision 1: Use phased extraction (4 phases) because it reduces risk of breaking changes
- Decision 2: Group small related commands (tail, question, abandon, retries) into agent_utils_cmd.go because they share agent interaction patterns
- Decision 3: Keep trivial commands (version, monitor, usage) in main.go because extraction overhead exceeds benefit

### Constraints Discovered
- Global variable `serverURL` defined in main.go must remain there or be carefully moved
- `DefaultServePort` constant defined in serve.go is used by main.go - cross-file dependency
- Test files (main_test.go) may need updates when functions move

### Pattern Codified
The existing extraction pattern works well:
1. `{command}_cmd.go` naming convention
2. Package doc comment explaining scope
3. `init()` function for flag registration
4. Sub-commands in same file as parent
5. Shared utilities in `shared.go`

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### Implementation Plan (4 Phases)

**Phase 1: complete_cmd.go** (P1 - Highest priority, ~450 lines)
- Extract: completeCmd, flags, runComplete
- Move: hasGoChangesInRecentCommits, detectNewCLICommands, detectNotableChangelogEntries
- Move: runAutoRebuild, restartOrchServe, invalidateServeCache
- Move: addApprovalComment, isSkillRelevantChange, truncateString
- Add to shared.go: formatDuration, extractProjectDirFromWorkspace

**Phase 2: clean_cmd.go** (P1 - ~400 lines)
- Extract: cleanCmd, flags, runClean
- Move: findCleanableWorkspaces, cleanOrphanedDiskSessions
- Move: cleanPhantomWindows, archiveEmptyInvestigations
- Move: DefaultLivenessChecker, DefaultBeadsStatusChecker, DefaultCompletionIndicatorChecker
- Move: CleanableWorkspace type, isEmptyInvestigation

**Phase 3: account_cmd.go + port_cmd.go** (P2 - ~470 lines combined)
- account_cmd.go: accountCmd with list/switch/add/remove sub-commands
- port_cmd.go: portCmd with allocate/list/release/tmuxinator sub-commands

**Phase 4: Remaining commands** (P3 - ~530 lines)
- send_cmd.go: sendCmd, runSend, sendViaOpenCodeAPI, sendViaTmux (~120 lines)
- agent_utils_cmd.go: tailCmd, questionCmd, abandonCmd, retriesCmd (~350 lines)
  OR individual files if preferred

**Target State:**
```
cmd/orch/main.go  (~300 lines) - rootCmd, versionCmd, monitorCmd, usageCmd, main()
```

### If Spawn Follow-up
**Issue:** Split cmd/orch/main.go Phase 1: Extract complete_cmd.go
**Skill:** feature-impl
**Context:**
```
Extract complete command and related helpers from main.go to complete_cmd.go following
the established pattern in spawn_cmd.go. See investigation at
.kb/investigations/2026-01-04-inv-cmd-orch-main-go-49.md for full design.
Start by moving formatDuration and extractProjectDirFromWorkspace to shared.go.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should main_test.go tests move with their command functions? (Needs investigation)
- Would extracting shared state to pkg/cli or similar improve testability? (Future consideration)
- Should commands like `retries` that are small but distinct get their own file? (Team preference)

**Areas worth exploring further:**
- Test file organization after extraction
- Whether any commands would benefit from pkg/ extraction for unit testing

**What remains unclear:**
- Exact test coverage impact of refactoring
- Whether extracting to pkg/ would improve code reuse with other tools

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-arch-cmd-orch-main-04jan/`
**Investigation:** `.kb/investigations/2026-01-04-inv-cmd-orch-main-go-49.md`
**Beads:** `bd show orch-go-ii10`
