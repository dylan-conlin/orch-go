## Summary (D.E.K.N.)

**Delta:** The daemon loop printed a misleading "No spawnable issues found" message when issues were found but couldn't be spawned due to capacity or errors.

**Evidence:** Code analysis showed `Once()` returns specific messages ("At capacity - no slots available", "Failed to spawn: ...") but daemon loop ignored them and printed generic message.

**Knowledge:** The `result.Message` from `Once()` contains the actual reason processing stopped; daemon loop should use it instead of hardcoded text.

**Next:** Fix applied - daemon loop now uses `result.Message` for accurate feedback.

**Confidence:** High (90%) - Root cause clear from code analysis; fix verified by code review and tests pass.

---

# Investigation: Daemon Finds Triage Ready Issues

**Question:** Why does daemon print "No spawnable issues found" after showing "DEBUG: Selected orch-go-mhec.1"?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Agent (og-debug-daemon-finds-triage-24dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Daemon loop prints generic message instead of actual reason

**Evidence:** 
- Line 251 of `cmd/orch/daemon.go` printed hardcoded "No spawnable issues found"
- This was printed when `result.Processed == false` regardless of `result.Message`
- `Once()` returns specific messages: "At capacity - no slots available", "Failed to spawn: ...", "No spawnable issues in queue"

**Source:** `cmd/orch/daemon.go:248-253`

**Significance:** This is the root cause - the DEBUG output from `NextIssue()` shows an issue was found, but then capacity or spawn failure causes `Processed: false`, and the daemon loop printed wrong message.

---

### Finding 2: OnceResult.Message contains accurate status

**Evidence:**
```go
// From pkg/daemon/daemon.go
if issue == nil {
    return &OnceResult{Processed: false, Message: "No spawnable issues in queue"}
}
if slot == nil {
    return &OnceResult{Processed: false, Message: "At capacity - no slots available"}
}
if spawnFunc fails {
    return &OnceResult{Processed: false, Message: "Failed to spawn: ..."}
}
```

**Source:** `pkg/daemon/daemon.go:378-417`

**Significance:** The information for accurate messaging already exists in `result.Message`, just wasn't being used.

---

### Finding 3: Current beads data shows no open issues with triage:ready

**Evidence:**
- `bd list --status open --json | jq` shows 188 open issues but 0 with triage:ready label
- Issues orch-go-mhec.1-.4 have triage:ready but status is `in_progress` (not `open`)
- Daemon uses `bd list --status open` so these issues aren't returned

**Source:** `bd list --status open --json`, `bd list --json | jq`

**Significance:** This explains why current testing shows "No spawnable issues in queue" - the test conditions have changed since the bug was reported.

---

## Synthesis

**Key Insights:**

1. **Message disconnect in verbose output** - When `NextIssue()` finds an issue (prints "Selected..."), but `Once()` can't spawn it (capacity, error), the daemon loop was printing wrong message.

2. **Result.Message already contains accurate info** - The `Once()` function already returns specific messages explaining why processing stopped. The fix is just to use this existing data.

3. **Test conditions changed since bug report** - The issues mentioned in bug report now have status `in_progress` instead of `open`, so they won't be returned by current queries.

**Answer to Investigation Question:**

The daemon printed "No spawnable issues found" after "Selected..." because the daemon loop used a hardcoded message instead of the actual `result.Message` from `Once()`. When an issue is found but can't be spawned (capacity full, spawn error), `Once()` returns `Processed: false` with a specific message, but the daemon loop ignored it. Fixed by using `result.Message` directly.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Clear root cause identified through code analysis. The fix is minimal (use existing data) and tests pass.

**What's certain:**

- ✅ Root cause: daemon loop prints hardcoded message instead of `result.Message`
- ✅ Fix works: changed line 251 to use `result.Message`
- ✅ Tests pass: `go test ./...` succeeds

**What's uncertain:**

- ⚠️ Cannot reproduce exact bug scenario (data has changed since report)
- ⚠️ May be additional edge cases not covered

**What would increase confidence to Very High:**

- Reproduce with actual capacity scenario
- Add integration test that verifies output messages

---

## Implementation (Complete)

**Fix applied:**

Changed `cmd/orch/daemon.go` line 251 from:
```go
fmt.Printf("[%s] No spawnable issues found\n", timestamp)
```

To:
```go
fmt.Printf("[%s] %s\n", timestamp, result.Message)
```

This ensures the daemon loop prints the actual reason from `Once()`:
- "No spawnable issues in queue" (when `NextIssue()` returns nil)
- "At capacity - no slots available" (when pool is full)
- "Failed to spawn: ..." (when spawn command fails)

---

## References

**Files Examined:**
- `cmd/orch/daemon.go` - Daemon loop with verbose output
- `pkg/daemon/daemon.go` - Once() function and result messages
- `pkg/daemon/daemon_test.go` - Existing tests for Once() messages

**Commands Run:**
```bash
# Check beads data
bd list --status open --json | jq
bd list --json | jq '.[] | select(.labels | any(. == "triage:ready"))'

# Build and test
go build ./cmd/orch/
go test ./pkg/daemon/...
go test ./cmd/orch/...
```

---

## Investigation History

**2025-12-24 11:30:** Investigation started
- Initial question: Why daemon shows "Selected..." then "No spawnable issues found"?
- Context: Beads issue orch-go-asxv

**2025-12-24 11:45:** Root cause identified
- Found daemon loop ignores `result.Message` and prints hardcoded text

**2025-12-24 12:00:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Fixed daemon loop to use `result.Message` for accurate feedback
