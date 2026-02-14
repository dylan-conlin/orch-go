# Session Synthesis

**Agent:** og-debug-inline-spawn-inline-13feb-3efd
**Issue:** orch-go-r6r
**Duration:** 2026-02-13
**Outcome:** success

---

## TLDR

Fixed inline spawn (`--inline`) workdir bug by switching from CLI subprocess (`opencode run --attach`) to HTTP API (`CreateSession` + `SendMessageInDirectory` + `WaitForSessionIdle`), ensuring sessions are created in the correct project directory via `x-opencode-directory` header. Same fix pattern as the headless spawn fix (commit f31e4ba0).

---

## Delta (What Changed)

### Files Modified
- `pkg/opencode/client.go` - Added `WaitForSessionIdle()` method that watches SSE events for busy→idle transition, used by inline spawn to maintain blocking behavior
- `cmd/orch/spawn_cmd.go` - Rewrote `runSpawnInline()` to use HTTP API (CreateSession + SendMessageInDirectory + WaitForSessionIdle) instead of CLI subprocess (BuildSpawnCommand)
- `pkg/opencode/client_test.go` - Added 3 tests for WaitForSessionIdle: happy path, error handling, session filtering

### Files Created
- `.kb/investigations/2026-02-13-inv-inline-spawn-inline-same-workdir.md` - Investigation documenting root cause and fix

---

## Evidence (What Was Observed)

- `runSpawnInline()` used `BuildSpawnCommand()` which creates `opencode run --attach --format json` subprocess — identical broken pattern to headless
- `cmd.Dir = cfg.ProjectDir` sets subprocess CWD but OpenCode's `run.ts` in `--attach` mode doesn't pass directory to server (confirmed in prior headless investigation)
- `CreateSession` and `SendMessageInDirectory` already exist from headless fix, both pass `x-opencode-directory` header
- `SendMessageWithStreaming` (client.go:822) already has SSE waiting pattern — extracted same pattern for `WaitForSessionIdle`

### Tests Run
```bash
go build ./cmd/orch/
# Success - no errors

go vet ./cmd/orch/ && go vet ./pkg/opencode/
# Success - no issues

go test ./pkg/opencode/ -run "TestWaitForSession" -v -timeout 30s
# PASS: TestWaitForSessionIdle (0.00s)
# PASS: TestWaitForSessionIdleError (0.00s)
# PASS: TestWaitForSessionIdleIgnoresOtherSessions (0.00s)

go test ./pkg/opencode/ -v -timeout 60s
# PASS: all tests (0.272s)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-13-inv-inline-spawn-inline-same-workdir.md` - Documents inline workdir bug and fix

### Decisions Made
- Used SSE-based waiting (WaitForSessionIdle) over polling for inline blocking behavior because SSE detects exact busy→idle transition and handles session errors
- Reused existing HTTP API methods (CreateSession, SendMessageInDirectory) rather than creating inline-specific variants

### Constraints Discovered
- All spawn modes using `opencode run --attach` have the directory bug — CLI attach mode fundamentally doesn't propagate directory
- `BuildSpawnCommand` should potentially be deprecated or documented as directory-unaware

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-r6r`

---

## Unexplored Questions

- Tmux spawn mode (`runSpawnTmux`) also uses `opencode run --attach` (via `BuildOpencodeAttachCommand`) — may have the same directory bug for cross-project spawns, though tmux creates the window in the target directory so it might be mitigated
- `WaitForSessionIdle` has no reconnection logic if the SSE connection drops — could be an issue for long-running agents
- `BuildSpawnCommand` is now only used by tmux spawn — consider whether tmux should also switch to HTTP API

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for automated verification steps.

Key outcomes:
- `go build ./cmd/orch/` passes
- `go vet ./cmd/orch/ && go vet ./pkg/opencode/` passes
- `go test ./pkg/opencode/ -run TestWaitForSession -v` — 3 tests pass

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-debug-inline-spawn-inline-13feb-3efd/`
**Investigation:** `.kb/investigations/2026-02-13-inv-inline-spawn-inline-same-workdir.md`
**Beads:** `bd show orch-go-r6r`
