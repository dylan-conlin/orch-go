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
OpenCode Server (http://localhost:4096)
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

## Dual Spawn Modes: Resilience by Design

orch supports two spawn modes for redundancy:

### Primary Path (Daemon + OpenCode API)
```bash
bd create "task" --type task -l triage:ready
orch daemon run  # Auto-spawns via opencode API, headless
```
**Use for:** Normal workflow, high concurrency, batch processing

**Characteristics:**
- Headless (no tmux window)
- High concurrency (5+ agents)
- Depends on OpenCode server (localhost:4096)
- Daemon-managed lifecycle

### Escape Hatch (Manual + Claude CLI)
```bash
orch spawn --bypass-triage --mode claude --model opus --tmux feature-impl "task" --issue ID
```
**Use for:**
- 🔥 Building infrastructure the primary path depends on
- 🔧 Debugging when OpenCode server is unstable
- 🎯 Critical work that can't afford to lose progress to crashes
- 👁️ Work requiring visual monitoring

**Characteristics:**
- Tmux window (visible progress)
- Independent of OpenCode server
- Crash-resistant (agents survive service restarts)
- Manual lifecycle management

### Architectural Principle: Critical Paths Need Escape Hatches

**Pattern discovered Jan 10, 2026:** When building observability infrastructure, OpenCode server crashed repeatedly (3 times in 1 hour), killing all agents working on the fixes. Switched to `--mode claude --tmux` for critical agents (orch doctor, overmind supervision, dashboard integration), which survived crashes and completed the work.

**General rule:** When infrastructure can fail, critical paths need independent secondary paths that:
1. **Don't depend on what failed** (claude CLI ≠ opencode server)
2. **Provide visibility** (tmux vs headless)
3. **Can complete the work** (opus for quality)

**See:** `.kb/guides/resilient-infrastructure-patterns.md` for implementation patterns

## Key References

**Before debugging, check the relevant guide in `.kb/guides/`:**

| Topic | Guide | When to Read |
|-------|-------|--------------|
| Agent lifecycle | `agent-lifecycle.md` | Agents not completing, dashboard wrong |
| Spawn | `spawn.md` | Spawn failures, wrong context, flags |
| Status/Dashboard | `status-dashboard.md` | Wrong status, dashboard issues |
| Beads integration | `beads-integration.md` | bd commands failing, issue tracking |
| Skill system | `skill-system.md` | Skill not loading, wrong behavior |
| Daemon | `daemon.md` | Auto-spawn issues, triage workflow |
| Resilient infrastructure | `resilient-infrastructure-patterns.md` | Building/fixing critical infrastructure, escape hatches |

These guides synthesize 280+ investigations into authoritative references. Created Jan 4, 2026 after repeatedly re-investigating documented problems.

## Dashboard Server Management

**Changed Jan 10, 2026:** Individual launchd plists for auto-restart reliability. Overmind available for dev workflow but NOT supervised by launchd. See `.kb/decisions/2026-01-10-individual-launchd-services.md` for rationale.

### Architecture (Production)

Each service runs directly under launchd supervision for auto-restart on crash:

```
launchd (macOS native supervisor)
├── com.opencode.serve → opencode serve --port 4096
├── com.orch.serve → orch serve
├── com.orch.web → ~/.orch/start-web.sh (bun run dev)
└── com.orch.doctor → orch doctor --daemon (monitoring)
```

### Service Ports

| Service | Port | Plist | Purpose |
|---------|------|-------|---------|
| OpenCode | 4096 | `com.opencode.serve.plist` | Claude/Gemini API sessions |
| orch serve | 3348 | `com.orch.serve.plist` | Dashboard backend API |
| Web UI | 5188 | `com.orch.web.plist` | Dashboard frontend (Vite) |
| orch doctor | N/A | `com.orch.doctor.plist` | Self-healing daemon |

### launchd Plists

All plists located at `~/Library/LaunchAgents/`:

**OpenCode:**
```xml
com.opencode.serve.plist
  ProgramArguments: /Users/dylanconlin/.bun/bin/opencode serve --port 4096
  WorkingDirectory: ~/Documents/personal/orch-go
  KeepAlive: true (auto-restart on crash)
  RunAtLoad: true (start at login)
```

**orch serve:**
```xml
com.orch.serve.plist
  ProgramArguments: /Users/dylanconlin/bin/orch serve
  WorkingDirectory: ~/Documents/personal/orch-go
  KeepAlive: true
  RunAtLoad: true
```

**Web UI:**
```xml
com.orch.web.plist
  ProgramArguments: /Users/dylanconlin/.orch/start-web.sh
  (Script uses real bun path: /opt/homebrew/bin/bun)
  KeepAlive: true
  RunAtLoad: true
```

**orch doctor (self-healing):**
```xml
com.orch.doctor.plist
  ProgramArguments: /Users/dylanconlin/bin/orch doctor --daemon
  KeepAlive: true (daemon itself supervised)
  Polls services every 30s, kills orphans
```

### Management Commands

**Check service status:**
```bash
orch doctor  # Health check all services
launchctl list | grep -E "opencode|orch"  # launchd status
```

**Restart individual services:**
```bash
launchctl kickstart -k gui/$(id -u)/com.opencode.serve
launchctl kickstart -k gui/$(id -u)/com.orch.serve
launchctl kickstart -k gui/$(id -u)/com.orch.web
```

**Atomic deployment (rebuild + restart all):**
```bash
orch deploy  # Builds, kills orphans, restarts services, verifies health
```

**View logs:**
```bash
orch logs server  # orch serve logs
orch logs daemon  # orch doctor logs
tail -f ~/.orch/opencode-stdout.log  # OpenCode logs
tail -f ~/.orch/web-stdout.log  # Web UI logs
```

### Crash Recovery (Verified)

All services tested and confirmed auto-restart:

**OpenCode** - Killed PID 33029 → launchd restarted as PID 17103 within 5s ✓
**orch serve** - Killed PID 80517 → launchd restarted as PID 18822 within 5s ✓
**Web UI** - Killed PID 18822 → launchd restarted as PID 34128 within 5s ✓

**Testing date:** 2026-01-10 09:04
**Success criteria:** Zero "did you restart?" for 1 week (tracking begins 2026-01-10 09:05)

### Overmind (Development Workflow Only)

Overmind is still available for **dev workflow** but NOT supervised by launchd:

**Why use overmind:**
- Unified start/stop for development: `overmind start`
- View unified logs with color coding: `overmind echo`
- Restart individual services: `overmind restart api`

**Why NOT supervised by launchd:**
- tmux PATH propagation issues
- Circular dependency (need overmind for restart, need launchd for overmind, launchd can't run overmind)
- Adds complexity layer without reliability benefit

**Procfile** (for overmind dev workflow):
```procfile
api: orch serve
web: cd web && bun run dev
opencode: ~/.bun/bin/opencode serve --port 4096
```

**Dev workflow commands:**
```bash
overmind start -D  # Start in daemon mode
overmind status  # Check what's running
overmind restart  # Restart all
overmind quit  # Stop all
```

### What This Architecture Provides

1. ✅ **Auto-restart on crash** - launchd KeepAlive for all services
2. ✅ **Auto-start at login** - launchd RunAtLoad
3. ✅ **Self-healing** - orch doctor daemon kills orphans
4. ✅ **Atomic deployment** - `orch deploy` for rebuild + restart + verify
5. ✅ **Observability** - `orch logs`, health checks, event streaming
6. ✅ **Cache invalidation** - X-Orch-Version headers prevent stale UI
7. ✅ **No tmux dependency** - Each service runs directly
8. ✅ **Simple, direct** - No overmind + launchd + tmux PATH complexity

### Troubleshooting

**Services not starting:**
```bash
orch doctor --fix  # Attempts to restart failed services
launchctl list | grep -E "opencode|orch"  # Check exit codes
tail -f ~/.orch/*-stderr.log  # View error logs
```

**Port conflicts:**
```bash
lsof -ti:4096 | xargs kill -9  # Kill orphan on 4096
lsof -ti:3348 | xargs kill -9  # Kill orphan on 3348
lsof -ti:5188 | xargs kill -9  # Kill orphan on 5188
launchctl kickstart -k gui/$(id -u)/com.opencode.serve  # Restart
```

**Reload plists after changes:**
```bash
launchctl unload ~/Library/LaunchAgents/com.opencode.serve.plist
launchctl load ~/Library/LaunchAgents/com.opencode.serve.plist
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
- Default: `google/gemini-3-flash-preview` (Opus restricted to Claude Code as of Jan 2026)
- Aliases: `opus`, `sonnet`, `haiku` (Anthropic), `flash`, `pro` (Gemini)
- Default: `google/gemini-3-flash-preview` (Opus restricted to Claude Code as of Jan 2026)
- Aliases: `opus`, `sonnet`, `haiku` (Anthropic), `flash`, `pro` (Gemini)
- Default: `google/gemini-3-flash-preview` (Opus restricted to Claude Code as of Jan 2026)

### pkg/account/ (Account Management)
- `LoadConfig()` reads `~/.orch/accounts.yaml`
- `Switch(name)` refreshes OAuth tokens and updates OpenCode auth
- Token sources: OpenCode auth file, macOS Keychain
- Same config format as Python orch-cli for interop

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
8. Returns immediately (unless --inline)

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

## Event Tracking

Agent lifecycle events are logged to `~/.orch/events.jsonl` for stats aggregation.

### Event Types

| Event | Source | Purpose |
|-------|--------|---------|
| `session.spawned` | `orch spawn` | Agent created |
| `agent.completed` | `orch complete` or `bd close` hook | Agent finished work |
| `agent.abandoned` | `orch abandon` | Agent abandoned |

### Beads Close Hook

When issues are closed directly via `bd close` (bypassing `orch complete`), the beads hook at `.beads/hooks/on_close` emits an `agent.completed` event. This closes the tracking gap.

**Hook location:** `.beads/hooks/on_close` (project-specific)

**Manual event emission:**
```bash
# Emit completion event directly (used by hooks)
orch emit agent.completed --beads-id proj-123 --reason "Closed via bd close"
```

**To enable in a project:**
1. Create `.beads/hooks/on_close` (executable)
2. Copy content from orch-go's hook as a template

## Related

- **Python orch-cli:** `~/Documents/personal/orch-cli` (fallback: `orch-py`)
- **Beads:** Issue tracking via `bd` CLI
- **Orchestrator skill:** `~/.claude/skills/meta/orchestrator/SKILL.md`
- **orch-knowledge:** Skill sources, decisions, investigations
