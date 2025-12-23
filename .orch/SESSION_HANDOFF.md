# Session Handoff - 2025-12-23

## What Happened This Session

### Fixed: Orchestrator Skill Loading Bug
- **Problem:** Workers were getting orchestrator skill (1224 lines) loaded despite `audience: orchestrator` field
- **Root cause:** session-context plugin checked `ORCH_WORKER` at plugin init (once), not per-session in config hook
- **Fix:** `orch-cli@ac945ea` - moved check into config hook
- **Deployed:** Plugin rebuilt to `~/.config/opencode/plugin/session-context.js`

### Fixed: Web UI Port Chaos
- **Problem:** Web UI showed 0 agents, intermittent failures, confusing errors
- **Root causes:** 
  1. `orch serve` used dynamic port (3348 from registry) but web UI hardcoded to 3333
  2. Multiple `orch serve` processes running
  3. Race conditions on page load
  4. Missing favicon
- **Fix:** Standardized fixed ports:
  - OpenCode: **4096**
  - orch serve API: **3348** 
  - Web UI dev: **5188**
- **Commits:** `7fc718c` (port standardization, favicon, SSE cleanup)

### Shipped: Project-Declared Server Ports
- **New feature:** `.orch/config.yaml` with `servers:` section for declaring ports per-project
- **New package:** `pkg/config` for parsing project config
- **Commit:** `3a5cf17`
- Example:
  ```yaml
  servers:
    web: 5173
    api: 3000
  ```

## Investigations Completed

1. **orch-go-oh2d:** Server awareness for workers - conditional inclusion based on skill type recommended
2. **orch-go-tymf:** Find command performance - OpenCode already has fast glob/grep via ripgrep, behavioral issue not tooling
3. **orch-go-d3yi:** External content workflow - WebFetch already works, needs documentation in research skill
4. **orch-go-ndgj:** Nate Jones "LLM psychosis" article - research on AI validation loops (check workspace)

## Next Session Priority

**orch-go-g1cz: Harden orch servers command**
- Test and reliability work after port config feature landed
- Unit tests for pkg/port and pkg/config
- Integration tests for orch servers list/start/stop/attach/open  
- Error handling (missing config, port conflicts, tmux not running)
- Documentation updates
- Smoke test across multiple projects

## Open Issues

| ID | Priority | Description |
|----|----------|-------------|
| orch-go-g1cz | P1 | Harden orch servers - next session focus |
| orch-go-4ufh | P1 | `orch wait` fails with session ID |
| orch-go-xe2j | P2 | Add web-to-markdown MCP for research spawns |

## Key Decisions Made

1. **Fixed ports for infrastructure** - No more dynamic allocation for orch tooling itself
2. **Project config owns ports** - `.orch/config.yaml` declares, global registry just tracks runtime
3. **Slow down to debug** - Session got chaotic with too many agents; direct debugging was faster

## Account Status

- work: 36% used (resets in 6 days)
- personal: N/A

## Commands to Start

```bash
# Check status
orch status
bd ready

# Start servers for web UI
orch serve &  # Port 3348
cd web && bun run dev &  # Port 5188

# Next task
bd show orch-go-g1cz
```
