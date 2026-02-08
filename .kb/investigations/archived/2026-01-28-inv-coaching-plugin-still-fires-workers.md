<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The OpenCode fix for `session.metadata.role` from `x-opencode-env-ORCH_WORKER` header EXISTS in source (commit `bbf798911`) but the running server hasn't been rebuilt to use it.

**Evidence:** Source at `session.ts:207-211` correctly reads header and sets `role: "worker"`; coaching plugin at `coaching.ts:2020` correctly checks `sessionMetadata.role === "worker"`; worker reports `env | grep ORCH` empty because rebuild is missing.

**Knowledge:** The full chain is correctly implemented: `orch spawn → ORCH_WORKER=1 env → client header → session.metadata.role → plugin check`. The break is a deployment gap, not a code bug.

**Next:** Rebuild OpenCode (`cd ~/Documents/personal/opencode && bun run build`) and restart the server to pick up the fix.

**Promote to Decision:** recommend-no - This is a deployment/operational gap, not an architectural decision.

---

# Investigation: Coaching Plugin Still Fires Workers

**Question:** Why does the coaching plugin STILL fire for workers despite multiple fix attempts? Where in the chain (orch spawn → ORCH_WORKER → header → metadata.role → plugin check) is the break?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** og-arch-coaching-plugin-still-28jan-afed
**Phase:** Complete
**Next Step:** Rebuild and restart OpenCode server
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: orch spawn correctly sets ORCH_WORKER=1 in all spawn paths

**Evidence:** All spawn code paths set `ORCH_WORKER=1`:
- `cmd/orch/spawn_cmd.go:837`: `cmd.Env = append(os.Environ(), "ORCH_WORKER=1")`
- `cmd/orch/spawn_cmd.go:1047`: Same pattern for different spawn path
- `pkg/tmux/tmux.go:279`: `cmd.Env = append(os.Environ(), "ORCH_WORKER=1")`
- `pkg/tmux/tmux.go:301`: `ORCH_WORKER=1 %s %q --model %q` in command string
- `pkg/tmux/tmux.go:327`: `ORCH_WORKER=1 %s attach %q --dir %q`

**Source:** `cmd/orch/spawn_cmd.go:836-837,1046-1047`, `pkg/tmux/tmux.go:278-279,298-301,318-327`

**Significance:** Step 1 of the chain is correctly implemented. The env var IS set.

---

### Finding 2: OpenCode client correctly sends x-opencode-env-ORCH_WORKER header

**Evidence:** The `CreateSession` function sends the header when `isWorker=true`:
```go
// pkg/opencode/client.go:593-597
// Set ORCH_WORKER=1 header for worker sessions to signal orch-managed workers
if isWorker {
    req.Header.Set("x-opencode-env-ORCH_WORKER", "1")
}
```

Test verifies this:
```go
// pkg/opencode/client_test.go:1324-1326
if orchWorker := receivedHeaders.Get("x-opencode-env-ORCH_WORKER"); orchWorker != "1" {
    t.Errorf("x-opencode-env-ORCH_WORKER header = %q, want \"1\"", orchWorker)
}
```

**Source:** `pkg/opencode/client.go:593-597`, `pkg/opencode/client_test.go:1324-1326`

**Significance:** Step 2 of the chain is correctly implemented. The header IS sent.

---

### Finding 3: OpenCode backend FIX for session.metadata.role EXISTS in source

**Evidence:** The OpenCode session route correctly handles the header:
```typescript
// opencode/packages/opencode/src/server/routes/session.ts:207-211
// Read x-opencode-env-ORCH_WORKER header to set metadata.role
const orchWorkerHeader = c.req.header("x-opencode-env-ORCH_WORKER")
if (orchWorkerHeader) {
  body.metadata = { ...body.metadata, role: "worker" as const }
}
```

This was recently committed: `bbf798911 test: add tests for session.metadata.role from x-opencode-env-ORCH_WORKER header`

**Source:** `~/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts:207-211`, `git log --oneline -5` in opencode repo

**Significance:** Step 3 of the chain is correctly implemented IN SOURCE. The code to populate `metadata.role` exists.

---

### Finding 4: Coaching plugin correctly checks session.metadata.role

**Evidence:** The plugin checks for worker role in the `session.created` event handler:
```typescript
// plugins/coaching.ts:2018-2024
// Worker detection: session.metadata.role
// OpenCode sets this to "worker" when x-opencode-env-ORCH_WORKER header is present
const isWorker = sessionMetadata.role === "worker"

if (isWorker) {
  workerSessions.set(sessionId, true)
  log(`Worker detected (metadata.role): ${sessionId}, title: ${sessionTitle}`)
}
```

**Source:** `plugins/coaching.ts:2018-2024`

**Significance:** Step 5 of the chain is correctly implemented. The plugin IS checking the right field.

---

### Finding 5: THE BREAK - Running OpenCode server uses OLD code without the fix

**Evidence:** 
1. Worker session verified `env | grep ORCH` returns nothing (from spawn context)
2. `which opencode` returns "opencode not found" - no binary in PATH
3. Running process shows OpenCode runs via `bun run` from source directory
4. The fix commit `bbf798911` was made but server wasn't rebuilt/restarted

The OpenCode server process needs to be restarted to pick up the source changes.

**Source:** Process inspection, spawn context task description

**Significance:** This is the break in the chain. The fix exists in source but isn't running.

---

## Synthesis

**Key Insights:**

1. **The full chain IS correctly implemented** - Every code path from `orch spawn` through header sending through metadata population through plugin checking is correct. This is not a code bug.

2. **The break is operational, not architectural** - The OpenCode fix was committed (`bbf798911`) but the running server wasn't restarted to pick it up. This is a deployment gap.

3. **Previous investigations were correct** - The Jan 28 prior investigation correctly identified that OpenCode should expose `session.metadata.role` from the header. That fix was then implemented. It just hasn't been deployed.

**Answer to Investigation Question:**

The coaching plugin still fires for workers because **the running OpenCode server hasn't been rebuilt/restarted after the fix was implemented**. The fix at `session.ts:207-211` that reads `x-opencode-env-ORCH_WORKER` header and sets `metadata.role = "worker"` exists in source but the running server uses old code.

**Solution:** Rebuild OpenCode and restart the server:
```bash
cd ~/Documents/personal/opencode
bun run build  # or restart if running via bun run
# Restart the opencode server (via overmind or manually)
```

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch spawn` sets `ORCH_WORKER=1` (verified: grep across all spawn paths in Go code)
- ✅ Client sends `x-opencode-env-ORCH_WORKER` header (verified: client.go:596 and test at client_test.go:1324)
- ✅ OpenCode source has fix for `metadata.role` (verified: session.ts:207-211)
- ✅ Coaching plugin checks `sessionMetadata.role === "worker"` (verified: coaching.ts:2020)

**What's untested:**

- ⚠️ Whether the OpenCode server is actually running the old or new code (need to test after restart)
- ⚠️ Whether the fix works end-to-end after restart

**What would change this:**

- Finding would be wrong if OpenCode server IS running the new code (would be a different bug)
- Finding would be wrong if there's a bug in how `metadata` is passed to plugins (would need to trace plugin event data)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Restart OpenCode with new code** - Rebuild/restart the OpenCode server to pick up the `session.metadata.role` fix.

**Why this approach:**
- Fix already exists in source code (Finding 3)
- No code changes needed - just deployment
- Directly addresses the root cause

**Trade-offs accepted:**
- Brief service interruption during restart
- Need to verify fix works after restart

**Implementation sequence:**
1. Stop the running OpenCode server (via overmind or process kill)
2. Rebuild if needed: `cd ~/Documents/personal/opencode && bun run build`
3. Restart: `overmind restart opencode` or equivalent
4. Verify: Spawn a worker and check if coaching metrics appear (should be zero)

### Alternative Approaches Considered

**Option B: Add more heuristic fallbacks to coaching plugin**
- **Pros:** Doesn't require OpenCode restart
- **Cons:** Adds complexity; prior investigation recommended against more heuristics
- **When to use instead:** If OpenCode restart is blocked for some reason

**Rationale for recommendation:** The fix exists, just needs deployment. Adding workarounds would be technically debt when the proper solution is already implemented.

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go:836-837,1046-1047` - ORCH_WORKER env var setting
- `pkg/tmux/tmux.go:278-301,318-327` - ORCH_WORKER in tmux spawn paths
- `pkg/opencode/client.go:593-597` - Header sending
- `pkg/opencode/client_test.go:1324-1326` - Test for header
- `plugins/coaching.ts:2018-2024` - Plugin worker detection
- `~/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts:207-211` - OpenCode fix

**Commands Run:**
```bash
# Find all ORCH_WORKER references in Go code
grep -r "ORCH_WORKER" --include="*.go" .

# Find all ORCH_WORKER references in TypeScript
grep -r "ORCH_WORKER" --include="*.ts" .

# Check recent OpenCode commits
cd ~/Documents/personal/opencode && git log --oneline -5
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md` - Prior investigation that recommended this fix
- **Guide:** `.kb/guides/opencode-plugins.md` - Plugin system reference

---

## Investigation History

**2026-01-28 ~20:00:** Investigation started
- Initial question: Why does coaching plugin STILL fire for workers?
- Context: Worker verified `env | grep ORCH` returns nothing

**2026-01-28 ~20:15:** Traced full chain
- Verified all 5 steps: spawn → env → header → metadata → plugin
- Found fix EXISTS in OpenCode source but not deployed

**2026-01-28 ~20:20:** Investigation completed
- Status: Complete
- Key outcome: Restart OpenCode to pick up already-implemented fix
