**TLDR:** Ported wait command from Python orch-cli to Go. Implementation complete with TDD approach - timeout parsing, phase polling via beads, and proper exit codes (0=success, 1=timeout, 2=error). High confidence (95%) - all tests pass and command works as expected.

---

# Investigation: Add Wait Command to orch-go

**Question:** How to port the Python orch-cli wait command to Go?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker agent (spawned from orch-go-j66)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Python wait command pattern

**Evidence:** Python wait command in `~/Documents/personal/orch-cli/src/orch/monitoring_commands.py` lines 1276-1411 shows:
- Uses Click CLI framework with `@click.option` decorators
- Parses timeout strings like '30s', '5m', '1h', '1h30m' 
- Polls beads phase status at configurable interval
- Returns exit codes: 0 (success), 1 (timeout), 2 (error)
- Uses `_parse_timeout()` and `_format_duration()` helper functions
- Integrates with registry to find agent, but phase comes from beads

**Source:** `~/Documents/personal/orch-cli/src/orch/monitoring_commands.py:1276-1531`

**Significance:** Clear reference implementation to port. Key insight: phase status comes from beads comments, not registry. Registry only needed for tmux window operations.

---

### Finding 2: Go codebase patterns

**Evidence:** Existing commands in `cmd/orch/main.go` follow patterns:
- Use cobra.Command with RunE returning error
- Define flags via cmd.Flags().StringVar/BoolVar
- Use verify.GetPhaseStatus for beads phase checking
- Use events.Logger for event logging
- Exit directly with os.Exit for non-zero codes

**Source:** `cmd/orch/main.go`, `cmd/orch/daemon.go`, `pkg/verify/check.go`

**Significance:** Established patterns make porting straightforward. verify package already has GetPhaseStatus which does exactly what wait needs.

---

### Finding 3: Test patterns

**Evidence:** Existing tests in `cmd/orch/main_test.go` and `pkg/verify/check_test.go` use:
- Table-driven tests with named test cases
- Error checking via `(err != nil) != tt.wantErr` pattern
- Direct value comparison for expected results

**Source:** `cmd/orch/main_test.go`, `pkg/verify/check_test.go`

**Significance:** TDD approach using these patterns ensures consistency and test coverage.

---

## Synthesis

**Key Insights:**

1. **Phase checking via beads** - The verify package already has GetPhaseStatus which queries beads comments. No need to check registry or tmux - beads is the source of truth for agent phase.

2. **Timeout parsing is self-contained** - The timeout parsing logic (30s, 5m, 1h30m) is a pure function with no external dependencies, ideal for TDD.

3. **Exit codes align with shell conventions** - 0 for success, 1 for timeout, 2 for errors matches Unix convention and Python implementation.

**Answer to Investigation Question:**

Port completed successfully. The wait command is implemented in `cmd/orch/wait.go` with:
- parseTimeout() for duration parsing (30s, 5m, 1h, 1h30m formats)
- formatDuration() for human-readable elapsed time
- Polling loop using verify.GetPhaseStatus
- Event logging on success/timeout
- Proper exit codes

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All tests pass, the command builds and runs correctly, and the implementation follows established patterns.

**What's certain:**

- ✅ Timeout parsing works for all documented formats (tested)
- ✅ Duration formatting produces expected output (tested)
- ✅ Command is registered and help text displays correctly (verified)
- ✅ Exit code 2 returned for nonexistent issues (verified)

**What's uncertain:**

- ⚠️ Long-running polling behavior not tested in integration (would require real agent)
- ⚠️ Timeout of exactly 0 seconds behavior

**What would increase confidence to 100%:**

- Integration test with real beads issue and phase transitions
- Test coverage for edge case of agent disappearing mid-wait

---

## Implementation Recommendations

**Purpose:** Document what was implemented.

### Implemented Approach ⭐

**Separate wait.go file** - Wait command isolated in its own file for maintainability.

**Why this approach:**
- Keeps main.go from growing too large
- Follows pattern of daemon.go for multi-function commands
- Makes testing easier with focused test file

**Trade-offs accepted:**
- Slightly more files to navigate
- Init functions spread across files

**Implementation completed:**
1. Created `cmd/orch/wait.go` with parseTimeout, formatDuration, runWait
2. Created `cmd/orch/wait_test.go` with comprehensive tests
3. Added waitCmd to root command in main.go

---

### Implementation Details

**What was implemented:**
- `wait` command with `--phase`, `--timeout`, `--interval`, `--quiet` flags
- Timeout parsing supporting s, m, h units and combinations
- Phase polling using verify.GetPhaseStatus
- Event logging for wait completion/timeout

**Things to watch out for:**
- ⚠️ os.Exit() bypasses defer statements - logging happens before exit
- ⚠️ Phase matching is case-insensitive partial match (e.g., "Complete" matches "complete")

**Success criteria:**
- ✅ `orch-go wait --help` displays correct help text
- ✅ All unit tests pass
- ✅ Exit code 2 for invalid issues
- ✅ Command registered in CLI

---

## References

**Files Created:**
- `cmd/orch/wait.go` - Main implementation
- `cmd/orch/wait_test.go` - Unit tests

**Files Modified:**
- `cmd/orch/main.go` - Added waitCmd to root command

**Commands Run:**
```bash
# Run tests
go test ./cmd/orch/... -run "TestParseTimeout|TestFormatDuration" -v

# Build and test help
go build -o orch-test ./cmd/orch/ && ./orch-test wait --help

# Test error handling
./orch-test wait nonexistent-issue --timeout 1s
```

**Related Artifacts:**
- **Python Source:** `~/Documents/personal/orch-cli/src/orch/monitoring_commands.py`
- **Beads Issue:** orch-go-j66

---

## Investigation History

**2025-12-20 22:00:** Investigation started
- Initial question: How to port wait command from Python orch-cli?
- Context: Part of agent management commands (high priority)

**2025-12-20 22:02:** Found Python implementation
- Located wait command in monitoring_commands.py
- Identified key patterns: timeout parsing, phase polling, exit codes

**2025-12-20 22:05:** TDD cycle started
- Wrote failing tests for parseTimeout and formatDuration
- Implemented functions to make tests pass

**2025-12-20 22:08:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Wait command fully implemented with tests
