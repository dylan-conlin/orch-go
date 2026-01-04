<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** main.go (2705 lines) should be split into 9 focused subcommand files following the established spawn_cmd.go/status_cmd.go pattern, reducing it to ~300 lines (root + version + init).

**Evidence:** Analyzed existing refactoring (spawn_cmd.go: 1500 lines, status_cmd.go: 902 lines, shared.go: 301 lines) and identified 12 command groups still in main.go totaling ~2400 lines.

**Knowledge:** The codebase already has a proven pattern: `{command}_cmd.go` for large commands with handlers, sub-commands in same file with `init()` registration. Shared utilities go in `shared.go`.

**Next:** Implement in 4 phases: (1) complete_cmd.go, (2) clean_cmd.go, (3) account_cmd.go + port_cmd.go, (4) small commands (send, tail, question, abandon, retries).

---

# Investigation: Splitting cmd/orch/main.go into Focused Subcommand Files

**Question:** How should cmd/orch/main.go (2705 lines, 49 fix commits in 28 days) be split into focused subcommand files following cobra patterns?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None - Ready for implementation
**Status:** Complete

---

## Findings

### Finding 1: Existing Refactoring Pattern Established

**Evidence:** The codebase already has 3 extracted command files that demonstrate the pattern:
- `spawn_cmd.go` (1500 lines): spawn command + work command + all spawn modes (inline, headless, tmux) + gap analysis
- `status_cmd.go` (902 lines): status command + output formatting + agent info collection
- `shared.go` (301 lines): 9 shared utility functions used across commands

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go:1-8` (package doc comment)
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/shared.go:1-6` (package doc comment)
- Git history: commits `c129bf15`, `529b2e86`, `5ce7f2f8`

**Significance:** This is the proven pattern. Each command file:
1. Has package doc comment explaining scope
2. Contains cobra command definition(s)
3. Contains handler functions (runXxx)
4. Uses `init()` for flag registration
5. Sub-commands stay with parent (e.g., daemonCmd contains daemonRunCmd, daemonPreviewCmd)

---

### Finding 2: 12 Command Groups Remain in main.go

**Evidence:** Commands still in main.go (2705 lines total):

| Command | Lines (est) | Complexity | Priority |
|---------|------------|------------|----------|
| completeCmd | ~400 | High (verification, UI approval, auto-rebuild) | P1 |
| cleanCmd | ~350 | Medium (workspace scanning, phantom detection) | P1 |
| accountCmd + subs | ~200 | Medium (4 sub-commands) | P2 |
| portCmd + subs | ~230 | Low (4 sub-commands) | P2 |
| sendCmd | ~100 | Low | P3 |
| tailCmd | ~90 | Low | P3 |
| questionCmd | ~80 | Low | P3 |
| abandonCmd | ~160 | Low | P3 |
| retriesCmd | ~80 | Low | P3 |
| versionCmd | ~60 | Trivial | P4 |
| monitorCmd | ~30 | Trivial | P4 |
| usageCmd | ~20 | Trivial | Stay |

**Source:** `grep -n "var.*Cmd = &cobra.Command" /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go`

**Significance:** The largest commands (complete, clean) should be extracted first as they have the most complexity and likely the most fix commits. Small trivial commands (version, monitor, usage) can stay in main.go.

---

### Finding 3: Command Grouping Patterns in Codebase

**Evidence:** The codebase shows two patterns for command organization:

**Pattern A - Single File with Sub-commands (e.g., daemon.go):**
```go
// daemon.go
var daemonCmd = &cobra.Command{...}
var daemonRunCmd = &cobra.Command{...}
var daemonPreviewCmd = &cobra.Command{...}
func init() {
    daemonCmd.AddCommand(daemonRunCmd)
    daemonCmd.AddCommand(daemonPreviewCmd)
}
```

**Pattern B - Split by Handler Domain (e.g., serve_*.go):**
```
serve.go       - Main command, routing, small handlers
serve_agents.go - /api/agents handler (large, complex)
serve_beads.go  - /api/beads handlers
serve_errors.go - /api/errors handler
```

**Source:** 
- `cmd/orch/daemon.go:17-31` (sub-commands in same file)
- `cmd/orch/serve*.go` (domain-split pattern)

**Significance:** For CLI commands, Pattern A (sub-commands in same file) is preferred. Pattern B is only for HTTP handlers where each domain has significant logic.

---

### Finding 4: Utility Functions Scattered Through main.go

**Evidence:** Several utility functions in main.go should be evaluated for extraction to `shared.go`:

Already in shared.go:
- `truncate()`, `extractBeadsIDFromTitle()`, `extractSkillFromTitle()`
- `extractBeadsIDFromWindowName()`, `extractSkillFromWindowName()`
- `extractProjectFromBeadsID()`, `findWorkspaceByBeadsID()`
- `resolveSessionID()`, `findTmuxWindowByIdentifier()`, `resolveShortBeadsID()`

Still in main.go that should move:
- `formatDuration()` - used by status, complete, and other commands
- `extractProjectDirFromWorkspace()` - used by status and complete
- `hasGoChangesInRecentCommits()` - specific to complete but used by auto-rebuild

**Source:** 
- `cmd/orch/shared.go:16-301`
- `cmd/orch/main.go` (various utility functions)

**Significance:** Keep shared utilities in `shared.go`. Command-specific utilities should stay with their command file (e.g., `hasGoChangesInRecentCommits` stays with complete_cmd.go).

---

## Synthesis

**Key Insights:**

1. **Established Pattern Works** - The spawn_cmd.go/status_cmd.go extraction provides a clear template. Each command file is self-contained with command definition, handlers, and init().

2. **High-Impact Commands First** - complete_cmd.go and clean_cmd.go should be extracted first as they are the largest (~750 lines combined) and likely sources of many fix commits.

3. **Group Related Commands** - Port commands (4 sub-commands) and account commands (4 sub-commands) should each be in single files following the daemon.go pattern.

4. **Minimal main.go** - After refactoring, main.go should contain only: rootCmd, versionCmd (trivial), monitorCmd (trivial), usageCmd (trivial), and the main() function. Target: ~300 lines.

**Answer to Investigation Question:**

Split main.go into focused files following this organization:

```
cmd/orch/
├── main.go           (~300 lines) - rootCmd, version, monitor, usage, main()
├── spawn_cmd.go      (1500 lines) - EXISTING
├── status_cmd.go     (902 lines)  - EXISTING  
├── shared.go         (350 lines)  - EXISTING + formatDuration, extractProjectDirFromWorkspace
├── complete_cmd.go   (~450 lines) - NEW: complete + verification + auto-rebuild
├── clean_cmd.go      (~400 lines) - NEW: clean + workspace scanning + phantom detection
├── account_cmd.go    (~220 lines) - NEW: account + list/switch/add/remove
├── port_cmd.go       (~250 lines) - NEW: port + allocate/list/release/tmuxinator
├── send_cmd.go       (~120 lines) - NEW: send + tmux/api resolution
├── agent_utils_cmd.go (~350 lines) - NEW: tail + question + abandon + retries (related commands)
├── daemon.go         (577 lines)  - EXISTING (already extracted)
├── focus.go          (418 lines)  - EXISTING
├── serve.go          (326 lines)  - EXISTING
├── ... (other existing files)
```

---

## Structured Uncertainty

**What's tested:**

- ✅ spawn_cmd.go extraction works correctly (verified: compiles and runs)
- ✅ status_cmd.go extraction works correctly (verified: compiles and runs)
- ✅ shared.go utilities accessible from all command files (verified: package main scope)
- ✅ Go handles cross-file function visibility within package main (verified: prior decision)

**What's untested:**

- ⚠️ Exact line counts for each extracted file (estimated based on code analysis)
- ⚠️ Whether agent_utils_cmd.go grouping is optimal (could be 4 separate files)
- ⚠️ Build time impact of more files in package main (likely negligible)

**What would change this:**

- If complete_cmd.go dependencies require significant shared code, may need more utilities in shared.go
- If agent commands (tail/question/abandon) diverge significantly, split into separate files
- If testability requirements emerge, may need interface extraction to pkg/

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Phased Extraction** - Extract commands in 4 phases over 4 sessions, starting with highest-impact files.

**Why this approach:**
- Reduces risk of breaking changes (one focus area per session)
- Enables incremental testing after each extraction
- Follows established pattern proven by prior extractions

**Trade-offs accepted:**
- Multiple sessions instead of one big refactor (acceptable: reduces risk)
- Some temporary code duplication during transition (acceptable: cleaned up at end)

**Implementation sequence:**

1. **Phase 1: complete_cmd.go** (Highest priority - 400+ lines, complex)
   - Extract completeCmd, flags, runComplete, verification helpers
   - Move hasGoChangesInRecentCommits, detectNewCLICommands, runAutoRebuild
   - Keep invalidateServeCache (related to complete)
   - Add `formatDuration` to shared.go (used by multiple commands)

2. **Phase 2: clean_cmd.go** (Second priority - 350+ lines)
   - Extract cleanCmd, flags, runClean
   - Move findCleanableWorkspaces, cleanOrphanedDiskSessions
   - Move cleanPhantomWindows, archiveEmptyInvestigations
   - Move DefaultLivenessChecker, DefaultBeadsStatusChecker types

3. **Phase 3: account_cmd.go + port_cmd.go** (Medium priority)
   - Extract accountCmd with all 4 sub-commands (list, switch, add, remove)
   - Extract portCmd with all 4 sub-commands (allocate, list, release, tmuxinator)

4. **Phase 4: Remaining small commands** (Lower priority)
   - Extract sendCmd (with tmux/api resolution helpers)
   - Group tail, question, abandon, retries into agent_utils_cmd.go
   - OR extract as individual files if complexity warrants

### Alternative Approaches Considered

**Option B: Single Big Refactor**
- **Pros:** Done in one session, atomic change
- **Cons:** High risk of breakage, difficult to test incrementally, hard to review
- **When to use instead:** If CI/CD is very robust and team wants single PR

**Option C: Extract Everything to Individual Files**
- **Pros:** Maximum granularity, easy to find any command
- **Cons:** 12+ new files, some very small (20-30 lines), overhead
- **When to use instead:** If team prefers one-command-per-file convention

**Rationale for recommendation:** Phased approach with related command grouping balances risk reduction with maintainability. The 4-phase plan maps to ~4 work sessions, each producing a testable increment.

---

### Implementation Details

**What to implement first:**
- `formatDuration()` and `extractProjectDirFromWorkspace()` to shared.go
- complete_cmd.go extraction (largest, most complex, most fixes)

**Things to watch out for:**
- ⚠️ Global variables like `serverURL` are defined in main.go - keep there or move to a config file
- ⚠️ `DefaultServePort` constant is used by both serve.go and main.go - defined in serve.go
- ⚠️ Some init() functions register with rootCmd - ensure order doesn't matter
- ⚠️ Test files (main_test.go) may need updates if they access extracted functions

**Areas needing further investigation:**
- Whether main_test.go tests need to move with their commands
- Whether any pkg/ refactoring would improve testability

**Success criteria:**
- ✅ main.go reduced to ~300 lines
- ✅ All tests pass after each phase
- ✅ `make build` succeeds with no new warnings
- ✅ `orch version`, `orch status`, `orch spawn`, `orch complete`, `orch clean` all work

---

## References

**Files Examined:**
- `cmd/orch/main.go` - The god file being analyzed
- `cmd/orch/spawn_cmd.go` - Pattern A: large command with sub-commands
- `cmd/orch/status_cmd.go` - Pattern A: large command with formatting
- `cmd/orch/shared.go` - Shared utilities pattern
- `cmd/orch/daemon.go` - Pattern A: parent with sub-commands
- `cmd/orch/serve.go` - HTTP handler organization
- `cmd/orch/focus.go` - Medium-sized command file pattern

**Commands Run:**
```bash
# Count total lines
wc -l /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go
# Output: 2705

# List cobra commands in main.go
grep -n "var.*Cmd = &cobra.Command" main.go
# Found 22 command definitions

# Recent commits to main.go
git log --oneline -n 60 --since="28 days ago" -- cmd/orch/main.go
# Found 49+ commits (confirming god file status)

# Line counts for all files
wc -l /Users/dylanconlin/Documents/personal/orch-go/cmd/orch/*.go | sort -n
```

**External Documentation:**
- Cobra library documentation (command organization patterns)
- Go package conventions (all files in package main share scope)

**Related Artifacts:**
- **Decision:** Function extraction within package main requires no import changes (from kb context)
- **Investigation:** Prior spawn_cmd.go refactoring commits: c129bf15, 529b2e86, 5ce7f2f8

---

## Investigation History

**[2026-01-04 10:00]:** Investigation started
- Initial question: How to split main.go god file into focused subcommand files
- Context: 49 fix commits in 28 days indicates high churn on a large file

**[2026-01-04 10:30]:** Analysis complete
- Identified existing pattern in spawn_cmd.go, status_cmd.go, shared.go
- Catalogued 12 command groups remaining in main.go
- Designed 4-phase extraction plan

**[2026-01-04 11:00]:** Investigation completed
- Status: Complete
- Key outcome: 4-phase extraction plan with specific file targets and ~300 line main.go goal
