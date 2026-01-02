## Summary (D.E.K.N.)

**Delta:** Headless spawn was not calling StartBackgroundCleanup(), causing stdout buffer blocking; scanner buffer too small for large JSON events.

**Evidence:** Code review showed StartBackgroundCleanup() method existed but was never called in headless path; default bufio.Scanner uses 64KB max token size, OpenCode emits 100KB+ events.

**Knowledge:** Headless spawns MUST call StartBackgroundCleanup() to drain output; large JSON handling requires explicit buffer sizing.

**Next:** close - Implementation complete, tests pass.

---

# Investigation: Task Fix Headless Spawn Lifecycle

**Question:** Why do headless spawns sometimes hang or fail with scanner errors?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: StartBackgroundCleanup() never called in headless path

**Evidence:** The `headlessSpawnResult` struct has a `StartBackgroundCleanup()` method that drains stdout and waits for the process in a goroutine. This method was defined but never called after successful headless spawn.

**Source:** `cmd/orch/main.go` - runSpawnHeadless function (around line 1860)

**Significance:** Without calling this method, stdout is never drained, which can cause the subprocess to block when its output buffer fills up.

---

### Finding 2: Default bufio.Scanner buffer is too small

**Evidence:** OpenCode emits JSON events that can be very large (especially tool outputs containing file contents). The default `bufio.MaxScanTokenSize` is 64KB, but OpenCode events can exceed 100KB.

**Source:** `pkg/opencode/client.go` - ExtractSessionIDFromReader, ProcessOutput, ProcessOutputWithStreaming functions

**Significance:** When a JSON event exceeds 64KB, bufio.Scanner returns `bufio.ErrTooLong`, causing spawn failures.

---

### Finding 3: Pre-existing infrastructure was already solid

**Evidence:** Found that WaitForMessage(), SendPromptWithVerification(), and related infrastructure already existed with proper retry logic and timeout handling. The issue was specifically in the headless spawn lifecycle, not the message delivery.

**Source:** `pkg/opencode/client.go` lines 931-987, `pkg/opencode/types.go` (ErrMessageDeliveryTimeout)

**Significance:** Fix was surgical - just needed to call existing cleanup method and increase buffer size.

---

## Synthesis

**Key Insights:**

1. **Cleanup methods must be called** - Having infrastructure isn't enough; it must be wired into the execution path.

2. **Large event handling is critical** - OpenCode's JSON events can be arbitrarily large due to tool outputs containing file contents.

3. **Buffer sizing must be explicit** - Default bufio.Scanner settings are insufficient for this use case.

**Answer to Investigation Question:**

Headless spawns hang because stdout isn't drained (StartBackgroundCleanup not called) and fail with scanner errors because the default 64KB buffer is too small for OpenCode's large JSON events.

---

## Structured Uncertainty

**What's tested:**

- ✅ Large event handling works up to 100KB (verified: unit tests pass with 100KB payloads)
- ✅ StartBackgroundCleanup call compiles and builds (verified: make install succeeds)
- ✅ All existing tests still pass (verified: go test ./pkg/opencode/...)

**What's untested:**

- ⚠️ Real-world headless spawn with large tool outputs (not tested in production)
- ⚠️ Events larger than 1MB (would still fail, but unlikely in practice)

**What would change this:**

- Finding would be wrong if actual OpenCode events exceed 1MB (would need even larger buffer)
- Finding would be wrong if there's another stdout blocking issue beyond buffer draining

---

## Implementation (Complete)

**Changes made:**

1. Added `result.StartBackgroundCleanup()` call after successful headless spawn in `cmd/orch/main.go:1863`

2. Added `LargeScannerBufferSize = 1024 * 1024` constant in `pkg/opencode/client.go:21`

3. Applied large buffer to all three scanner usages:
   - `ExtractSessionIDFromReader()` line 118
   - `ProcessOutput()` line 145
   - `ProcessOutputWithStreaming()` line 180

4. Added tests for large event handling in `pkg/opencode/client_test.go`

---

## References

**Files Modified:**
- `cmd/orch/main.go` - Added StartBackgroundCleanup call
- `pkg/opencode/client.go` - Added large scanner buffer
- `pkg/opencode/client_test.go` - Added large event tests

**Commands Run:**
```bash
# Ran tests
go test ./pkg/opencode/... -v

# Committed changes
git commit -m "fix: headless spawn lifecycle cleanup and scanner buffer size"
```

---

## Investigation History

**2026-01-02:** Investigation started
- Initial question: Why do headless spawns hang or fail?
- Context: orch-go-cqlx issue

**2026-01-02:** Found root cause
- StartBackgroundCleanup not called, scanner buffer too small

**2026-01-02:** Investigation completed
- Status: Complete
- Key outcome: Fixed both issues, tests pass, committed as 5f240869
