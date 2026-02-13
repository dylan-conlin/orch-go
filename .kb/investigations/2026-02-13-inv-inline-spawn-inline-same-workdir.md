## Summary (D.E.K.N.)

**Delta:** Inline spawn uses CLI subprocess (`opencode run --attach`) which doesn't forward CWD to server, identical to the headless bug fixed in commit f31e4ba0.

**Evidence:** Code inspection: `runSpawnInline()` called `BuildSpawnCommand()` (CLI subprocess with `cmd.Dir`), but OpenCode's `run.ts` in `--attach` mode doesn't pass directory to server. Verified fix compiles and passes unit tests.

**Knowledge:** All spawn modes that create sessions should use HTTP API with `x-opencode-directory` header, not CLI subprocess. The CLI's `--attach` mode is fundamentally broken for directory propagation.

**Next:** Fix implemented and committed. Both headless and inline spawn now use HTTP API.

**Authority:** implementation - Direct bug fix applying established pattern from headless fix, no architectural change.

---

# Investigation: Inline Spawn Same Workdir Bug as Headless

**Question:** Does inline spawn (`--inline`) have the same workdir bug as headless, and can the same HTTP API fix be applied?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-13-inv-headless-spawn-doesn-set-agent.md | extends | Yes - same root cause confirmed | None |

---

## Findings

### Finding 1: Inline spawn uses identical broken pattern

**Evidence:** `runSpawnInline()` at spawn_cmd.go:1442 calls `client.BuildSpawnCommand()` which creates `opencode run --attach --format json` subprocess. Sets `cmd.Dir = cfg.ProjectDir` but this doesn't propagate to the OpenCode server because `run.ts` in `--attach` mode doesn't pass directory.

**Source:** `cmd/orch/spawn_cmd.go:1447-1449` (before fix)

**Significance:** Confirms identical bug to headless spawn. Sessions created in inline mode get the server's CWD, not the target project directory.

---

### Finding 2: HTTP API with x-opencode-directory already available

**Evidence:** `CreateSession(title, directory, model)` passes `x-opencode-directory` header (client.go:554-556). `SendMessageInDirectory(sessionID, content, model, directory)` also passes the header (client.go:613-614). Both were added in the headless fix.

**Source:** `pkg/opencode/client.go:536-580` (CreateSession), `pkg/opencode/client.go:589-627` (SendMessageInDirectory)

**Significance:** The fix infrastructure already exists. Inline spawn just needs to use these API methods instead of CLI subprocess.

---

### Finding 3: Inline mode needs blocking behavior preserved

**Evidence:** Unlike headless (fire-and-forget), inline spawn blocks until the agent finishes (`cmd.Wait()` at line 1467). The HTTP API approach requires a separate wait mechanism. Added `WaitForSessionIdle()` which watches SSE events for the busy→idle transition, matching the pattern in `SendMessageWithStreaming()` (client.go:822).

**Source:** `pkg/opencode/client.go:629` (new WaitForSessionIdle), `pkg/opencode/client.go:822` (existing SSE pattern)

**Significance:** The SSE-based wait is more reliable than polling because it detects the exact busy→idle transition. Also handles session errors.

---

## Synthesis

**Key Insights:**

1. **CLI subprocess is fundamentally broken for directory** - `opencode run --attach` never propagates client CWD to server. All spawn modes should use HTTP API with `x-opencode-directory` header.

2. **HTTP API + SSE = blocking spawn** - Creating session via API then watching SSE for completion provides the same blocking behavior as CLI subprocess, with correct directory handling.

3. **ORCH_WORKER=1 already handled by API** - `CreateSession` sets `x-opencode-env-ORCH_WORKER=1` header (client.go:561), so no separate env var setup needed.

**Answer to Investigation Question:**

Yes, inline spawn has the identical workdir bug. The same HTTP API fix applies: replace `BuildSpawnCommand` (CLI subprocess) with `CreateSession` + `SendMessageInDirectory` (HTTP API with `x-opencode-directory` header). A new `WaitForSessionIdle` method preserves inline mode's blocking behavior.

---

## Structured Uncertainty

**What's tested:**

- go build ./cmd/orch/ passes (verified: ran build)
- go vet passes on all changed packages (verified: ran vet)
- WaitForSessionIdle returns on busy→idle transition (verified: TestWaitForSessionIdle)
- WaitForSessionIdle returns error on session error (verified: TestWaitForSessionIdleError)
- WaitForSessionIdle ignores events from other sessions (verified: TestWaitForSessionIdleIgnoresOtherSessions)

**What's untested:**

- End-to-end inline spawn with `--workdir` flag (requires running OpenCode server + target project)
- Model selection via HTTP API in inline mode (per-message model works in unit tests but not tested e2e)
- SSE connection failure recovery (if SSE drops during wait, no reconnection logic)

**What would change this:**

- If OpenCode server doesn't emit session.status events for API-created sessions (would need polling fallback)
- If SSE connection drops under load (would need reconnection logic in WaitForSessionIdle)

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - runSpawnInline function (modified)
- `pkg/opencode/client.go` - BuildSpawnCommand, CreateSession, SendMessageInDirectory, SendMessageWithStreaming (SSE pattern reference)
- `pkg/opencode/sse.go` - ParseSSEEvent, ParseSessionStatus
- `.kb/investigations/2026-02-13-inv-headless-spawn-doesn-set-agent.md` - Prior headless fix investigation

**Commands Run:**
```bash
go build ./cmd/orch/
go vet ./cmd/orch/ && go vet ./pkg/opencode/
go test ./pkg/opencode/ -run "TestWaitForSession" -v -timeout 30s
go test ./pkg/opencode/ -v -timeout 60s
```
