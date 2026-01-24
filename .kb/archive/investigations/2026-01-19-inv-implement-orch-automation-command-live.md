<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented `orch automation list` and `orch automation check` commands to audit custom launchd agents.

**Evidence:** Commands run successfully, detecting 7 agents with 4 issues (3 failures, 1 not loaded).

**Knowledge:** launchctl list provides PID/exit code; plist files contain schedule configuration; howett.net/plist library handles XML plist parsing.

**Next:** Close - implementation complete and tested.

**Promote to Decision:** recommend-no (tactical implementation, not architectural pattern)

---

# Investigation: Implement Orch Automation Command Live

**Question:** How to implement a live audit command for custom launchd agents?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** Worker agent (orch-go-vbqmk)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: launchctl list provides runtime status

**Evidence:** Output format: `PID\tExitCode\tLabel` where PID is `-` if not running.

**Source:** `launchctl list | grep -E "com\.(dylan|user|orch|cdd)\."`

**Significance:** Can determine loaded/running status and last exit code without parsing plist.

---

### Finding 2: Plist files contain schedule configuration

**Evidence:**
- `StartCalendarInterval` for cron-like schedules (Hour, Minute, Weekday, Day)
- `StartInterval` for repeat intervals in seconds
- `RunAtLoad` for startup behavior
- `KeepAlive` for daemon behavior

**Source:** ~/Library/LaunchAgents/com.dylan.disk-cleanup.plist

**Significance:** Multiple schedule types need distinct parsing and display.

---

### Finding 3: Seven custom agents exist with three failure patterns

**Evidence:**
- com.cdd.artifact-watcher: exit 78
- com.user.claude-version-monitor: exit 1
- com.user.tmuxinator: exit 1
- com.orch.daemon: not loaded

**Source:** `./orch automation check`

**Significance:** Live audit immediately surfaces issues that static docs would miss.

---

## Synthesis

**Key Insights:**

1. **Dual data sources** - Plist files for configuration, launchctl for runtime status.

2. **Schedule variety** - Calendar intervals, repeat intervals, file triggers, and on-load all need distinct handling.

3. **Failure detection** - Non-zero exit codes and "not loaded" states are the primary health indicators.

**Answer to Investigation Question:**

Implemented via pkg/launchd for parsing and cmd/orch/automation.go for CLI commands. The `list` command shows all agents with their status, and `check` runs health checks that flag issues and return exit code 1 for scripting.

---

## Structured Uncertainty

**What's tested:**

- ✅ List command shows all 7 custom agents (verified: ran `./orch automation list`)
- ✅ Check command detects 4 issues (verified: ran `./orch automation check`)
- ✅ JSON output works for both commands (verified: ran with `--json` flag)
- ✅ Unit tests pass for schedule parsing (verified: `go test ./pkg/launchd/...`)

**What's untested:**

- ⚠️ Behavior on systems without custom launchd agents
- ⚠️ Behavior on Linux (launchd is macOS-specific)

**What would change this:**

- Finding would be wrong if launchctl output format differs on other macOS versions

---

## References

**Files Created:**
- `pkg/launchd/launchd.go` - Agent struct, plist parsing, launchctl integration
- `pkg/launchd/launchd_test.go` - Unit tests for parsing logic
- `cmd/orch/automation.go` - CLI commands (list, check)

**Commands Run:**
```bash
# List launchd agents
./orch automation list

# Health check
./orch automation check

# JSON output
./orch automation list --json
```

---

## Investigation History

**2026-01-19 14:03:** Investigation started
- Initial question: How to implement live audit of launchd automation jobs?
- Context: Issue orch-go-vbqmk - replacing static inventory docs with live audit

**2026-01-19 14:20:** Implementation complete
- Status: Complete
- Key outcome: `orch automation list` and `orch automation check` implemented and tested
