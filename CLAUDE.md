# orch-go

Go rewrite of orch-cli - AI agent orchestration via OpenCode API.

## Architecture Overview

```
orch-go (this binary)
    │
    ├── opencode run --model    ← spawn/Q&A (wraps CLI)
    ├── GET /event              ← SSE monitoring (direct HTTP)
    └── bd CLI                  ← beads integration (shells out)
    │
    ▼
OpenCode Server (http://127.0.0.1:4096)
    │
    ▼
Claude API / Gemini API
```

```
cmd/orch/
├── main.go              # Entry point, Cobra commands, command registration
├── daemon.go            # Daemon command (autonomous processing)
├── resume.go            # Resume command (continue paused agents)
└── wait.go              # Wait command (block until phase reached)

pkg/
├── opencode/            # OpenCode HTTP client + SSE streaming
│   ├── client.go       # Session/message management via REST API
│   ├── sse.go          # Server-sent events for real-time monitoring
│   └── types.go        # API response types
├── model/               # Model alias resolution
│   └── model.go        # opus→anthropic/claude-opus, flash→google/gemini-2.5-flash
├── account/             # Claude Max account management
│   └── account.go      # Read/write ~/.orch/accounts.yaml, token refresh
├── registry/            # Agent state management
│   └── registry.go     # JSON registry with file locking, reconcile with tmux
├── tmux/                # Tmux window management
│   └── tmux.go         # Create windows, send keys, capture output
├── spawn/               # Spawn context generation
│   ├── config.go       # SpawnConfig struct
│   └── context.go      # SPAWN_CONTEXT.md template generation
├── skills/              # Skill discovery and loading
│   └── loader.go       # Parse ~/.claude/skills/{category}/{skill}/SKILL.md
├── verify/              # Completion verification
│   ├── check.go        # Verify deliverables, phase, commits
│   └── update.go       # Update beads issue on completion
├── events/              # Event logging
│   └── logger.go       # Append to ~/.orch/events.jsonl
├── notify/              # Desktop notifications
│   └── notify.go       # macOS notification center integration
├── daemon/              # Autonomous processing
│   └── daemon.go       # Poll bd ready, spawn triage:ready issues
├── focus/               # North star tracking
│   └── focus.go        # ~/.orch/focus.json for cross-project prioritization
└── question/            # Question extraction
    └── question.go     # Parse pending questions from agent output
```

## Key Packages

### cmd/orch/main.go (Entry Point)
- Uses Cobra framework for CLI structure
- All commands defined inline (spawn, status, complete, send, etc.)
- Global `--server` flag for OpenCode URL
- Subcommand groups: `account`, `daemon`

### pkg/opencode/ (OpenCode Client)
- `Client` struct with HTTP methods for OpenCode REST API
- `ListSessions()`, `GetSession()`, `CreateSession()`, `GetMessages()`
- `SSEClient` for real-time event streaming
- Session status polling for completion detection
- `ExtractRecentText()` for extracting text from message history

### pkg/model/ (Model Resolution)
- `Resolve(spec)` maps aliases to full provider/model format
- Aliases: `opus`, `sonnet`, `haiku` (Anthropic), `flash`, `pro` (Gemini)
- Supports `provider/model` format passthrough
- Default: `anthropic/claude-opus-4-5-20251101`

### pkg/account/ (Account Management)
- `LoadConfig()` reads `~/.orch/accounts.yaml`
- `Switch(name)` refreshes OAuth tokens and updates OpenCode auth
- Token sources: OpenCode auth file, macOS Keychain
- Same config format as Python orch-cli for interop

### pkg/registry/ (Agent State)
- `Registry` struct managing `~/.orch/agent-registry.json`
- File locking via `syscall.Flock`
- `Reconcile()`: Syncs registry with tmux window state
- Agent states: `active`, `completed`, `abandoned`

### pkg/spawn/ (Spawn Context)
- `SpawnConfig` struct with all spawn parameters
- `GenerateContext()` creates SPAWN_CONTEXT.md content
- Embeds skill content, task description, beads issue context
- Sets deliverables paths for verification
- Conditionally includes server context for UI-focused skills (feature-impl, systematic-debugging, reliability-testing)

### pkg/config/ (Project Config)
- `Load()` reads `.orch/config.yaml` from project directory
- `Config.Servers` maps service names to ports (e.g., `web: 5173`)
- Used by `orch servers` commands and spawn context generation

### pkg/verify/ (Completion Verification)
- `Check()` validates agent work before closing
- Verifies: Phase Complete, deliverables exist, commits present
- `Update()` closes beads issue with completion reason

## Spawn Flow

1. `orch spawn SKILL "task"` invokes spawn command in main.go
2. Resolves model alias via `pkg/model.Resolve()`
3. Creates workspace: `.orch/workspace/{name}/`
4. Generates `SPAWN_CONTEXT.md` via `pkg/spawn`
5. **Default (headless):** Creates session via HTTP API, sends prompt
6. **With --tmux:** Creates session + tmux window for monitoring (opt-in)
7. **With --inline:** Runs OpenCode TUI in current terminal (blocking)
8. Registers agent in `~/.orch/agent-registry.json`
9. Returns immediately (unless --inline)

## Commands

### Agent Lifecycle
- `spawn <skill> "task"` - Create agent with skill context
- `status` - List active agents
- `send <session-id> "message"` - Q&A on existing session
- `complete <agent-id>` - Verify and close agent work
- `abandon <agent-id>` - Mark stuck agent as abandoned
- `clean` - Remove completed agents from registry

### Monitoring
- `monitor` - Real-time SSE event watching
- `wait <agent-id>` - Block until phase reached
- `tail <agent-id>` - Capture recent tmux output (requires `--tmux` spawn)
- `question <agent-id>` - Extract pending question
- `serve` - HTTP API server for web UI (port 3348)

### Account & Model
- `account list` - Show saved Claude Max accounts
- `account switch <name>` - Switch to different account
- `account remove <name>` - Remove saved account
- `usage` - Show Claude Max usage (delegates to Python)

### Automation
- `work <issue-id>` - Spawn from beads issue with skill inference
- `daemon run` - Run autonomous processing in foreground
- `daemon preview` - Show what would be spawned

### Server Management
- `servers list` - Show all projects with port allocations and running status
- `servers start <project>` - Start servers via tmuxinator
- `servers stop <project>` - Stop servers for a project
- `servers attach <project>` - Attach to servers window
- `servers open <project>` - Open web port in browser
- `servers status` - Show summary view (running/stopped counts)

## Development

```bash
# Build
make build

# Test
make test

# Install to ~/bin/orch
make install

# Verify version
orch version
```

### Adding New Commands

1. Add command to `cmd/orch/main.go` (inline with Cobra)
2. Or create `cmd/orch/{name}.go` for complex commands
3. Add to `rootCmd.AddCommand()` in init()

### Adding New Packages

1. Create `pkg/{name}/{name}.go`
2. Create `pkg/{name}/{name}_test.go`
3. Import in cmd/orch as needed

## Gotchas

- **File locking**: Registry uses `syscall.Flock` - Unix only
- **Window targeting**: Use workspace name, not window index
- **Model default**: Opus (Max subscription), not Gemini (pay-per-token)
- **SSE parsing**: Event type is inside JSON data, not `event:` prefix
- **Beads integration**: Shells out to `bd` CLI, doesn't use API directly
- **OpenCode auth**: Reads from `~/.local/share/opencode/auth.json`

## Common Commands

```bash
# Spawn with specific model (headless by default)
orch spawn --model flash investigation "explore X"

# Spawn in tmux window (opt-in for visual monitoring)
orch spawn --tmux investigation "explore X"

# Run inline with TUI (blocking)
orch spawn --inline investigation "explore X"

# Switch accounts when rate-limited
orch account switch work

# Spawn from beads issue
orch spawn feature-impl "implement X" --issue proj-123

# Wait for agent to complete
orch wait proj-123 --timeout 30m

# Complete and verify agent work
orch complete proj-123

# Clean up finished agents
orch clean
```

## Related

- **Python orch-cli:** `~/Documents/personal/orch-cli` (fallback: `orch-py`)
- **Beads:** Issue tracking via `bd` CLI
- **Orchestrator skill:** `~/.claude/skills/policy/orchestrator/SKILL.md`
- **orch-knowledge:** Skill sources, decisions, investigations
