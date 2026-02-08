# Side-by-Side Code Quality: GPT-5.3-codex vs Claude Opus Extraction

**Date:** 2026-02-06
**Status:** Complete
**Beads:** orch-go-21346

## Context

Two extraction tasks ran in parallel on similar-complexity source files:

- **GPT-5.3-codex** (og-feat-extract-doctor-go-06feb-a099): `doctor.go` (2165 lines) -> 5+ modules
- **Claude Opus** (og-feat-extract-status-cmd-06feb-54b5): `status_cmd.go` (1508 lines) -> 4 modules

Both tasks had identical instructions: split by domain, keep files under 500 lines, preserve behavior and tests, follow `.kb/guides/code-extraction-patterns.md`.

## Summary

| Dimension                 | GPT-5.3-codex                 | Claude Opus                     | Winner |
| ------------------------- | ----------------------------- | ------------------------------- | ------ |
| **Compiles**              | No (duplicate test funcs)     | Yes                             | Claude |
| **File count (source)**   | 11 files                      | 4 files                         | Tie    |
| **File count (test)**     | 6 files                       | 5 files                         | Tie    |
| **Max file size**         | 484 lines (doctor_daemon.go)  | 686 lines (status_cmd.go)       | GPT    |
| **Under 500 line target** | 10/11 pass, 1 fail (484)      | 2/4 fail (686, 579)             | GPT    |
| **Test duplication**      | Critical: 17 funcs duplicated | None                            | Claude |
| **Testability patterns**  | Globals everywhere            | Function var injection          | Claude |
| **Dead code**             | Orphaned comment block        | 1 likely dead function          | Tie    |
| **Naming conventions**    | Consistent Go                 | Consistent Go                   | Tie    |
| **Error handling**        | Inconsistent across files     | Consistent graceful degradation | Claude |
| **Code duplication**      | Health check logic 3x         | Minor display setup             | Claude |

**Overall winner: Claude Opus** - produces shippable code. GPT-5.3 extraction does not compile.

---

## Detailed Findings

### 1. Correctness: Does It Compile?

**GPT-5.3: NO.** This is the most critical finding.

GPT created per-module test files (`doctor_correctness_test.go`, `doctor_daemon_test.go`, `doctor_config_test.go`, `doctor_install_test.go`, `doctor_sessions_test.go`) AND left every duplicated test function in the original `doctor_test.go`. Since all files are `package main`, Go rejects duplicate function names at compile time.

17 test functions are duplicated across files:

- `TestCheckBeadsIntegrity` (doctor_test.go + doctor_correctness_test.go)
- `TestParseElapsedTime` (doctor_test.go + doctor_daemon_test.go)
- `TestParsePlistValues` (doctor_test.go + doctor_config_test.go)
- ...and 14 more

**Evidence:** `git status` shows all per-module test files as untracked (`??`), and `doctor_test.go` retains all original tests.

**Claude Opus: YES.** Committed as `d1e62db3` with "All 66 tests pass" in the commit message. Test files are properly split with no duplication. Tests are co-located with source domains.

**Significance:** A non-compiling extraction is a failed extraction regardless of other qualities. This alone makes GPT's output require significant rework.

### 2. File Structure and Granularity

**GPT-5.3** split into 11 source files:
| File | Lines | Domain |
|------|-------|--------|
| doctor.go | 332 | Command def, types, orchestration |
| doctor_services.go | 418 | Service health checks |
| doctor_beads.go | 59 | Beads DB integrity |
| doctor_registry.go | 73 | Registry reconciliation |
| doctor_docker.go | 72 | Docker backend check |
| doctor_stalled.go | 100 | Stalled session detection |
| doctor_binary.go | 174 | Binary staleness checks |
| doctor_config.go | 296 | Config drift detection |
| doctor_daemon.go | 484 | Daemon mode + watch mode |
| doctor_install.go | 131 | Plist install/uninstall |
| doctor_sessions.go | 298 | Session cross-reference |

**Claude Opus** split into 4 source files:
| File | Lines | Domain |
|------|-------|--------|
| status_cmd.go | 686 | Command def, types, orchestration |
| status_agents.go | 392 | Agent discovery, enrichment |
| status_health.go | 128 | Infrastructure health |
| status_display.go | 579 | Terminal output formatting |

**Analysis:** GPT produced finer-grained files. The smallest (doctor_beads.go at 59 lines) is arguably too small - a single function could stay in the parent. Claude's files are coarser, with status_cmd.go (686 lines) and status_display.go (579 lines) exceeding the 500-line target. However, Claude's splits follow cleaner domain boundaries (data/logic/display separation) while GPT's are more feature-based.

### 3. Function Organization and Cohesion

**GPT-5.3:** Each file has high cohesion (one domain per file), but introduces hidden coupling through package globals (`doctorVerbose`, `doctorFix`, `serverURL`, `sourceDir`, `gitHash`). Functions read these globals directly, making them impossible to test in isolation without setting global state.

**Claude Opus:** Better cohesion patterns. Key functions like `computeIsPhantom` and `computeSwarmStatus` are pure functions (no side effects, no global reads). The `tcpDialTimeout` function variable in `status_health.go` enables test mocking without global mutation. `printSwarmStatusWithWidth` takes explicit width parameter for testability rather than reading terminal width directly.

However, Claude has some misplaced functions:

- `printInfrastructureHealth` in health file (should be in display)
- `getAccountUsage` in display file (should be in agents or cmd)
- `getAgentStatus` tested in agents_test but defined in display

### 4. Idiomatic Go Patterns

**GPT-5.3:**

- Correct use of `struct{}` anonymous types but overuses them (repeated 3x for session registry)
- `[]interface{}` instead of `[]any` (pre-1.18 style, valid but dated)
- Good use of `make([]ServiceStatus, 0)` for JSON serialization (outputs `[]` not `null`)
- `os.Exit(1)` in cobra `RunE` functions instead of returning errors

**Claude Opus:**

- Proper `interface{ Close() error }` for mock-friendly abstractions
- `for i := range agents` with `agent := &agents[i]` for in-place mutation (textbook Go)
- Clean `fmt.Errorf("...: %w", err)` error wrapping throughout
- Named constants for all magic numbers (`compactOrchestratorSessionsMaxAge`, etc.)

### 5. Test Quality

**GPT-5.3 (if duplicates were fixed):**

- 23 unique test functions across 6 files
- Heavy on struct field validation tests ("does this field exist?") - low value since the compiler catches this
- Integration-style tests that call real functions (e.g., `TestCheckBeadsIntegrity` runs actual PRAGMA query)
- Good table-driven tests for `ParseElapsedTime` and `IsSessionInRegistry`
- No benchmarks

**Claude Opus:**

- 19 test functions across 5 files
- Well-targeted unit tests using mocks for infrastructure (TCP dial, HTTP servers)
- 8 benchmark functions in `status_bench_test.go` covering session count, untracked ratio, and API latency dimensions
- Custom `b.ReportMetric` for API calls per operation
- Mock server that self-validates (`TestMockServerBasicFunctionality`)
- Coverage gaps: `extractDateFromWorkspaceName`, `formatModelForDisplay`, `readAgentManifest` untested

**Analysis:** Claude's test suite is more sophisticated. The benchmark suite is a differentiator - it enables performance regression detection. GPT's field-existence tests are noise. Claude's mock patterns (function variable injection, httptest server) are production-grade.

### 6. Error Handling

**GPT-5.3:** Inconsistent patterns across files:

- `os.UserHomeDir()` errors silently ignored in multiple places
- `os.MkdirAll()` return sometimes checked, sometimes not
- Notification errors checked in watch mode but not daemon mode
- `DoctorDaemonLogger.Log()` silently swallows file errors

**Claude Opus:** Consistent graceful degradation:

- Non-critical errors (tmux, registry lookups) silently ignored - appropriate for status display
- Critical errors properly returned with `%w` wrapping
- `readDaemonStatus` returns nil on any error (correct - nil means "not running")
- `readAgentManifest` returns nil on any error (correct - nil means "no manifest")

### 7. Code Duplication

**GPT-5.3:** Three separate implementations of the health check sequence:

1. `runDoctor()` in doctor.go
2. `runHealthCheckWithNotifications()` in doctor_daemon.go
3. `runDaemonHealthCycle()` in doctor_daemon.go

Estimated ~100 lines of duplicated check-append-track logic. A shared `buildHealthReport()` would eliminate this.

**Claude Opus:** Minor duplication in `printAgentsWideFormat`/`printAgentsNarrowFormat`/`printAgentsCardFormat` (defaulting empty fields to "-"). Manageable at current scale.

### 8. Line Count Compliance

The task specified: "Each extracted file should be under 500 lines."

**GPT-5.3:** 10 of 11 source files under 500 lines. `doctor_daemon.go` at 484 is close but compliant. Strongest compliance.

**Claude Opus:** 2 of 4 source files exceed 500 lines:

- `status_cmd.go`: 686 lines (36% over target)
- `status_display.go`: 579 lines (16% over target)

The `runStatus` function alone is 520 lines within status_cmd.go.

---

## Recommendation

**For production use: Claude Opus produces cleaner, more reliable extractions.**

The GPT-5.3 output requires manual intervention before it can even compile. The duplicate test functions are a showstopper that would require reviewing all 6 test files and removing ~477 lines of duplicated tests from `doctor_test.go`. Beyond compilation, the code itself has inconsistent error handling, unnecessary code duplication in health check orchestration, and global-dependent functions that resist testing.

Claude's output ships as-is. It was committed, tested (66 tests passing), and the code demonstrates stronger engineering patterns: dependency injection via function variables, pure computation functions, benchmark infrastructure, and consistent error handling. Its weakness is file size compliance (2 files over 500 lines).

**Key takeaway:** GPT-5.3 produces more granular file splits but fails on the fundamental requirement of producing compiling code. Claude produces slightly-too-large files but delivers working, well-tested code.

### Scoring (1-5 scale)

| Dimension                          | GPT-5.3 | Claude Opus |
| ---------------------------------- | ------- | ----------- |
| Correctness (compiles, tests pass) | 1       | 5           |
| File structure                     | 4       | 3           |
| Function cohesion                  | 3       | 4           |
| Idiomatic Go                       | 3       | 4           |
| Test quality                       | 2       | 4           |
| Error handling                     | 2       | 4           |
| Documentation                      | 3       | 3           |
| Line count compliance              | 5       | 2           |
| **Average**                        | **2.9** | **3.6**     |

**Next:** Consider whether GPT-5.3's granularity approach is worth adopting as the target structure, then have Claude redo the extraction with tighter file size targets.
