<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Three separate issues identified: (1) SSE idle != Phase: Complete (by design), (2) FallbackClose doesn't respect beads.DefaultDir for cross-project ops, (3) RPC client may send wrong Cwd for cross-project closes.

**Evidence:** Code review of monitor.go:172-181 (idle detection), client.go:738-747 (FallbackClose has no Dir), check.go:570-583 (CloseIssue creates client without WithCwd).

**Knowledge:** The "silent failure" may be caused by wrong directory context in CLI fallback or RPC Cwd mismatch, but specific reproduction pending. Session idle is distinct from work complete.

**Next:** Implement fix for FallbackClose to use beads.DefaultDir, add WithCwd to CloseIssue RPC client, and add error logging to surface silent failures.

---

# Investigation: Agents Going Idle Without Phase

**Question:** Why do agents go idle without reporting Phase: Complete, and why does orch complete --force sometimes fail to persist issue closure?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Agent (og-debug-agents-going-idle-03jan)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: SSE Idle Detection != Phase: Complete (By Design)

**Evidence:** The OpenCode monitor (pkg/opencode/monitor.go:172-181) detects "completion" when a session transitions from busy to idle. This is explicitly documented in the spawn context constraint: "Session idle ≠ agent complete - Agents legitimately go idle during normal operation (loading, thinking, tool execution)".

**Source:** 
- `pkg/opencode/monitor.go:172-181` - idle transition detection
- SPAWN_CONTEXT.md prior knowledge constraints

**Significance:** The issue description conflated two separate concepts. Agents going "idle" in SSE is NOT a bug - it's expected behavior. The actual bug is agents not calling `bd comment "Phase: Complete"` before ending their session.

---

### Finding 2: FallbackClose Doesn't Respect beads.DefaultDir

**Evidence:** The `FallbackClose` function (pkg/beads/client.go:738-747) uses `exec.Command("bd", args...)` without setting `cmd.Dir`. This means CLI fallback runs in the current working directory, not the beads project directory.

```go
func FallbackClose(id, reason string) error {
    args := []string{"close", id}
    if reason != "" {
        args = append(args, "--reason", reason)
    }
    cmd := exec.Command("bd", args...)
    return cmd.Run()  // No Dir set!
}
```

Test confirmed: `bd close` from wrong directory fails with "no beads database found" (exit code 1).

**Source:**
- `pkg/beads/client.go:738-747` - FallbackClose implementation
- Manual test: `cd /tmp && bd close orch-go-rzch 2>&1` returns error

**Significance:** For cross-project completions where RPC fails and falls back to CLI, the close operation would fail silently because the error isn't captured with stderr.

---

### Finding 3: CloseIssue Doesn't Pass WithCwd to RPC Client

**Evidence:** The `CloseIssue` function (pkg/verify/check.go:570-583) creates an RPC client without the `WithCwd` option:

```go
func CloseIssue(beadsID, reason string) error {
    socketPath, err := beads.FindSocketPath("")
    if err == nil {
        client := beads.NewClient(socketPath, beads.WithAutoReconnect(3))
        // Missing: beads.WithCwd(beads.DefaultDir)
        ...
    }
    ...
}
```

The RPC client uses `os.Getwd()` for `Cwd` if not set (client.go:253-255). For cross-project operations, this sends the orchestrator's directory, not the beads project directory.

**Source:**
- `pkg/verify/check.go:570-583` - CloseIssue
- `pkg/beads/client.go:64-73` - NewClient
- `pkg/beads/client.go:246-262` - executeLocked (Cwd setting)

**Significance:** Depending on how the beads daemon validates `Cwd`, this could cause the close to apply to wrong database or be rejected.

---

## Synthesis

**Key Insights:**

1. **Idle vs Complete Distinction** - The SSE-based "completion" detection is about session state, not work completion. This is correct and by design. Agents must explicitly report Phase: Complete via beads comment.

2. **Cross-Project Context Loss** - Both the CLI fallback and RPC client have issues with cross-project directory context. The CLI fallback doesn't set `cmd.Dir`, and the RPC client doesn't use `WithCwd`.

3. **Error Visibility Gap** - FallbackClose uses `cmd.Run()` which only returns exit code, not stderr. Silent failures are possible when CLI fallback triggers.

**Answer to Investigation Question:**

Why do agents go idle without Phase: Complete?
- This is **expected behavior**. Agents go idle for many reasons (loading, thinking, context exhaustion). Only agents that explicitly run `bd comment "Phase: Complete"` are considered complete.

Why does orch complete --force fail to persist?
- **Hypothesized cause:** Cross-project directory context loss in `CloseIssue`. When RPC fails and falls back to CLI, or when RPC sends wrong `Cwd`, the close may fail silently.
- The issue was later closed (verified: 57dn shows status=closed), suggesting either: (a) retry worked, (b) manual intervention, or (c) timing issue with caching.

---

## Structured Uncertainty

**What's tested:**

- ✅ SSE idle detection triggers on busy→idle transition (verified: code review of monitor.go)
- ✅ FallbackClose has no Dir set (verified: code review of client.go:738-747)
- ✅ bd close from wrong directory fails with exit code 1 (verified: manual test in /tmp)

**What's untested:**

- ⚠️ Actual RPC Cwd behavior when beads daemon receives mismatched directory (not tested against live daemon)
- ⚠️ Exact conditions that trigger CLI fallback vs RPC success (not reproduced)
- ⚠️ Whether issue 57dn failure was due to this bug or other cause (can't reproduce - issue now closed)

**What would change this:**

- Finding would be wrong if beads daemon ignores Cwd for close operations
- Finding would be wrong if CLI fallback never triggers in practice (RPC always succeeds)
- Finding would be wrong if the original issue was user error, not code bug

---

## Implementation Recommendations

### Recommended Approach: Fix Directory Context in Both Paths

**Fix 1: FallbackClose should respect beads.DefaultDir**

```go
func FallbackClose(id, reason string) error {
    args := []string{"close", id}
    if reason != "" {
        args = append(args, "--reason", reason)
    }
    cmd := exec.Command("bd", args...)
    if DefaultDir != "" {
        cmd.Dir = DefaultDir
    }
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("bd close failed: %w: %s", err, string(output))
    }
    return nil
}
```

**Fix 2: CloseIssue should pass WithCwd to RPC client**

```go
func CloseIssue(beadsID, reason string) error {
    socketPath, err := beads.FindSocketPath("")
    if err == nil {
        opts := []beads.Option{beads.WithAutoReconnect(3)}
        if beads.DefaultDir != "" {
            opts = append(opts, beads.WithCwd(beads.DefaultDir))
        }
        client := beads.NewClient(socketPath, opts...)
        if err := client.CloseIssue(beadsID, reason); err == nil {
            return nil
        }
    }
    return beads.FallbackClose(beadsID, reason)
}
```

**Why this approach:**
- Fixes both RPC and CLI paths
- Uses existing `beads.DefaultDir` mechanism
- Adds error visibility with `CombinedOutput()`

**Trade-offs accepted:**
- Adds dependency on global `DefaultDir` variable
- Doesn't address root cause of why agents don't report Phase: Complete

**Implementation sequence:**
1. Fix FallbackClose to use DefaultDir and capture stderr
2. Fix CloseIssue to pass WithCwd to RPC client
3. Add integration test for cross-project close

### Alternative: Agent-Side Fix

**Option B: Improve agent phase reporting**
- **Pros:** Prevents agents from going idle without Phase: Complete
- **Cons:** Doesn't fix the orch complete silent failure bug
- **When to use:** In addition to, not instead of, the recommended fix

---

## References

**Files Examined:**
- `pkg/opencode/monitor.go:142-196` - SSE event handling and completion detection
- `pkg/beads/client.go:455-463` - RPC CloseIssue
- `pkg/beads/client.go:738-747` - FallbackClose
- `pkg/verify/check.go:570-583` - CloseIssue
- `cmd/orch/main.go:752-1022` - runComplete flow

**Commands Run:**
```bash
# Test bd close from wrong directory
cd /tmp && bd close orch-go-rzch --reason "test" 2>&1
# Result: Error: no beads database found, exit code 1

# Check issue status
bd show 57dn --json
# Result: status=closed, closed_at=2026-01-03T20:48:09
```

**Related Artifacts:**
- **Constraint:** "Session idle ≠ agent complete" - spawn context prior knowledge
- **Decision:** "SSE busy->idle cannot detect true agent completion" - prior decision

---

## Investigation History

**2026-01-03 20:49:** Investigation started
- Initial question: Why do agents go idle without Phase: Complete, and why does orch complete --force fail silently?
- Context: Evidence from 80tq (worked) vs 57dn (failed) completions

**2026-01-03 21:30:** Key findings identified
- SSE idle detection is by design, not a bug
- FallbackClose doesn't respect beads.DefaultDir
- CloseIssue doesn't pass WithCwd to RPC client

**2026-01-03 21:45:** Investigation completed
- Status: Complete
- Key outcome: Three separate issues identified with recommended fixes
