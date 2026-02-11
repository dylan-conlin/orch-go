# Investigation: Spawn Preflight Checks and Post-Spawn Liveness Probe

**Date:** 2026-02-11  
**Investigator:** Agent (orch-go-8pzpb)  
**Status:** Complete

## Question

How does `orch spawn` currently handle spawn failures, and where should we add pre-spawn runtime checks and post-spawn liveness probes?

## Findings

### Finding 1: All tmux-based spawns return success immediately without verification

**Evidence:**

- `cmd/orch/spawn_execute.go:434-590` - `runSpawnTmux` function:
  - Creates tmux window (line 453)
  - Sends opencode command (lines 473-480)
  - Returns success immediately after sending keys
  - No verification that opencode started successfully

- `pkg/spawn/claude.go:25-94` - `SpawnClaude` function:
  - Creates tmux window (line 46)
  - Sends `cat CONTEXT.md | claude` command (lines 81-86)
  - Returns immediately after sending enter (line 88)
  - No verification that claude CLI started

- `pkg/spawn/docker.go:29-130` - `SpawnDocker` function:
  - Creates tmux window (line 41)
  - Sends `docker run` command (lines 118-122)
  - Returns immediately after sending enter
  - No verification that Docker daemon is running

**Source:** Direct code reading  
**Significance:** This is the root cause of the bug. All tmux-based spawn functions report success after creating the window and sending the command, but never check if the runtime is available or if the agent actually started.

---

### Finding 2: Error patterns in tmux panes are never captured

**Evidence:**

When a spawn fails (e.g., Docker daemon not running, Claude CLI not found, OpenCode API down), the error appears in the tmux pane but the orchestrator already exited with status 0.

Example scenarios:
- Docker daemon not running → `Cannot connect to the Docker daemon` appears in tmux
- Claude CLI not found → `claude: command not found` appears in tmux
- OpenCode API down → `Connection refused` appears in tmux

**Source:** Task description reproduction steps  
**Significance:** The orchestrator has no way to detect these failures. Errors are silently hidden in tmux panes while the orchestrator proceeds thinking the spawn succeeded.

---

### Finding 3: Spawn pipeline has phases but no pre-spawn runtime validation

**Evidence:**

`cmd/orch/spawn_pipeline.go` has these phases:
1. `runPreFlightValidation` (line 99-191) - checks triage bypass, concurrency, rate limits, hotspots
2. `resolveProject` (line 194-224) - resolves project directory
3. `loadSkill` (line 228-257) - loads skill content
4. `setupIssueTracking` (line 261-359) - handles beads tracking
5. `gatherContext` (line 362-410) - gathers KB context
6. `buildSpawnConfig` (line 461-634) - builds spawn config
7. `executeSpawn` (line 661-706) - validates config and dispatches to backend

None of these phases check if the target runtime is available.

**Source:** `cmd/orch/spawn_pipeline.go:1-741`  
**Significance:** We need a new phase between `buildSpawnConfig` and `executeSpawn` to verify runtime availability.

---

### Finding 4: tmux package has capture-pane support

**Evidence:**

`pkg/spawn/claude.go:22` defines:
```go
getTmuxPaneContent = tmux.GetPaneContent
```

This is used by `MonitorClaude` (line 97-99) to capture pane content. The same mechanism can be used for post-spawn liveness checks.

**Source:** `pkg/spawn/claude.go:22,97-99`  
**Significance:** We already have the infrastructure to capture tmux pane content - we just need to use it after spawning.

---

### Finding 5: Three backends need runtime checks (OpenCode, Claude, Docker)

**Evidence:**

From `cmd/orch/spawn_pipeline.go:709-740` - `dispatchSpawn` function routes to:
- `runSpawnClaude` / `runSpawnClaudeInline` - requires `claude` CLI binary
- `runSpawnDocker` - requires Docker daemon running
- `runSpawnTmux` / `runSpawnHeadless` / `runSpawnInline` - requires OpenCode API at serverURL

**Source:** `cmd/orch/spawn_pipeline.go:709-740`  
**Significance:** We need different preflight checks depending on the backend:
- **OpenCode backend:** Check if API responds at serverURL
- **Claude backend:** Check if `claude` binary exists and is executable
- **Docker backend:** Check if Docker daemon socket exists and is accessible

---

## Synthesis

The current spawn implementation has a critical gap: all tmux-based spawns (OpenCode tmux, Claude, Docker) report success immediately after creating the tmux window and sending the command, without verifying that the underlying runtime is available or that the agent actually started.

**Two fixes are needed:**

1. **Pre-spawn runtime checks** - Add a new pipeline phase before `executeSpawn` to verify:
   - Docker backend: `/var/run/docker.sock` exists and is accessible
   - Claude backend: `claude` binary exists in PATH
   - OpenCode backend: API responds at serverURL (health check or ping)
   - If unavailable: fail with actionable error message

2. **Post-spawn liveness probe** - Modify tmux spawn functions to verify agent startup:
   - Wait 5-10 seconds after sending command
   - Use `tmux.GetPaneContent()` to capture pane output
   - Check for error patterns: "Cannot connect", "command not found", "connection refused"
   - If error detected: kill tmux window, clean up, return error with captured output

**Implementation locations:**

- **Pre-spawn checks:** New function in `cmd/orch/spawn_pipeline.go`, called in `executeSpawn` before dispatching
- **Post-spawn liveness:** Modify `runSpawnTmux`, `SpawnClaude`, `SpawnDocker` to add verification after sending command

**Error patterns to detect:**
- Docker: "Cannot connect to the Docker daemon", "docker: command not found"
- Claude: "claude: command not found", "No such file or directory"
- OpenCode: "Connection refused", "Cannot connect", "ECONNREFUSED"

---

## Recommendations

1. Create preflight check functions for each backend in `cmd/orch/spawn_execute.go`:
   - `checkDockerAvailable()` - check Docker daemon socket
   - `checkClaudeAvailable()` - check Claude CLI binary
   - `checkOpencodeAvailable(serverURL)` - ping OpenCode API

2. Add liveness probe helper in `pkg/tmux/`:
   - `ProbeWindowForErrors(windowTarget, timeout, errorPatterns)` - wait, capture, check patterns

3. Modify spawn functions to use liveness probe:
   - `runSpawnTmux` - after sending prompt
   - `SpawnClaude` - after sending command
   - `SpawnDocker` - after sending docker run

4. Provide actionable error messages:
   - Docker: "Docker daemon not running. Start with: colima start"
   - Claude: "claude CLI not found. Install from: https://github.com/anthropics/claude-code"
   - OpenCode: "OpenCode API not responding at {serverURL}. Start with: orch-dashboard start"
