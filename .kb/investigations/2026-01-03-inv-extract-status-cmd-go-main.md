<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully extracted status command from main.go (~500 lines) into status_cmd.go (815 lines).

**Evidence:** Build passes, all tests pass, main.go reduced from 4964 to 3906 lines (~1058 line reduction).

**Knowledge:** Extraction follows established pattern: command definition + flags + init + run function + related types/helpers stay together. No circular import issues since all files are in package main.

**Next:** Phase 3 complete. Consider next extraction phases (spawn_cmd.go, agent_ops.go, etc.) per the mapping investigation.

---

# Investigation: Extract Status Cmd Go Main

**Question:** How to extract status command (~500 lines) from main.go to status_cmd.go following established patterns?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Feature agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Status command included substantial related code

**Evidence:** The status command includes:
- Command definition (statusCmd) and flags (statusJSON, statusAll, statusProject)
- init() function for flag registration
- Types: SwarmStatus, AccountUsage, AgentInfo, StatusOutput
- Main function: runStatus()
- Display helpers: printSwarmStatus, printSwarmStatusWithWidth, printAgentsWideFormat, printAgentsNarrowFormat, printAgentsCardFormat
- Formatting helpers: getAgentStatus, abbreviateSkill, formatTokenCount, formatTokenStats, formatTokenStatsCompact
- Data helpers: getAccountUsage, getPhaseAndTask, extractDateFromWorkspaceName

**Source:** cmd/orch/main.go lines 321-351 (command), 2001-2765 (types and functions)

**Significance:** The extraction is larger than the estimated 500 lines because of the many helper functions for formatting and display.

---

### Finding 2: Existing shared.go already contains cross-command utilities

**Evidence:** The shared.go file (269 lines) already contains:
- truncate()
- extractBeadsIDFromTitle()
- extractSkillFromTitle()
- extractBeadsIDFromWindowName()
- extractSkillFromWindowName()
- extractProjectFromBeadsID()
- findWorkspaceByBeadsID()
- resolveSessionID()
- findTmuxWindowByIdentifier()

**Source:** cmd/orch/shared.go

**Significance:** The status command can reuse these existing utilities. No need to duplicate or move them.

---

### Finding 3: Cross-file dependencies work seamlessly in package main

**Evidence:** 
- formatDuration() is defined in wait.go and used by status_cmd.go
- extractProjectDirFromWorkspace() is defined in review.go and used by status_cmd.go
- readDevModeFile() is defined in mode.go and used by status_cmd.go
- All compile and work without import statements because they're in the same package

**Source:** Build succeeds with `go build ./cmd/orch`

**Significance:** Go's package-level visibility allows splitting code across files without managing imports between them.

---

## Synthesis

**Key Insights:**

1. **Existing patterns work well** - Following the daemon.go/focus.go pattern (command + flags + init + run + helpers in one file) creates clean, maintainable splits.

2. **Shared utilities are already centralized** - The shared.go file already exists and contains cross-command utilities, so status_cmd.go can use them without duplication.

3. **Line count reduction is significant** - main.go reduced by ~1058 lines (21%), and the status_cmd.go at 815 lines is a reasonable size for a single-concern file.

**Answer to Investigation Question:**

The extraction was completed successfully by:
1. Creating status_cmd.go with the statusCmd definition, flags, init(), and all status-related types and functions
2. Removing the same code from main.go
3. Keeping statusCmd registration in main.go's init() (rootCmd.AddCommand(statusCmd))
4. Using existing utilities from shared.go and other command files

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes (verified: `go build ./cmd/orch`)
- ✅ All tests pass (verified: `go test ./cmd/orch/...` and `go test ./...`)
- ✅ No duplicate function definitions (verified: build would fail)

**What's untested:**

- ⚠️ Runtime behavior identical to before (not manually tested, but tests pass)
- ⚠️ Edge cases in status display (not manually verified)

**What would change this:**

- Finding would need revision if runtime tests revealed behavior changes
- Finding would need revision if other commands had hidden dependencies on status types

---

## Implementation Recommendations

**Purpose:** Document the implementation approach that was taken.

### Recommended Approach ⭐

**Direct file extraction** - Move all status-related code to a new status_cmd.go file.

**Why this approach:**
- Follows existing pattern (daemon.go, focus.go, review.go)
- Keeps related code together
- No import management needed within package main

**Trade-offs accepted:**
- status_cmd.go is 815 lines (larger than ideal ~500 lines)
- Why acceptable: All code is related to the status command, so it makes sense to keep it together

**Implementation sequence:**
1. Create status_cmd.go with all status-related code
2. Remove same code from main.go
3. Verify build and tests

### Implementation Details

**What was implemented:**
- Created status_cmd.go (815 lines)
- Removed ~1058 lines from main.go (now 3906 lines from original 4964)

**Success criteria:**
- ✅ Build succeeds
- ✅ All tests pass
- ✅ main.go reduced by ~500+ lines

---

## References

**Files Examined:**
- cmd/orch/main.go - Primary target for extraction
- cmd/orch/shared.go - Cross-command utilities
- cmd/orch/wait.go - formatDuration() definition
- cmd/orch/review.go - extractProjectDirFromWorkspace() definition
- cmd/orch/mode.go - readDevModeFile() definition
- .kb/investigations/2026-01-03-inv-map-main-go-command-dependencies.md - Mapping investigation for extraction plan

**Commands Run:**
```bash
# Build verification
go build ./cmd/orch

# Test verification
go test ./cmd/orch/...
go test ./...

# Line count check
wc -l cmd/orch/main.go cmd/orch/status_cmd.go
```

---

## Investigation History

**2026-01-03:** Investigation started
- Initial question: Extract status command from main.go
- Context: Phase 3 of main.go refactoring

**2026-01-03:** Implementation completed
- Status: Complete
- Key outcome: status_cmd.go created (815 lines), main.go reduced by ~1058 lines (21%)
