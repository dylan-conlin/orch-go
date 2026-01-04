## Summary (D.E.K.N.)

**Delta:** 815 fmt.Print calls break down into ~700 CLI user output (keep fmt) and ~115 daemon/service logging (convert to slog).

**Evidence:** Analyzed all Go files - cmd/orch/*.go is user-facing output with emojis and formatting; pkg/daemon/*.go has DEBUG prints that need structure.

**Knowledge:** CLI output and operational logging are fundamentally different concerns - structured logging only applies to background/daemon processes.

**Next:** Created decision record with implementation plan. See `.kb/decisions/2026-01-03-structured-logging-orch-go.md`.

---

# Investigation: Structured Logging for orch-go

**Question:** What logging library should orch-go adopt, and how should it handle the 815 existing fmt.Printf calls?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** None - decision record created
**Status:** Complete

---

## Findings

### Finding 1: Output Call Distribution

**Evidence:** 
- Total `fmt.Print*` calls: 815
- Top files by count:
  - cmd/orch/main.go: 247 (CLI user output)
  - cmd/orch/review.go: 66 (CLI user output)
  - cmd/orch/daemon.go: 56 (mix of CLI and daemon)
  - pkg/daemon/daemon.go: 15 (daemon debug logging)

**Source:** `rg 'fmt\.Print' --type go -c | sort -rn`

**Significance:** Not all 815 calls need structured logging. CLI output (emojis, interactive prompts, status displays) should stay as fmt.Printf. Only daemon/background service logging needs structure.

---

### Finding 2: Existing Logging Infrastructure

**Evidence:**
- `pkg/events/logger.go` - JSONL logger for agent lifecycle events (spawn, complete, error)
- Uses pattern: `~/.orch/events.jsonl`
- Already has: LogSpawn, LogCompleted, LogError, LogStatusChange, LogAutoCompleted
- 86 existing `log.*` calls scattered across codebase

**Source:** pkg/events/logger.go (154 lines)

**Significance:** The events logger is for "what happened" (lifecycle events). We need a separate concern for "why/how" (debug/operational logging). These complement, don't replace each other.

---

### Finding 3: Daemon Debug Pattern

**Evidence:**
```go
// Current pattern in pkg/daemon/daemon.go
if d.Config.Verbose {
    fmt.Printf("  DEBUG: Skipping %s (type %s not spawnable)\n", issue.ID, issue.IssueType)
}
```
- Guarded by Verbose flag
- Uses "DEBUG:" prefix convention
- No structured fields
- Output goes to stdout (lost when running via launchd)

**Source:** pkg/daemon/daemon.go:294-339

**Significance:** Daemon debug output needs:
1. JSON format for machine parsing
2. File destination (not stdout)
3. Log levels instead of "DEBUG:" prefix
4. Structured fields for querying

---

### Finding 4: Go Version Enables stdlib slog

**Evidence:**
- go.mod: `go 1.24.0`
- log/slog added in Go 1.21
- No external logging dependencies currently

**Source:** go.mod, `go version`

**Significance:** Can use stdlib slog without adding dependencies. Aligns with Go ecosystem direction.

---

## Synthesis

**Key Insights:**

1. **CLI output is not logging** - The 700+ calls in cmd/orch/*.go are user-facing output with emojis, formatting, and interactive elements. Converting these to structured logging would make output worse, not better.

2. **Daemon needs structured logging** - The ~115 calls in pkg/* are operational/debug logging that would benefit from JSON format, log levels, and structured fields for analysis.

3. **Zero-dependency solution available** - stdlib slog provides everything needed without adding external dependencies.

**Answer to Investigation Question:**

Adopt **stdlib log/slog** with a hybrid approach:
- **Daemon/services:** Use slog with JSON handler to `~/.orch/daemon.log`
- **CLI commands:** Keep fmt.Printf for user-facing output
- **Error returns:** Keep `fmt.Errorf()` pattern as-is

This right-sizes the solution: structured logging where it helps (daemon debugging), simple output where it's appropriate (CLI).

---

## Structured Uncertainty

**What's tested:**

- ✅ 815 fmt.Print calls counted (ran: `rg 'fmt\.Print' --type go -c`)
- ✅ pkg/daemon has verbose-guarded DEBUG prints (read: daemon.go)
- ✅ Go 1.24 includes slog (verified: go.mod and Go docs)
- ✅ events.jsonl pattern works for lifecycle events (read: pkg/events/logger.go)

**What's untested:**

- ⚠️ Log rotation for daemon.log (assumed logrotate or manual)
- ⚠️ Performance of JSON serialization (assumed negligible for CLI tool)
- ⚠️ How daemon.log interacts with existing events.jsonl (assumed complementary)

**What would change this:**

- If sub-millisecond logging needed → zerolog
- If CLI output needs internationalization → different approach
- If log aggregation/shipping needed → may need different handler

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: stdlib slog with Hybrid Handling

**Why this approach:**
- Zero dependencies (matches orch-go's minimal dependency philosophy)
- Future-proof (slog is Go's standard direction)
- Right-sized (not over-engineering a CLI tool)
- Clear separation (logging vs output are different concerns)

**Trade-offs accepted:**
- CLI output stays as fmt.Printf (intentional, not technical debt)
- No fancy log aggregation (not needed for single-user CLI)

**Implementation sequence:**
1. Create `pkg/log/log.go` with daemon/CLI mode initialization
2. Replace pkg/daemon DEBUG prints with slog calls
3. Replace other pkg/*.go service logging
4. Leave cmd/orch/*.go untouched

### Alternative Approaches Considered

**Option B: zerolog**
- **Pros:** Zero-allocation, very fast, fluent API
- **Cons:** External dependency, overkill for CLI tool
- **When to use instead:** High-throughput server applications

**Option C: zap**
- **Pros:** Battle-tested at Uber scale, flexible
- **Cons:** Heavy dependency, complex config
- **When to use instead:** Large distributed systems

**Rationale for recommendation:** stdlib slog provides exactly what we need without the complexity or dependencies of external libraries. The hybrid approach respects that CLI output and operational logging are fundamentally different.

---

## References

**Files Examined:**
- go.mod - Dependency and Go version check
- pkg/events/logger.go - Existing JSONL logging pattern
- pkg/daemon/daemon.go - Current DEBUG printing approach
- cmd/orch/main.go - CLI output patterns (247 prints)

**Commands Run:**
```bash
# Count fmt.Print calls
rg 'fmt\.Print' --type go -c | awk -F: '{sum += $2} END {print sum}'
# Result: 815

# Find daemon DEBUG pattern
rg 'fmt\.Printf.*DEBUG' --type go -l
# Result: pkg/daemon/daemon.go

# Check Go version
go version
# Result: go1.23.5 darwin/arm64 (go.mod specifies 1.24.0)
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-03-structured-logging-orch-go.md` - Implementation decision
- **Prior constraint:** Action logging uses action-log.jsonl pattern (from kb context)

---

## Investigation History

**2026-01-03 14:xx:** Investigation started
- Initial question: How to handle 815 fmt.Printf calls for structured logging
- Context: Task spawned from orchestrator

**2026-01-03 14:xx:** Context gathering complete
- Analyzed codebase structure
- Identified CLI vs daemon output split
- Reviewed existing events logger

**2026-01-03 14:xx:** Investigation completed
- Status: Complete
- Key outcome: Decision record created with hybrid slog approach
