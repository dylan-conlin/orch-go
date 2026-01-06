# Session Synthesis

**Agent:** og-debug-fix-headless-spawn-23dec
**Issue:** orch-go-dwws
**Duration:** 2025-12-24 ~02:10 → ~02:25
**Outcome:** success

---

## TLDR

Fixed headless spawn to use CLI subprocess instead of HTTP API, enabling proper model selection. The `--model` flag now works correctly for headless spawns (verified with flash and opus models).

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/main.go` - Rewrote `runSpawnHeadless()` to use CLI subprocess with `BuildSpawnCommand()` instead of HTTP API (`CreateSession`/`SendPrompt`)
- `pkg/opencode/client.go` - Added `ExtractSessionIDFromReader()` function to extract session ID from streaming CLI output
- `pkg/opencode/client_test.go` - Added test cases for `ExtractSessionIDFromReader()`

### Key Changes
1. `runSpawnHeadless` now uses `client.BuildSpawnCommand()` like inline mode
2. Runs opencode CLI in background with `cmd.Start()` (non-blocking)
3. Reads session ID from JSON events via new `ExtractSessionIDFromReader()`
4. Spawns goroutine to drain stdout and cleanup process

---

## Evidence (What Was Observed)

- Prior investigation confirmed OpenCode HTTP API ignores model parameter (see `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md`)
- CLI mode (`opencode run --model`) correctly honors the model flag
- Smoke test: `orch spawn --model flash hello` used `gemini-2.5-flash` (verified via `/session/{id}/message` API)
- Smoke test: `orch spawn hello` used `claude-opus-4-5-20251101` (verified default model works)

### Tests Run
```bash
go test ./pkg/opencode/... -v
# PASS: all 55 tests passing (0.620s)

go test ./...
# PASS: all packages passing
```

---

## Knowledge (What Was Learned)

### Key Insight
OpenCode HTTP API doesn't support model selection in `POST /session` - this is an API limitation. The CLI (`opencode run --model`) is the only way to specify models for now.

### Decisions Made
- Use CLI subprocess for headless spawns (reusing `BuildSpawnCommand`)
- Accept subprocess overhead (negligible: <100ms) for model selection capability
- Spawn goroutine to drain stdout and prevent blocking

### Implementation Pattern
```go
// Extract session ID quickly, let process run in background
sessionID, err := opencode.ExtractSessionIDFromReader(stdout)
go func() {
    io.Copy(io.Discard, stdout)  // Drain remaining
    cmd.Wait()                    // Cleanup
}()
```

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Smoke-test passed (flash model, default opus model)
- [x] Ready for `orch complete orch-go-dwws`

---

## Unexplored Questions

**What remains unclear:**
- Whether OpenCode team plans to add model support to the HTTP API (could simplify implementation long-term)

*(Straightforward fix, no significant unexplored territory)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-debug-fix-headless-spawn-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-model-selection-issue-architect-agent.md` (prior investigation)
**Beads:** `bd show orch-go-dwws`
