# Session Synthesis

**Agent:** og-debug-fix-orch-spawn-19feb-43cb
**Issue:** orch-go-1123
**Outcome:** success

---

## Plain-Language Summary

The `--mcp` flag on `orch spawn` was accepted by the CLI but never passed through to the Claude Code backend. When an agent was spawned with `--mcp playwright`, the resulting `claude` CLI process had no `--mcp-config` flag, so the agent couldn't use Playwright MCP tools. The fix adds MCP preset resolution (mapping "playwright" to its server config JSON) and wires the `--mcp-config` flag into the Claude CLI command construction. This is Claude-backend only; OpenCode backend MCP is server-level config and cannot be injected per-session.

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace.
Key outcome: `orch spawn --mcp playwright --backend claude` now passes `--mcp-config '{"mcpServers":{"playwright":{"command":"npx","args":["-y","@playwright/mcp@latest"]}}}'` to the Claude CLI.

---

## Delta (What Changed)

### Files Modified
- `pkg/spawn/claude.go` - Added `MCPConfigJSON()` for preset resolution, `BuildClaudeLaunchCommand()` for testable command construction, and `mcpPresets` map. Modified `SpawnClaude()` to use new builder.
- `pkg/spawn/claude_test.go` - Added `TestMCPConfigJSON` and `TestBuildClaudeLaunchCommand` tests.
- `pkg/orch/extraction.go` - Added MCP to event data and spawn summary output for Claude backend.

---

## Evidence (What Was Observed)

- Root cause: `claude.go:55` constructed command without checking `cfg.MCP`
- The `--mcp-config` flag on Claude CLI accepts inline JSON strings (verified via `claude --help`)
- Playwright MCP package is `@playwright/mcp` (verified via `npm info`)
- Pre-existing test failures in `resolve_test.go` (5 tests) unrelated to this change

### Tests Run
```bash
go test ./pkg/spawn/ -run "TestMCPConfigJSON|TestBuildClaudeLaunchCommand" -v
# PASS: 6/6 tests passing (2 MCP config + 4 command build)
go build ./cmd/orch/ && go vet ./cmd/orch/
# Build and vet passed
```

---

## Knowledge (What Was Learned)

### Constraints Discovered
- OpenCode backend (headless/tmux) uses server-level MCP config; no per-session MCP injection is possible. `--mcp` only works with `--backend claude`.

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All deliverables complete
- [x] Tests passing
- [x] Ready for `orch complete orch-go-1123`

---

## Unexplored Questions

- Should `orch spawn` warn when `--mcp` is used with a non-claude backend? Currently silently ignored.
- Future MCP presets beyond playwright (e.g., glass, web-to-markdown) could be added to `mcpPresets` map.

---

## Session Metadata

**Skill:** systematic-debugging
**Workspace:** `.orch/workspace/og-debug-fix-orch-spawn-19feb-43cb/`
**Beads:** `bd show orch-go-1123`
