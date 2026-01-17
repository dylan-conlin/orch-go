<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The build failed because `GetActiveAgents` and `ResumeAgentByBeadsID` functions were called in daemon.go but never implemented.

**Evidence:** `make build` failed with "undefined: GetActiveAgents" and "undefined: ResumeAgentByBeadsID" at daemon.go:1092 and daemon.go:1146.

**Knowledge:** The tiered stuck agent recovery feature added config and struct fields but left the implementation functions unfinished.

**Next:** Fixed - created pkg/daemon/recovery.go with the two missing functions.

**Promote to Decision:** recommend-no (tactical fix, not architectural)

---

# Investigation: Fix Build Error Undefined GetActiveAgents

**Question:** Why does the build fail with undefined GetActiveAgents and ResumeAgentByBeadsID in pkg/daemon/daemon.go?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-debug-fix-build-error-17jan-7ed5
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Functions Called But Never Defined

**Evidence:** The daemon.go file has `RunPeriodicRecovery()` at line 1086 which calls:
- Line 1092: `agents, err := GetActiveAgents()`
- Line 1146: `if err := ResumeAgentByBeadsID(agent.BeadsID)`

Neither function exists anywhere in the codebase (grep confirmed).

**Source:** `pkg/daemon/daemon.go:1092, 1146`

**Significance:** The tiered stuck agent recovery feature was partially implemented - config fields and the recovery loop were added, but the two helper functions were not created.

---

### Finding 2: Config and Struct Fields Already Exist

**Evidence:** The daemon.go already has:
- `Config.RecoveryEnabled`, `Config.RecoveryInterval`, `Config.RecoveryIdleThreshold`, `Config.RecoveryRateLimit`
- `Daemon.lastRecovery`, `Daemon.resumeAttempts` fields
- Default values in `DefaultConfig()`

**Source:** `pkg/daemon/daemon.go:69-84, 177-185`

**Significance:** Only the implementation functions were missing - the infrastructure was ready.

---

### Finding 3: Existing Resume Infrastructure

**Evidence:** `cmd/orch/resume.go` has `runResumeByBeadsID()` function that:
- Finds workspace by beads ID
- Looks up session ID from workspace or OpenCode API
- Generates resume prompt
- Sends message via OpenCode API
- Logs resume event

**Source:** `cmd/orch/resume.go:140-217`

**Significance:** The resume logic exists but in the cmd package. Created new implementation in daemon package to avoid circular imports.

---

## Synthesis

**Key Insights:**

1. **Incomplete Feature Implementation** - The tiered stuck agent recovery feature was designed and partially implemented but the two key functions were left as TODO stubs.

2. **Pattern Follows Existing Daemon Patterns** - The recovery uses the same `ShouldRun/RunPeriodic` pattern as reflection and cleanup.

**Answer to Investigation Question:**

The build fails because the tiered stuck agent recovery feature added code that calls `GetActiveAgents()` and `ResumeAgentByBeadsID()` but never implemented these functions. Created `pkg/daemon/recovery.go` with:
- `ActiveAgent` struct for recovery purposes
- `GetActiveAgents()` that queries beads for in_progress issues and their phase timestamps
- `ResumeAgentByBeadsID()` that finds agent session and sends resume prompt

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes (verified: `make build` succeeds)
- ✅ Daemon tests pass (verified: `go test ./pkg/daemon/...` - 7.6s, all pass)
- ✅ Functions compile correctly (verified: no build errors)

**What's untested:**

- ⚠️ Actual recovery behavior (would need running daemon with stuck agent)
- ⚠️ Cross-project agent recovery (different beads database)

**What would change this:**

- If OpenCode API changes, the resume logic would need updating
- If beads comment format changes, phase parsing would break

---

## References

**Files Examined:**
- `pkg/daemon/daemon.go:1080-1180` - The RunPeriodicRecovery function with undefined calls
- `cmd/orch/resume.go` - Existing resume implementation for reference
- `pkg/verify/beads_api.go` - Phase parsing and beads API
- `pkg/daemon/completion_processing.go` - Pattern for querying beads and phase

**Files Created:**
- `pkg/daemon/recovery.go` - New file with ActiveAgent struct, GetActiveAgents(), ResumeAgentByBeadsID()

**Commands Run:**
```bash
# Verify build
make build

# Run daemon tests
go test ./pkg/daemon/... -count=1
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-17-inv-implement-tiered-stuck-agent-recovery.md` - The feature design that left these functions unimplemented

---

## Investigation History

**2026-01-17 21:50:** Investigation started
- Initial question: Why does build fail with undefined GetActiveAgents and ResumeAgentByBeadsID?
- Context: Build error blocking development

**2026-01-17 21:55:** Root cause identified
- Functions called but never implemented
- Recovery feature partially done

**2026-01-17 22:00:** Investigation completed
- Status: Complete
- Key outcome: Created pkg/daemon/recovery.go with the two missing functions, build passes
