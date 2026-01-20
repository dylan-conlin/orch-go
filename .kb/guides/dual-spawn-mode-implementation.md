# Triple Spawn Mode Implementation Guide

**Date:** 2026-01-09 (updated 2026-01-20)
**Decision:** `.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md`

## Overview

Implement triple backend system for agent spawning:
- **Claude mode:** tmux + `claude` CLI (Max subscription, unlimited Opus)
- **OpenCode mode:** HTTP API + dashboard (paid API)
- **Docker mode:** Host tmux + Docker container (fresh Statsig fingerprint, rate limit escape hatch)

## Implementation Order

```
1. Config (orch-go-5w0fj)
   ├─→ 2a. Claude spawn (orch-go-0z5i4)
   └─→ 2b. Registry schema (orch-go-1rk4z)
        ├─→ 3a. Status command (orch-go-7ocqx)
        ├─→ 3b. Complete command (orch-go-ec9kh)
        └─→ 3c. Other commands (orch-go-wjf89)
             └─→ 4. Testing (orch-go-h4eza)
```

## Task Breakdown

### 1. Config System (orch-go-5w0fj)

**Files to create/modify:**
- `pkg/config/config.go` - Add `SpawnMode` field
- `cmd/orch/config.go` - Add `config set` subcommand

**Schema:**
```go
type Config struct {
    SpawnMode   string            `yaml:"spawn_mode"`   // "claude" | "opencode"
    Claude      ClaudeConfig      `yaml:"claude,omitempty"`
    OpenCode    OpenCodeConfig    `yaml:"opencode,omitempty"`
    Servers     map[string]int    `yaml:"servers,omitempty"`
}

type ClaudeConfig struct {
    Model       string `yaml:"model"`         // "opus" | "sonnet" | "haiku"
    TmuxSession string `yaml:"tmux_session"`  // e.g., "workers-orch-go"
}

type OpenCodeConfig struct {
    Model  string `yaml:"model"`  // default model for spawns
    Server string `yaml:"server"` // HTTP server URL
}
```

**Default values:**
```yaml
spawn_mode: opencode  # backward compatible
claude:
  model: opus
  tmux_session: workers-orch-go
opencode:
  model: flash
  server: http://localhost:4096
```

**Command:**
```bash
orch config set spawn_mode claude
orch config set spawn_mode opencode
orch config get spawn_mode
```

### 2a. Claude Spawn (orch-go-0z5i4)

**Files to create:**
- `pkg/spawn/claude.go` - Tmux spawn implementation

**Key functions:**
```go
// SpawnClaude creates tmux window and launches claude CLI
func SpawnClaude(cfg SpawnConfig) (AgentInfo, error) {
    // 1. Generate SPAWN_CONTEXT.md
    // 2. Create tmux window in workers session
    // 3. Send keys: cd <workspace> && claude --file SPAWN_CONTEXT.md
    // 4. Return agent info with mode=claude, tmux_window set
}

// MonitorClaude reads tmux pane output
func MonitorClaude(agentID string) (string, error) {
    // tmux capture-pane -p -t <window>
}

// SendClaude sends message to claude instance
func SendClaude(agentID string, message string) error {
    // tmux send-keys -t <window> "<message>" Enter
}

// AbandonClaude kills tmux window
func AbandonClaude(agentID string) error {
    // tmux kill-window -t <window>
}
```

**Tmux workflow:**
```bash
# Create window
tmux new-window -t workers-orch-go: -n "inv-task-abc"

# Send claude command
tmux send-keys -t workers-orch-go:inv-task-abc \
  "cd .orch/workspace/og-inv-task-abc && claude --file SPAWN_CONTEXT.md" Enter

# Monitor output
tmux capture-pane -p -t workers-orch-go:inv-task-abc

# Detect completion (look for /exit or idle)
# Parse for "Phase: Complete" in output
```

### 2c. Docker Spawn (Added Jan 20, 2026)

**Files created:**
- `pkg/spawn/docker.go` - Docker spawn implementation

**Key functions:**
```go
// SpawnDocker creates host tmux window and launches claude in Docker container.
// Provides Statsig fingerprint isolation for rate limit escape hatch.
func SpawnDocker(cfg SpawnConfig) (AgentInfo, error) {
    // 1. Generate SPAWN_CONTEXT.md
    // 2. Create host tmux window in workers session
    // 3. Build Docker command with volume mounts
    // 4. Send docker run command to tmux window
    // 5. Return agent info with mode=docker, tmux_window set
}

// MonitorDocker reads host tmux pane output (same as claude mode)
func MonitorDocker(agentID string) (string, error)

// SendDocker sends keys to host tmux window (same as claude mode)
func SendDocker(agentID string, message string) error

// AbandonDocker kills host tmux window (container auto-removes with --rm)
func AbandonDocker(agentID string) error
```

**Docker workflow:**
```bash
# Create host tmux window
tmux new-window -t workers-orch-go: -n "inv-task-abc"

# Send docker run command
tmux send-keys -t workers-orch-go:inv-task-abc \
  'docker run -it --rm \
    --user "$(id -u):$(id -g)" \
    -v "$HOME":"$HOME" \
    -v "$HOME/.claude-docker":"$HOME/.claude" \
    -w /path/to/project \
    -e HOME="$HOME" \
    -e CLAUDE_CONTEXT=worker \
    -e TERM=xterm-256color \
    claude-code-mcp \
    bash -c "cat SPAWN_CONTEXT.md | claude --dangerously-skip-permissions"' Enter

# Monitor output (same as claude mode)
tmux capture-pane -p -t workers-orch-go:inv-task-abc

# Detect completion (same as claude mode)
# Parse for "Phase: Complete" in output
```

**Key differences from Claude mode:**
- Uses `~/.claude-docker/` instead of `~/.claude/` for fresh fingerprint
- Docker image `claude-code-mcp` must be pre-built
- Container runs with `--rm` flag (auto-cleanup)
- ~2-5s startup overhead per spawn

### 2b. Registry Schema (orch-go-1rk4z)

**Files to modify:**
- `pkg/registry/registry.go` - Add mode tracking

**Schema changes:**
```go
type Agent struct {
    ID          string    `json:"id"`
    BeadsID     string    `json:"beads_id"`
    Mode        string    `json:"mode"`         // "claude" | "opencode" | "docker"
    Status      string    `json:"status"`

    // Mode-specific fields
    SessionID   string    `json:"session_id,omitempty"`    // OpenCode mode
    TmuxWindow  string    `json:"tmux_window,omitempty"`   // Claude mode AND Docker mode

    SpawnedAt   time.Time `json:"spawned_at"`
    // ... rest of fields
}
```

**Note:** Docker mode uses the same `TmuxWindow` field as Claude mode since both use host tmux windows.

**Backward compatibility:**
- Load old registry → default `mode` to "opencode"
- Populate `SessionID` from existing field

### 3a. Status Command (orch-go-7ocqx)

**Files to modify:**
- `cmd/orch/status.go` - Mode-aware routing

**Implementation:**
```go
func getAgentStatus(agent *registry.Agent) (Status, error) {
    switch agent.Mode {
    case "claude":
        return getClaudeStatus(agent)
    case "opencode":
        return getOpenCodeStatus(agent)
    case "docker":
        return getDockerStatus(agent)  // Uses same mechanism as claude
    default:
        return Status{}, fmt.Errorf("unknown mode: %s", agent.Mode)
    }
}

func getClaudeStatus(agent *registry.Agent) (Status, error) {
    // tmux list-windows | grep <window>
    // tmux capture-pane -p | parse for phase
    // Check if window still exists
}

func getOpenCodeStatus(agent *registry.Agent) (Status, error) {
    // HTTP GET /session/<id>
    // Parse response for status/phase
}

func getDockerStatus(agent *registry.Agent) (Status, error) {
    // Same as getClaudeStatus - uses host tmux window
    return getClaudeStatus(agent)
}
```

### 3b. Complete Command (orch-go-ec9kh)

**Files to modify:**
- `cmd/orch/complete.go` - Mode-aware verification

**Implementation:**
```go
func completeAgent(agent *registry.Agent) error {
    switch agent.Mode {
    case "claude":
        return completeClaude(agent)
    case "opencode":
        return completeOpenCode(agent)
    }
}

func completeClaude(agent *registry.Agent) error {
    // 1. Capture tmux pane output
    output := tmux.CapturePaneText(agent.TmuxWindow)

    // 2. Parse for Phase: Complete
    if !strings.Contains(output, "Phase: Complete") {
        return fmt.Errorf("agent not at Complete phase")
    }

    // 3. Verify artifacts exist (SYNTHESIS.md, etc.)
    workspace := getWorkspacePath(agent.ID)
    if err := verifyArtifacts(workspace); err != nil {
        return err
    }

    // 4. Close beads issue
    // 5. Kill tmux window
    // 6. Update registry
}
```

### 3c. Other Commands (orch-go-wjf89)

**Monitor:**
```go
// Claude mode: tmux capture + follow new output
// OpenCode mode: SSE stream

func monitorAgent(agent *registry.Agent) error {
    switch agent.Mode {
    case "claude":
        return monitorClaudeTmux(agent)
    case "opencode":
        return monitorOpenCodeSSE(agent)
    }
}
```

**Send:**
```go
func sendMessage(agent *registry.Agent, msg string) error {
    switch agent.Mode {
    case "claude":
        return tmux.SendKeys(agent.TmuxWindow, msg)
    case "opencode":
        return opencode.SendMessage(agent.SessionID, msg)
    }
}
```

**Abandon:**
```go
func abandonAgent(agent *registry.Agent) error {
    switch agent.Mode {
    case "claude":
        // Kill tmux window, export transcript if possible
        return tmux.KillWindow(agent.TmuxWindow)
    case "opencode":
        // Delete HTTP session, export transcript
        return opencode.DeleteSession(agent.SessionID)
    }
}
```

### 4. Testing (orch-go-h4eza)

**Test scenarios:**

1. **Mode toggle:**
   ```bash
   orch config set spawn_mode claude
   orch spawn investigation "test claude mode"
   # Verify tmux window created

   orch config set spawn_mode opencode
   orch spawn investigation "test opencode mode"
   # Verify HTTP session created
   ```

2. **Mixed registry:**
   ```bash
   # Create agents in different modes
   orch spawn --backend claude investigation "task 1"
   orch spawn --backend opencode investigation "task 2"
   orch spawn --backend docker investigation "task 3"

   # Verify status shows all three
   orch status
   ```

3. **Mode-specific operations:**
   ```bash
   # Test send in all modes
   orch send <claude-agent> "message"
   orch send <opencode-agent> "message"
   orch send <docker-agent> "message"

   # Test complete in all modes
   orch complete <claude-agent>
   orch complete <opencode-agent>
   orch complete <docker-agent>
   ```

4. **Graceful fallback:**
   ```bash
   # Stop opencode server
   pkill -f "opencode serve"

   # Verify opencode agents show "unavailable"
   orch status

   # Verify claude mode still works
   orch spawn --backend claude investigation "test"

   # Verify docker mode still works (independent of OpenCode)
   orch spawn --backend docker investigation "test"
   ```

5. **Docker fingerprint isolation:**
   ```bash
   # Verify fresh fingerprint per spawn
   orch spawn --backend docker investigation "task 1"
   # Check ~/.claude-docker/ is used, not ~/.claude/
   # Verify rate limits don't carry over from host
   ```

## Implementation Notes

### SPAWN_CONTEXT.md Generation

Both modes use the same context generation (existing `pkg/spawn/context.go`). Only the delivery mechanism differs:
- OpenCode: Send as prompt via HTTP
- Claude: Write to file, reference via `--file` flag

### Phase Detection

Both modes need to parse agent output for "Phase: X":
- OpenCode: Already implemented in SSE parsing
- Claude: Parse tmux capture output with same regex

### Workspace Management

Workspace paths remain the same for both modes:
- `.orch/workspace/{name}/`
- `SPAWN_CONTEXT.md`, `SYNTHESIS.md`, etc.

### Tmux Session Auto-Creation

If tmux session doesn't exist, create it:
```bash
if ! tmux has-session -t workers-orch-go 2>/dev/null; then
    tmux new-session -d -s workers-orch-go
fi
```

## Migration Path

**Phase 1: Add config (backward compatible)**
- Default mode = opencode
- Existing workflows unchanged

**Phase 2: Implement claude mode**
- Add `--mode claude` flag
- Test in parallel with opencode

**Phase 3: Switch default (breaking change)**
- Change default to claude
- Document opencode usage for dashboard needs

## Success Criteria

- [x] Config toggle works (`orch config set spawn_mode`)
- [x] Claude mode spawns create tmux windows with `claude` CLI
- [x] OpenCode mode spawns create HTTP sessions (existing behavior)
- [x] Docker mode spawns create host tmux windows with Docker containers
- [x] Status command works with all three backends
- [x] Complete command works with all three backends
- [x] Can switch modes mid-project without breaking registry
- [x] Mixed registry (claude, opencode, docker agents) works correctly
- [x] Graceful fallback when backend unavailable
- [x] Docker provides fresh Statsig fingerprint per spawn

## Cost Impact Summary

| Mode | Monthly Cost | When to Use |
|------|--------------|-------------|
| Claude | $100 (Max only) | Default, budget-constrained, need Opus quality |
| OpenCode | $200-300 (Max + API) | Need dashboard, parallel spawning, specific models |
| Docker | $100 (Max only) | Rate limit escape hatch, fingerprint isolation |

**Docker Prerequisites:**
- Docker installed and running
- Docker image `claude-code-mcp` built from `~/.claude/docker-workaround/`
- Docker socket accessible to current user
