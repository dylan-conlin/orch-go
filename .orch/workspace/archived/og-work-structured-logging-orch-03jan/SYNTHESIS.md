# Session Synthesis

**Agent:** og-work-structured-logging-orch-03jan
**Issue:** orch-go-enkk
**Duration:** 2026-01-03 → 2026-01-03
**Outcome:** success

---

## TLDR

Analyzed 815 fmt.Printf calls and produced decision record recommending stdlib log/slog with hybrid approach: structured JSON logging for daemon/background processes, keep fmt.Printf for CLI user-facing output.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-03-inv-structured-logging-orch-go-808.md` - Investigation with findings
- `.kb/decisions/2026-01-03-structured-logging-orch-go.md` - Decision record with implementation plan

### Files Modified
- None (design session, no code changes)

### Commits
- (to be committed)

---

## Evidence (What Was Observed)

- 815 total fmt.Print* calls across codebase
- 247 in cmd/orch/main.go (CLI user output with emojis, prompts)
- 15 in pkg/daemon/daemon.go (DEBUG prints guarded by Verbose flag)
- 125 stderr prints for error handling
- 86 existing log.* calls scattered
- pkg/events/logger.go already implements JSONL pattern for lifecycle events
- Go 1.24 enables stdlib log/slog (added in Go 1.21)

### Tests Run
```bash
# Count fmt.Print calls
rg 'fmt\.Print' --type go -c | awk -F: '{sum += $2} END {print sum}'
# Result: 815

# Find daemon DEBUG pattern
rg 'fmt\.Printf.*DEBUG' --type go -l
# Result: pkg/daemon/daemon.go
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-03-inv-structured-logging-orch-go-808.md` - Analysis of current logging patterns
- `.kb/decisions/2026-01-03-structured-logging-orch-go.md` - Structured logging strategy

### Decisions Made
- **Use stdlib slog:** Zero dependencies, Go ecosystem direction, right-sized for CLI tool
- **Hybrid approach:** Daemon gets JSON to file, CLI keeps fmt.Printf
- **Don't convert CLI output:** User-facing output with emojis/formatting is not log data

### Constraints Discovered
- CLI output and operational logging are fundamentally different concerns
- Events.jsonl is for lifecycle events, daemon.log would be for debug/operational logs
- Daemon runs via launchd - stdout logging is lost, needs file destination

### Externalized via `kn`
- (none needed - decision record captures everything)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement structured logging with slog
**Skill:** feature-impl
**Context:**
```
Implement stdlib log/slog per decision record .kb/decisions/2026-01-03-structured-logging-orch-go.md.

Phase 1: Create pkg/log/log.go with Init(daemonMode bool)
Phase 2: Replace pkg/daemon/daemon.go DEBUG prints with slog
Phase 3: Replace pkg/opencode/service.go, pkg/account/oauth.go logging
Phase 4: Add --log-level flag to daemon command

Do NOT convert cmd/orch/*.go - that's intentionally user-facing output.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Log rotation for daemon.log - may need logrotate or similar
- Should events.jsonl and daemon.log be consolidated into one file?
- Could daemon.log feed into orch patterns for automated analysis?

**Areas worth exploring further:**
- Adding caller info (file:line) to log entries
- Structured error handling (errors.As/Is patterns)

**What remains unclear:**
- Whether current launchd setup captures stdout (need to verify)
- Performance impact of JSON serialization (assumed negligible)

---

## Session Metadata

**Skill:** design-session
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-work-structured-logging-orch-03jan/`
**Investigation:** `.kb/investigations/2026-01-03-inv-structured-logging-orch-go-808.md`
**Beads:** `bd show orch-go-enkk`
