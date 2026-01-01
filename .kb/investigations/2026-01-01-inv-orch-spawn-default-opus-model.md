<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Headless spawn mode was using HTTP API (startHeadlessSessionAPI) which ignores the model parameter; switched to CLI mode (startHeadlessSession) which correctly honors --model flag.

**Evidence:** Prior investigation confirmed OpenCode HTTP API ignores model in POST /session; code trace showed runSpawnHeadless called startHeadlessSessionAPI; tests pass after switching to startHeadlessSession.

**Knowledge:** OpenCode has two spawn paths: HTTP API (ignores model) and CLI (honors --model). MCP support was missing from CLI path but has been added.

**Next:** Verify end-to-end by spawning agent and checking model in session (orchestrator task).

---

# Investigation: Orch Spawn Default Opus Model

**Question:** Why does orch spawn default to Sonnet model instead of Opus, and how can we fix the model selection for headless spawns?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** og-debug-orch-spawn-default-01jan
**Phase:** Complete
**Next Step:** None (fix implemented)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** .kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md (implements its recommendations)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Prior investigation identified root cause

**Evidence:** The investigation at `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md` already identified that:
- OpenCode's HTTP API (`POST /session` with model in body) ignores the model parameter
- OpenCode's CLI (`opencode run --attach --model`) correctly honors the model flag
- The recommended fix was to switch headless spawn to use CLI mode

**Source:** 
- `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md`

**Significance:** No new root cause investigation needed - the issue was already well understood and a solution was proposed.

---

### Finding 2: Two headless session functions existed but only API version was used

**Evidence:** Two functions exist in `cmd/orch/main.go`:
- `startHeadlessSession` (line 2023): Uses CLI mode via `BuildSpawnCommand`, supports model correctly
- `startHeadlessSessionAPI` (line 2087): Uses HTTP API, ignores model

The `runSpawnHeadless` function (line 1830) was calling `startHeadlessSessionAPI`.

**Source:** 
- `cmd/orch/main.go:1840` - call to `startHeadlessSessionAPI`
- `cmd/orch/main.go:2023` - unused `startHeadlessSession` function
- `cmd/orch/main.go:2087` - `startHeadlessSessionAPI` function

**Significance:** The fix was already partially implemented - just needed to switch which function is called.

---

### Finding 3: CLI mode was missing MCP support

**Evidence:** `startHeadlessSession` only set `ORCH_WORKER=1` in environment, but didn't set `OPENCODE_CONFIG_CONTENT` for MCP configuration. Compare with `runSpawnInline` which had:
```go
if cfg.MCP != "" {
    cmd.Env = append(cmd.Env, "OPENCODE_CONFIG_CONTENT="+mcpConfigContent)
}
```

**Source:** 
- `cmd/orch/main.go:1751-1757` - inline mode MCP support
- `cmd/orch/main.go:2027` - headless CLI mode originally missing MCP

**Significance:** Had to add MCP support to make CLI mode feature-equivalent with API mode before switching.

---

## Synthesis

**Key Insights:**

1. **Existing solution was already in codebase** - The CLI-based `startHeadlessSession` function existed but wasn't being used. The fix was to switch from API mode to CLI mode.

2. **HTTP API was chosen for directory handling** - The API mode was originally preferred because "The opencode CLI's --attach mode has a bug where it always uses '/' as the directory" (per comment at line 2087). However, this sacrificed model selection capability.

3. **CLI mode needed MCP parity** - Before switching, had to add MCP config environment variable support to ensure feature parity with API mode.

**Answer to Investigation Question:**

Headless spawns defaulted to Sonnet because `runSpawnHeadless` called `startHeadlessSessionAPI`, which uses the OpenCode HTTP API that ignores the model parameter. The fix was to:
1. Add MCP support to `startHeadlessSession` (CLI mode)
2. Switch `runSpawnHeadless` to call `startHeadlessSession` instead of `startHeadlessSessionAPI`

This change ensures the `--model` flag (which defaults to opus via `model.Resolve("")`) is correctly passed to OpenCode via CLI.

---

## Structured Uncertainty

**What's tested:**

- ✅ Code compiles without errors (verified: `go build ./cmd/orch/`)
- ✅ All existing tests pass (verified: `go test ./cmd/orch/...` passes)
- ✅ MCP config is added to CLI mode environment (verified: code review of change)

**What's untested:**

- ⚠️ End-to-end model selection (spawning agent and checking actual model used) - requires running orch spawn and checking session
- ⚠️ Directory handling with CLI mode (the original reason API was chosen) - needs real-world validation
- ⚠️ MCP servers work correctly in CLI mode (same as API mode)

**What would change this:**

- If CLI mode's `cmd.Dir` doesn't properly set the session directory, we may need a hybrid approach
- If MCP config via environment variable doesn't work in attach mode, need different approach

---

## Implementation Recommendations

**Purpose:** This section documents the implementation that was done, not future recommendations.

### Implemented Approach ⭐

**Switch from HTTP API to CLI mode for headless spawns**

**Changes made:**
1. Added MCP config support to `startHeadlessSession`:
   ```go
   if cfg.MCP != "" {
       mcpConfigContent, err := spawn.GenerateMCPConfig(cfg.MCP)
       if err != nil {
           return nil, spawn.WrapSpawnError(err, "Failed to generate MCP config")
       }
       cmd.Env = append(cmd.Env, "OPENCODE_CONFIG_CONTENT="+mcpConfigContent)
   }
   ```

2. Changed `runSpawnHeadless` to call `startHeadlessSession` instead of `startHeadlessSessionAPI`:
   ```go
   result, retryResult := spawn.Retry(retryCfg, func() (*headlessSpawnResult, error) {
       return startHeadlessSession(client, serverURL, sessionTitle, minimalPrompt, cfg, verbose)
   })
   ```

3. Updated comments on `startHeadlessSessionAPI` to mark it as deprecated

**Why this approach:**
- Model selection works immediately (CLI honors --model flag)
- Reuses existing tested code path (same as inline mode)
- Maintains headless behavior (non-blocking, background process)

**Trade-offs accepted:**
- Subprocess management vs pure HTTP
- Potential directory issues (need validation)

---

## References

**Files Examined:**
- `cmd/orch/main.go` - spawn command implementation
- `pkg/opencode/client.go` - BuildSpawnCommand function
- `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md` - prior investigation

**Commands Run:**
```bash
# Build to verify changes compile
/usr/local/go/bin/go build ./cmd/orch/

# Run tests
/usr/local/go/bin/go test ./cmd/orch/...
/usr/local/go/bin/go test ./pkg/opencode/...
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md` - identified root cause
- **Beads Issue:** orch-go-ys34 - tracking this work

---

## Investigation History

**2026-01-01 22:20:** Investigation started
- Initial question: Why does orch spawn default to Sonnet instead of Opus?
- Context: Prior investigation had identified root cause but fix wasn't implemented

**2026-01-01 22:25:** Reviewed prior investigation
- Found complete root cause analysis and implementation recommendations
- Confirmed two spawn paths exist (API and CLI)

**2026-01-01 22:30:** Implemented fix
- Added MCP support to `startHeadlessSession`
- Switched `runSpawnHeadless` to use CLI mode
- Verified all tests pass

**2026-01-01 22:35:** Investigation completed
- Status: Complete
- Key outcome: Headless spawns now use CLI mode which correctly honors --model flag
