# Session Synthesis

**Agent:** og-inv-map-main-go-03jan
**Issue:** untracked (ad-hoc spawn)
**Duration:** 2026-01-03 17:00 → 2026-01-03 18:00
**Outcome:** success

---

## TLDR

Mapped the 4964-line main.go god object, identified 10 command domains with clear boundaries, and created an actionable incremental refactoring plan to split it into 8 domain files plus shared utilities without circular import risks.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-03-inv-map-main-go-command-dependencies.md` - Complete investigation with command mapping, shared utility analysis, and implementation recommendations

### Files Modified
- None (investigation only)

### Commits
- Investigation checkpoint commit

---

## Evidence (What Was Observed)

- main.go contains 4964 lines with 26 cobra.Command definitions and 85+ functions (verified: `wc -l`, `grep "^func "`)
- 10 command groups identified: spawn, agent-ops, status, complete, clean, account, port, utility (verified: code analysis)
- 9 shared utility functions used across multiple domains: extractBeadsIDFromTitle, findWorkspaceByBeadsID, truncate, etc. (verified: `grep -rn` across cmd/orch/)
- Existing extracted files (daemon.go, focus.go, review.go) follow consistent pattern: command + flags + init + run* functions (verified: `head -150 cmd/orch/daemon.go`)
- All imports from pkg/ packages, never cross-cmd files - no circular import risk (verified: import statement analysis)

### Tests Run
```bash
# Build verification
go build ./cmd/orch
# SUCCESS - proves current code compiles

# Line counts
wc -l cmd/orch/*.go | sort -n
# 4964 cmd/orch/main.go
# 26176 total
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-03-inv-map-main-go-command-dependencies.md` - Complete dependency map and split strategy

### Decisions Made
- Decision 1: Split within package main (not subpackages) because all existing extractions follow this pattern and it avoids import complexity
- Decision 2: Create shared.go for cross-domain utilities because 9 functions are used by multiple command domains
- Decision 3: Incremental extraction (one domain at a time) because it's lower risk and allows testing after each phase

### Constraints Discovered
- Constraint 1: Multiple init() functions per package are fine in Go, but flag registration must happen before command registration in root init()
- Constraint 2: Existing test files (main_test.go) may need adjustment after splitting

### Externalized via `kn`
- Not applicable (investigation only, no operational decisions)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up

**Issue 1: Create shared.go with utility functions**
**Skill:** feature-impl
**Context:**
```
Extract 9 shared utility functions from main.go to cmd/orch/shared.go:
extractBeadsIDFromTitle, extractSkillFromTitle, extractBeadsIDFromWindowName,
extractSkillFromWindowName, extractProjectFromBeadsID, findWorkspaceByBeadsID,
resolveSessionID, truncate, findTmuxWindowByIdentifier.
These are used across spawn, status, send, tail, question, abandon, complete commands.
```

**Issue 2: Extract spawn_cmd.go (~750 lines)**
**Skill:** feature-impl
**Context:**
```
Move spawn and work commands from main.go to cmd/orch/spawn_cmd.go following
the daemon.go pattern. Includes: spawnCmd, workCmd, all spawn* flags, spawn init(),
runSpawnWithSkill, runSpawnInline, runSpawnHeadless, runSpawnTmux, and related helpers.
Depends on shared.go being completed first.
```

**Issue 3: Extract status_cmd.go (~500 lines)**
**Skill:** feature-impl
**Context:**
```
Move status command from main.go to cmd/orch/status_cmd.go. Includes: statusCmd,
status flags, runStatus, SwarmStatus/AccountUsage/AgentInfo/StatusOutput types,
getAccountUsage, printSwarmStatus*, getAgentStatus, abbreviateSkill, formatToken*.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should any utilities move to pkg/ for external use? (e.g., extractBeadsIDFromTitle could be useful in tests)
- Could agent_ops commands (send/tail/question/abandon) share more code via a common lookup pattern?

**Areas worth exploring further:**
- Test file structure after splitting (main_test.go is 1758 lines)
- Whether status display logic should move to a pkg/display package

**What remains unclear:**
- Optimal file naming convention (spawn_cmd.go vs spawn.go vs cmd_spawn.go)
- Whether port commands warrant their own file or could stay in main.go

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-map-main-go-03jan/`
**Investigation:** `.kb/investigations/2026-01-03-inv-map-main-go-command-dependencies.md`
**Beads:** N/A (untracked spawn)
