## Summary (D.E.K.N.)

**Delta:** Use stdlib `log/slog` for structured logging with distinct handling for daemon logs (JSON to file) vs CLI output (keep fmt for user-facing).

**Evidence:** 815 fmt.Print calls analyzed - 247 in main.go CLI, 71 in pkg/ (daemon/services). Only daemon/background needs structured logging; CLI output is user-facing.

**Knowledge:** CLI output != operational logging. Don't over-engineer user-facing output; focus structured logging on daemon/background processes that need machine-parseable logs.

**Next:** Implement in phases: 1) Create pkg/log wrapper, 2) Replace daemon fmt.Printf DEBUG with slog, 3) Leave CLI fmt.Printf for user output.

---

# Decision: Structured Logging for orch-go

**Date:** 2026-01-03
**Status:** Accepted

---

## Context

The orch-go codebase has 815 raw `fmt.Printf` calls that need evaluation for structured logging replacement. The daemon runs as a background service (via launchd) overnight, making log analysis critical for debugging. Current approach:

- **pkg/events/logger.go**: Existing JSONL logger for agent lifecycle events (spawn, complete, error) - works well
- **Daemon debug output**: Uses `fmt.Printf("DEBUG: ...")` guarded by `d.Config.Verbose`
- **CLI commands**: User-facing output with emojis, status messages, interactive prompts
- **Error handling**: Mix of `fmt.Fprintf(os.Stderr, ...)` and `return fmt.Errorf(...)`

**Key insight from analysis:** Not all 815 calls need structured logging. Breaking down by purpose:

| Category | Count | Location | Needs Structured? |
|----------|-------|----------|-------------------|
| CLI user output | ~700 | cmd/orch/*.go | No - keep fmt |
| Daemon debug | ~15 | pkg/daemon/ | Yes - needs slog |
| Background services | ~25 | pkg/*.go | Yes - needs slog |
| Error returns | ~75 | everywhere | No - keep as-is |

---

## Options Considered

### Option A: stdlib log/slog (Recommended)
- **Pros:** 
  - Zero dependencies (already in Go 1.21+, we use 1.24)
  - Designed by Go team, consistent with ecosystem direction
  - JSON handler for daemon, Text handler for CLI debug
  - Structured fields via slog.Attr
  - Built-in log levels (Debug, Info, Warn, Error)
  - Easy to extend with custom handlers
- **Cons:** 
  - Slightly more verbose than zerolog for simple cases
  - No caller info by default (easy to add)

### Option B: zerolog
- **Pros:** 
  - Zero-allocation, very fast
  - Fluent API: `log.Debug().Str("key", "val").Msg("...")`
  - Popular in high-performance Go
- **Cons:** 
  - External dependency (vs stdlib)
  - Over-engineered for CLI tool (perf gains not needed)
  - Different API style than emerging stdlib standard

### Option C: zap (Uber)
- **Pros:** 
  - Battle-tested at scale
  - Very flexible configuration
  - Sugar API for convenience
- **Cons:** 
  - Heavy dependency (Uber ecosystem)
  - Complex configuration
  - Overkill for CLI orchestration tool

### Option D: Keep fmt with conventions
- **Pros:** 
  - No changes needed
  - Simple
- **Cons:** 
  - No log levels
  - No structured fields for daemon log analysis
  - Can't easily grep/parse daemon.log

---

## Decision

**Chosen:** Option A - stdlib log/slog with hybrid approach

**Rationale:** 
1. **Zero dependencies** - orch-go already has minimal deps, keep it that way
2. **Go ecosystem direction** - slog is the future, adopting early
3. **Right-sized solution** - not over-engineering a CLI tool
4. **Hybrid approach works** - CLI output stays fmt (intentional), only daemon/services get slog

**Trade-offs accepted:**
- CLI output remains fmt.Printf (this is correct - user-facing output shouldn't be structured)
- Not using zerolog's zero-alloc (not needed for CLI tool)

---

## Implementation Strategy

### Phase 1: Create pkg/log wrapper (1 hour)
```go
// pkg/log/log.go
package log

import (
    "log/slog"
    "os"
)

var (
    // Logger is the global structured logger
    Logger *slog.Logger
    
    // DaemonLogger writes JSON to ~/.orch/daemon.log
    DaemonLogger *slog.Logger
)

func Init(daemonMode bool) {
    if daemonMode {
        // JSON to file for daemon
        f, _ := os.OpenFile(daemonLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        Logger = slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        }))
    } else {
        // Text to stderr for CLI debug (only when verbose)
        Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        }))
    }
}
```

### Phase 2: Replace daemon DEBUG prints (2 hours)
Convert:
```go
// Before
if d.Config.Verbose {
    fmt.Printf("  DEBUG: Skipping %s (type %s not spawnable)\n", issue.ID, issue.IssueType)
}

// After
log.Logger.Debug("skipping issue",
    "issue_id", issue.ID,
    "issue_type", issue.IssueType,
    "reason", "type_not_spawnable")
```

### Phase 3: Replace pkg/ service logging (2 hours)
Focus on:
- `pkg/daemon/daemon.go` (15 prints)
- `pkg/opencode/service.go` (5 prints)
- `pkg/account/oauth.go` (8 prints)

### Phase 4: Leave CLI output alone
**Intentionally do NOT convert:**
- `cmd/orch/*.go` user-facing output
- Interactive prompts
- Status displays with emojis
- Progress indicators

---

## Log Levels Strategy

| Level | Use Case | Example |
|-------|----------|---------|
| **Error** | Operation failed, needs attention | Spawn failed, API unreachable |
| **Warn** | Degraded but continuing | Rate limited, retrying |
| **Info** | Normal operations worth noting | Agent spawned, cycle completed |
| **Debug** | Verbose troubleshooting | Issue skipped (reason), capacity check |

**Daemon flag mapping:**
- `--verbose` → slog.LevelDebug
- default → slog.LevelInfo
- Could add `--quiet` → slog.LevelWarn

---

## Daemon vs CLI Output Handling

### Daemon Mode (background, launchd)
- **Destination:** `~/.orch/daemon.log` (JSON lines)
- **Handler:** `slog.NewJSONHandler`
- **Why JSON:** Machine-parseable for `jq`, pattern analysis, error aggregation

### CLI Mode (interactive)
- **Destination:** Stay with fmt.Printf for user output
- **Debug only:** If we need debug in CLI, use slog.TextHandler to stderr
- **Why fmt:** User output with emojis, colors, formatting is not log data

### Existing events.jsonl
- **Keep as-is** - This is for agent lifecycle events, not operational logs
- **Complement, don't replace** - events.jsonl = "what happened", daemon.log = "why/how"

---

## Structured Uncertainty

**What's tested:**
- ✅ 815 fmt.Print calls counted (verified: `rg 'fmt\.Print' --type go -c`)
- ✅ pkg/daemon has 15 DEBUG prints (verified: grep in daemon.go)
- ✅ Go 1.24 has slog (verified: go.mod shows 1.24.0)
- ✅ events.jsonl pattern works (verified: pkg/events/logger.go exists and is used)

**What's untested:**
- ⚠️ Log rotation for daemon.log (not addressed, may need logrotate or similar)
- ⚠️ Performance impact of JSON serialization (assumed negligible for CLI tool)
- ⚠️ Integration with existing events.jsonl (assumed complementary, not tested together)

**What would change this:**
- If daemon needs sub-millisecond logging → consider zerolog
- If we add a web UI that needs log streaming → consider structured API
- If CLI output needs i18n → might reconsider output strategy

---

## Consequences

**Positive:**
- Daemon logs become parseable: `jq '.level == "error"' daemon.log`
- Pattern analysis via `orch patterns` can use structured data
- Debug output cleaner with levels vs "DEBUG:" prefix
- Future-proof with stdlib adoption

**Risks:**
- Small learning curve for contributors
- Need to be disciplined about CLI output vs logging
- Could accidentally over-log (mitigate: review in implementation)

---

## Implementation Checklist

- [ ] Create `pkg/log/log.go` with Init() for daemon/CLI modes
- [ ] Replace `pkg/daemon/daemon.go` DEBUG prints with slog
- [ ] Replace `pkg/opencode/service.go` prints
- [ ] Replace `pkg/account/oauth.go` prints
- [ ] Add `--log-level` flag to daemon command
- [ ] Document in README.md
- [ ] Verify daemon.log output is parseable
- [ ] Do NOT convert cmd/orch/*.go user output

---

## References

**Investigation:** `.kb/investigations/2026-01-03-inv-structured-logging-orch-go-808.md`

**Prior knowledge:**
- Action logging constraint: Tool action outcomes use action-log.jsonl (consistent pattern)
- Events logger: pkg/events/logger.go (existing JSONL pattern)

**External:**
- [slog proposal](https://go.dev/blog/slog) - Official Go blog on slog design
- [slog package docs](https://pkg.go.dev/log/slog) - stdlib documentation
