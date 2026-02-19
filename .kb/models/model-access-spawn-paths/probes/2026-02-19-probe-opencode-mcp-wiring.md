# Probe: OpenCode MCP Flag Wiring

**Status:** Complete
**Date:** 2026-02-19
**Model:** model-access-spawn-paths
**Beads:** orch-go-1125

## Question

Does the OpenCode backend properly wire MCP config when `--mcp` flag is set on spawn?

The model claims dual spawn architecture with Claude CLI as escape hatch and OpenCode as primary path. The `--mcp` flag was wired through Claude CLI backend (orch-go-1123) but OpenCode backend silently ignores it. This probe tests whether adding `opencode.json` MCP injection fixes the gap.

## What I Tested

### Test 1: OpenCode MCP config format differs from Claude format
- Claude: `{"mcpServers":{"playwright":{"command":"npx","args":["-y","@playwright/mcp@latest"]}}}`
- OpenCode: `{"mcp":{"playwright":{"type":"local","command":["npx","-y","@playwright/mcp@latest"],"enabled":true}}}`
- Key differences: top-level key (`mcp` vs `mcpServers`), command format (single array vs command+args), extra fields (`type`, `enabled`)

### Test 2: EnsureOpenCodeMCP writes/merges config
- Ran `go test ./pkg/spawn/ -run "TestEnsureOpenCodeMCP" -v`
- 5 subtests: creates file when missing, preserves existing keys, preserves other MCP servers, handles already-present preset, rejects unknown preset
- All 5 pass

### Test 3: Build and vet pass
- `go build ./cmd/orch/` — success
- `go vet ./cmd/orch/` — success
- Pre-existing resolve_test.go failures unrelated to this change

## What I Observed

- OpenCode reads MCP config from `opencode.json` in the project directory at session startup
- The `DispatchSpawn` function in `pkg/orch/extraction.go` is the single entry point for all spawn modes
- Injecting MCP config at the top of `DispatchSpawn` (before mode routing) catches all three OpenCode paths: inline, headless, and tmux
- Claude backend is correctly excluded (it uses `--mcp-config` CLI flag)

## Model Impact

- **Extends** invariant about dual spawn architecture: MCP support now works across both paths (Claude CLI and OpenCode), closing a feature gap where `--mcp` was silently dropped for OpenCode spawns.
- The injection is a warning-on-failure (non-blocking) — if opencode.json write fails, spawn proceeds without MCP. This matches the model's resilience pattern.
