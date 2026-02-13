## Summary (D.E.K.N.)

**Delta:** Headless spawn uses `opencode run --attach` CLI which doesn't forward client CWD to server; sessions default to server's CWD instead of target project directory.

**Evidence:** OpenCode server.ts middleware reads `x-opencode-directory` header or falls back to `process.cwd()`; the CLI's run.ts doesn't pass `directory: process.cwd()` when using `--attach` mode; verified by code inspection of both OpenCode (run.ts, server.ts) and orch-go (spawn_cmd.go).

**Knowledge:** The HTTP API (CreateSession + SendMessageInDirectory) properly supports directory via `x-opencode-directory` header, bypassing the CLI limitation entirely.

**Next:** Fix implemented and committed. Inline spawn has the same bug (orch-go-r6r created).

**Authority:** implementation - Direct bug fix within existing API patterns, no architectural change.

---

# Investigation: Headless Spawn Doesn't Set Agent Working Directory

**Question:** Why does headless spawn not set the agent's working directory to the --workdir value?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: OpenCode run --attach doesn't forward client CWD

**Evidence:** In OpenCode's `run.ts` (line 570-572), when `args.attach` is set, the SDK client is created with only `baseUrl` - no `directory` parameter. The SDK supports passing `directory` which sets `x-opencode-directory` header, but `run.ts` doesn't use it.

**Source:** `~/Documents/personal/opencode/packages/opencode/src/cli/cmd/run.ts:570-572`, `~/Documents/personal/opencode/.opencode/node_modules/@opencode-ai/sdk/dist/v2/client.js:17-24`

**Significance:** This is the root cause. Even though orch-go correctly sets `cmd.Dir = cfg.ProjectDir`, the opencode CLI subprocess doesn't communicate that CWD to the server.

---

### Finding 2: OpenCode server uses directory resolution chain

**Evidence:** The server middleware (`server.ts:187-204`) resolves directory via: `query("directory")` || `header("x-opencode-directory")` || `process.cwd()`. Without the header, all sessions default to the server's startup directory.

**Source:** `~/Documents/personal/opencode/packages/opencode/src/server/server.ts:187-204`

**Significance:** The server already supports directory specification - it just needs to be passed by the client.

---

### Finding 3: HTTP API already supports directory via header

**Evidence:** orch-go's `CreateSession()` method already passes `x-opencode-directory` header (client.go:554-556). The method was implemented but not used for headless spawns because of a documented concern that "HTTP API ignores model parameter."

**Source:** `pkg/opencode/client.go:534-580` (CreateSession), `pkg/opencode/client.go:232-268` (SendMessageAsync with per-message model)

**Significance:** The fix is to switch headless spawn from CLI subprocess to HTTP API calls, which properly passes the directory header AND supports per-message model selection.

---

## Synthesis

**Key Insights:**

1. **Process CWD != Session CWD** - Setting `cmd.Dir` on the opencode CLI subprocess only affects the subprocess's working directory, not the OpenCode server's session directory. The server ignores client CWD in `--attach` mode.

2. **HTTP API is the correct path** - The HTTP API (`CreateSession` + `SendMessageInDirectory`) properly supports directory via `x-opencode-directory` header. The original concern about "HTTP API ignores model" is addressed by per-message model selection in `SendMessageAsync`.

3. **Same bug exists in inline spawn** - `runSpawnInline()` at line 1449 has the identical `cmd.Dir = cfg.ProjectDir` pattern. Created orch-go-r6r for tracking.

**Answer to Investigation Question:**

Headless spawn doesn't set agent working directory because `opencode run --attach` doesn't forward the client process's CWD to the server. The fix is to use the HTTP API (CreateSession + SendMessageInDirectory) which passes `x-opencode-directory` header, allowing the server to create the session in the correct project directory.

---

## Structured Uncertainty

**What's tested:**

- go build ./cmd/orch/ passes (verified: ran build)
- go vet passes on all changed packages (verified: ran vet)
- SendMessageInDirectory sends x-opencode-directory header (verified: unit test TestSendMessageInDirectory)
- Empty directory omits header (verified: unit test TestSendMessageInDirectoryEmpty)

**What's untested:**

- End-to-end cross-project headless spawn (requires running OpenCode server + target project)
- Model selection via HTTP API SendMessageInDirectory (per-message model format works in SendMessageAsync tests but not tested end-to-end)
- Whether ORCH_WORKER=1 is properly communicated (CreateSession sets x-opencode-env-ORCH_WORKER header, but untested e2e)

**What would change this:**

- If OpenCode server ignores x-opencode-directory header on prompt_async endpoint (would need to test e2e)
- If per-message model selection doesn't work with HTTP API (would need the old CLI approach with a different directory solution)

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` - Main spawn command, startHeadlessSession function
- `pkg/opencode/client.go` - OpenCode HTTP client, BuildSpawnCommand, CreateSession, SendMessageAsync
- `pkg/spawn/config.go` - Spawn config with ProjectDir field
- `pkg/spawn/backends/headless.go` - Dead code backend (updated for consistency)
- `~/Documents/personal/opencode/packages/opencode/src/cli/cmd/run.ts` - OpenCode CLI run command
- `~/Documents/personal/opencode/packages/opencode/src/server/server.ts` - Server directory middleware

**Commands Run:**
```bash
go build ./cmd/orch/
go vet ./cmd/orch/ && go vet ./pkg/opencode/ && go vet ./pkg/spawn/backends/
go test ./pkg/opencode/ -run "TestSendMessageInDirectory" -v
go test ./pkg/opencode/ -v
go test ./pkg/spawn/backends/ -v
```
