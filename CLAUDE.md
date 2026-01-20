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

## Tool Restrictions

### Task Tool Disabled

**The Task tool is globally disabled in this project via `.opencode/opencode.json`.**

```json
{
  "permission": {
    "task": "deny"
  }
}
```

**Why:** Orchestrators were using Task tool to spawn subagents instead of using `orch spawn`. This bypasses:
- The spawn context system (skills, beads integration, workspace setup)
- Agent registry tracking (dashboard visibility)
- Completion verification workflow
- Event tracking for stats

**Correct delegation pattern:** Use `orch spawn` (or the Bash tool to invoke it) to delegate work to other agents.

**Reference:** `.kb/investigations/2026-01-20-research-disable-task-tool-opencode-orchestrator.md`

## Triple Spawn Modes: Resilience by Design

orch supports three spawn modes for redundancy:

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

### Double Escape Hatch (Docker + Claude CLI)
```bash
orch spawn --bypass-triage --backend docker feature-impl "task" --issue ID
```
**Use for:**
- When host fingerprint is rate-limited and you have a second Max account
- Clean account isolation (avoids Statsig contamination between Max accounts)
- Fresh "device" identity to Anthropic

**Characteristics:**
- Host tmux window running Docker container
- Fresh Statsig fingerprint per spawn via `~/.claude-docker/`
- Independent of host rate limits
- ~2-5s container startup overhead
- Requires Docker image `claude-code-mcp` (built from `~/.claude/docker-workaround/`)

### Architectural Principle: Critical Paths Need Escape Hatches

**Pattern discovered Jan 10, 2026:** When building observability infrastructure, OpenCode server crashed repeatedly (3 times in 1 hour), killing all agents working on the fixes. Switched to `--mode claude --tmux` for critical agents (orch doctor, overmind supervision, dashboard integration), which survived crashes and completed the work.

**General rule:** When infrastructure can fail, critical paths need independent secondary paths that:
1. **Don't depend on what failed** (claude CLI ≠ opencode server)
2. **Provide visibility** (tmux vs headless)
3. **Can complete the work** (opus for quality)

### Architectural Principle: Pain as Signal

**Pattern discovered Jan 17, 2026:** Autonomous error correction requires agents to "feel" the friction of their own failure in real-time. Passive logs/metrics are insufficient for agent self-healing.
1. **Infrastructure Injection:** System-level sensors (coaching plugins) inject detections (loops, thrashing) directly into the agent's sensory stream.
2. **Pressure over Compensation:** Friction is injected as tool-layer messages, forcing the agent to confront its own degradation rather than relying on human babysitting.

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

**Always use `orch-dashboard` script** - handles orphan cleanup, stale sockets, and proper startup:

```bash
orch-dashboard start    # Start all services (kills orphans first)
orch-dashboard stop     # Stop all services
orch-dashboard restart  # Full restart with cleanup
orch-dashboard status   # Check service status
orch-dashboard logs     # View service logs (overmind echo)
```

**Service ports:** OpenCode (4096), orch serve (3348), Web UI (5188)

**Dashboard URL:** http://localhost:5188

**Why not raw overmind?** Direct `overmind start` can fail silently when orphan processes hold ports or stale sockets exist. The `orch-dashboard` script handles these edge cases.

**Production:** Future VPS deployment will use systemd. See `.kb/decisions/2026-01-10-dev-vs-prod-architecture.md`.

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
- Default: `opus` (Claude Max subscription)

### pkg/account/ (Account Management)
- `LoadConfig()` reads `~/.orch/accounts.yaml`
- `Switch(name)` refreshes OAuth tokens and updates OpenCode auth
- Token sources: OpenCode auth file, macOS Keychain
- Same config format as Python orch-cli for interop

### pkg/spawn/ (Spawn Context)
- `SpawnConfig` struct with all spawn parameters
- `GenerateContext()` creates SPAWN_CONTEXT.md content
- `docker.go` - Docker backend spawn implementation (fresh Statsig fingerprint)
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

# Use Docker escape hatch for fresh fingerprint (rate limit bypass)
orch spawn --backend docker investigation "explore X"

# Use Claude escape hatch for infrastructure work
orch spawn --backend claude --tmux --model opus feature-impl "fix spawn"

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
