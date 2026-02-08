**TLDR:** Question: Can we create a reusable Go package for OpenCode session management? Answer: Yes - created pkg/opencode with Client (spawn/ask commands), types (Event, SSEEvent, Result), and SSE client with 93.2% test coverage. High confidence (90%) - all tests pass, refactored from working POC.

---

# Investigation: OpenCode Client Package Session Management

**Question:** Can we refactor the POC into a reusable pkg/opencode package with session CRUD, message sending, and event parsing?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: POC code refactors cleanly into package structure

**Evidence:** Successfully extracted types and functions from main.go into:
- `pkg/opencode/types.go` - Event, SessionInfo, StepInfo, SSEEvent, SessionStatus, Result types
- `pkg/opencode/client.go` - Client struct with BuildSpawnCommand, BuildAskCommand, ProcessOutput, ExtractSessionID
- `pkg/opencode/sse.go` - SSEClient with Connect, ReadSSEStream, ParseSSEEvent, ParseSessionStatus, DetectCompletion

**Source:** pkg/opencode/*.go files

**Significance:** Clean separation of concerns enables reuse. Types can be imported independently, client and SSE functionality are separate.

---

### Finding 2: Existing tests validate the refactoring

**Evidence:** 
- 93.2% test coverage on pkg/opencode
- Tests cover: event parsing, session ID extraction, command building, SSE stream reading, completion detection
- All 13 tests in main_test.go still pass (no regressions)

**Source:** 
- `go test ./... -v -cover` output
- pkg/opencode/client_test.go (182 lines)
- pkg/opencode/sse_test.go (374 lines)

**Significance:** High confidence that the package works correctly - comprehensive tests validate parsing, SSE handling, and command building.

---

### Finding 3: ErrNoSessionID provides clear error handling

**Evidence:** Created `ErrNoSessionID` sentinel error in types.go for consistent error handling when session ID is not found in output.

**Source:** pkg/opencode/types.go:10

**Significance:** Enables callers to check for specific error types with `errors.Is(err, opencode.ErrNoSessionID)`.

---

## Synthesis

**Key Insights:**

1. **Package structure follows Go idioms** - Types in types.go, client logic in client.go, SSE handling in sse.go. No circular dependencies.

2. **High test coverage validates implementation** - 93.2% coverage on pkg/opencode with comprehensive table-driven tests for all parsing functions.

3. **Clean separation enables evolution** - Client can be extended with Spawn/Ask methods that handle command execution, SSE can be used independently for monitoring.

**Answer to Investigation Question:**

Yes, the POC refactors cleanly into pkg/opencode. The package provides:
- Session management via Client (spawn commands, ask commands)
- Event types for JSON parsing (Event, Result, SSEEvent)
- SSE utilities for monitoring completion (SSEClient, DetectCompletion)

All functionality is validated by comprehensive tests with 93.2% coverage.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

High test coverage (93.2%) validates all parsing and command building. The code is extracted from a working POC, not written from scratch. Only missing integration testing against live OpenCode.

**What's certain:**

- ✅ JSON parsing works for all documented event types
- ✅ SSE parsing correctly handles event streams
- ✅ Command building creates correct opencode CLI arguments
- ✅ ErrNoSessionID error handling works correctly
- ✅ Package compiles and all tests pass

**What's uncertain:**

- ⚠️ Real OpenCode output may have additional undocumented event types
- ⚠️ Long-running SSE connections not stress-tested
- ⚠️ Actual session execution not tested (requires live OpenCode)

**What would increase confidence to Very High (95%+):**

- Integration test against live OpenCode session
- Test spawn/ask with real prompts
- Verify SSE monitoring catches real completion events

---

## Implementation Recommendations

**Purpose:** Package is complete and ready for use.

### Recommended Next Steps

1. **Use package in cmd/orch** - Import pkg/opencode in CLI commands
2. **Add Spawn/Ask methods** - Extend Client with methods that execute commands and return Results
3. **Integration test** - Run against live OpenCode when available

---

## References

**Files Examined:**
- main.go - Original POC implementation
- main_test.go - Original tests

**Commands Run:**
```bash
# Verify build
go build ./...

# Run all tests with coverage
go test ./... -v -cover
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-19-simple-opencode-poc-spawn-session-via.md` - Original POC investigation

---

## Investigation History

**2025-12-19:** Investigation started
- Initial question: Can we refactor POC into pkg/opencode?
- Context: Need reusable package for orchestration tools

**2025-12-19:** Package structure created
- types.go, client.go, sse.go with clean separation
- Resolved duplicate type definitions

**2025-12-19:** Tests validated
- 93.2% coverage on pkg/opencode
- All 13 main_test.go tests still pass (no regressions)

**2025-12-19:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: pkg/opencode package ready with Client, types, and SSE handling
