# Session Synthesis

**Agent:** og-debug-headless-spawn-doesn-13feb-fba8
**Issue:** orch-go-xwk
**Duration:** 2026-02-13
**Outcome:** success

---

## TLDR

Fixed headless spawn workdir bug by switching from CLI subprocess (`opencode run --attach`) to HTTP API (`CreateSession` + `SendMessageInDirectory`), which properly passes `x-opencode-directory` header to set the session's working directory.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/spawn_cmd.go` - Replaced `startHeadlessSession` from CLI subprocess to HTTP API; removed subprocess fields from `headlessSpawnResult`; removed unused `bytes` and `io` imports
- `pkg/opencode/client.go` - Added `SendMessageInDirectory()` method that sends `x-opencode-directory` header with messages
- `pkg/opencode/client_test.go` - Added `TestSendMessageInDirectory` and `TestSendMessageInDirectoryEmpty` tests
- `pkg/spawn/backends/headless.go` - Updated dead code backend for consistency (same HTTP API approach)

---

## Evidence (What Was Observed)

- OpenCode `run.ts:570-572` creates SDK client without `directory` param when `--attach` is used
- OpenCode `server.ts:187-204` middleware: `query("directory") || header("x-opencode-directory") || process.cwd()`
- Without the header, ALL sessions default to server's startup directory
- `cmd.Dir = cfg.ProjectDir` only sets subprocess CWD - doesn't reach the server
- `CreateSession()` already sends `x-opencode-directory` header - just wasn't used for headless

### Tests Run
```bash
go build ./cmd/orch/
# PASS

go vet ./cmd/orch/ && go vet ./pkg/opencode/ && go vet ./pkg/spawn/backends/
# PASS

go test ./pkg/opencode/ -run "TestSendMessageInDirectory" -v
# PASS: TestSendMessageInDirectory, TestSendMessageInDirectoryEmpty

go test ./pkg/opencode/ -v
# PASS: all 35+ tests

go test ./pkg/spawn/backends/ -v
# PASS: all tests
```

---

## Verification Contract

See: `VERIFICATION_SPEC.yaml` in this workspace.

Key outcomes:
- Build passes: `go build ./cmd/orch/`
- Unit tests pass: `go test ./pkg/opencode/ -v`
- `SendMessageInDirectory` sends `x-opencode-directory` header (unit tested)
- E2E verification requires running OpenCode server with cross-project spawn

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-13-inv-headless-spawn-doesn-set-agent.md` - Root cause analysis and fix documentation

### Decisions Made
- Switch from CLI subprocess to HTTP API for headless spawn because HTTP API properly supports `x-opencode-directory` header while CLI `--attach` mode doesn't forward CWD
- Per-message model selection (providerID/modelID in SendMessageAsync payload) replaces the CLI's `--model` flag

### Constraints Discovered
- `opencode run --attach` ignores client CWD for session directory - this is a bug in OpenCode itself
- The "HTTP API ignores model parameter" concern from the original investigation was about session-level model, but per-message model selection works

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-xwk`

### Discovered Work
- Created orch-go-r6r: Inline spawn (`--inline`) has the same workdir bug

---

## Unexplored Questions

- Does OpenCode's `prompt_async` endpoint properly use `x-opencode-directory` for tool execution context? (Unit tested header propagation, but e2e verification needed)
- Should OpenCode's `run.ts` be fixed upstream to pass `directory: process.cwd()` when `--attach` is used? This would fix all CLI clients, not just orch-go.

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-headless-spawn-doesn-13feb-fba8/`
**Investigation:** `.kb/investigations/2026-02-13-inv-headless-spawn-doesn-set-agent.md`
**Beads:** `bd show orch-go-xwk`
