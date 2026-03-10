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
├── main.go              # Entry point, Cobra root command setup
├── *_cmd.go             # Command implementations (spawn, complete, status, etc.)
├── daemon.go            # Daemon autonomous processing
├── serve*.go            # HTTP API server and handlers (agents, beads, system)
├── session.go           # Session management and state
└── shared.go            # Shared utilities and helpers

pkg/
├── opencode/            # OpenCode HTTP client + SSE streaming
│   ├── client.go       # Session/message management via REST API
│   ├── sse.go          # Server-sent events for real-time monitoring
│   └── types.go        # API response types
├── model/               # Model alias resolution
│   └── model.go        # opus→anthropic/claude-opus, flash→google/gemini-2.5-flash
├── account/             # Claude Max account management
│   └── account.go      # Read/write ~/.orch/accounts.yaml, token refresh
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

## Spawn Backends

orch uses two backends for agent spawning, selected automatically via model-aware routing:

### Primary Path: Claude CLI (Tmux)

```bash
# Default — Anthropic models route here automatically
orch spawn --bypass-triage feature-impl "task" --issue ID

# Daemon also uses Claude CLI for Anthropic models
bd create "task" --type task -l triage:ready
orch daemon run
```

**Use for:** All Anthropic model work (opus, sonnet, haiku) — this is the default

**Characteristics:**

- Tmux window (visible progress)
- Independent of OpenCode server (crash-resistant)
- Claude Max 20x subscription (flat rate, unlimited)
- Daemon-managed or manual lifecycle

**History:** Was originally the "escape hatch" (Jan 2026). Became the default backend on Feb 19, 2026 when Anthropic banned subscription OAuth in third-party tools, making Claude CLI the only path for Anthropic models.

### Multi-Model Path: OpenCode API (Headless)

```bash
# For non-Anthropic models only
orch spawn --bypass-triage --model gpt-5 feature-impl "task" --issue ID
```

**Use for:** Non-Anthropic models (Google, OpenAI, DeepSeek)

**Characteristics:**

- Headless (no tmux window), returns immediately
- High concurrency (5+ agents)
- Depends on OpenCode server (localhost:4096)
- Dashboard visibility via SSE
- Pay-per-token pricing

### Architectural Principle: Backend Independence

**Pattern discovered Jan 10, 2026:** When building observability infrastructure, OpenCode server crashed repeatedly (3 times in 1 hour), killing all agents working on the fixes. Claude CLI agents in tmux survived crashes and completed the work.

**General rule:** Critical paths need independent secondary mechanisms. The Claude CLI backend provides this naturally — it doesn't depend on OpenCode server, so infrastructure work is crash-resistant by default.

### Architectural Principle: Pain as Signal

**Pattern discovered Jan 17, 2026:** Autonomous error correction requires agents to "feel" the friction of their own failure in real-time. Passive logs/metrics are insufficient for agent self-healing.

1. **Infrastructure Injection:** System-level sensors (coaching plugins) inject detections (loops, thrashing) directly into the agent's sensory stream.
2. **Pressure over Compensation:** Friction is injected as tool-layer messages, forcing the agent to confront its own degradation rather than relying on human babysitting.

**See:** `.kb/guides/resilient-infrastructure-patterns.md` for implementation patterns

## Accretion Boundaries

**Rule:** Files >1,500 lines require extraction before feature additions. Run `orch hotspot` to check current bloated files. If modifying large files, see `.kb/guides/code-extraction-patterns.md` for extraction workflow.

**Enforcement (three-layer):**
- **Layer 1 — Spawn gates (blocking):** `feature-impl` and `systematic-debugging` skills are blocked from spawning when targeting CRITICAL files (>1,500 lines). Exempt skills: `architect`, `investigation`, `capture-knowledge`, `codebase-audit`. Override: `--force-hotspot --architect-ref <closed-architect-issue>`. Auto-bypass when prior architect review exists.
- **Layer 2 — Daemon escalation:** Daemon routes feature-impl/systematic-debugging to architect when issue targets hotspot files.
- **Layer 3 — Spawn context advisory:** Hotspot info injected into SPAWN_CONTEXT.md for agent awareness.
- **Completion gates (warning):** Warn on additions >50 lines to files >800 lines.
- Decision: `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md`

## Architectural Constraints

### No Local Agent State

orch-go must not maintain local agent state (registries, projection DBs, SSE materializers, caches for agent discovery).
Query beads and OpenCode directly. If queries are slow, fix the authoritative source; do not build a projection.

## Knowledge Base Structure

This project has two knowledge directories:

- **`.kb/`** — Project-level knowledge (models, guides, decisions, investigations specific to orch-go)
- **`.kb/global/`** — Cross-project knowledge (models, guides, decisions shared across all projects)

`~/.kb` is a symlink to `.kb/global/`. The `kb context` CLI searches both automatically.

**When searching for models, guides, or investigations, always check BOTH paths:**
- `.kb/models/` — project models (e.g., spawn-architecture, daemon-autonomous-operation)
- `.kb/global/models/` — cross-project models (e.g., behavioral-grammars, skillc-testing)
- `.kb/guides/` — project guides
- `.kb/global/guides/` — cross-project guides (e.g., meta-orchestrator-mental-models)
- `.kb/decisions/` + `.kb/global/decisions/` — same pattern

**When creating probes for global models:** write to `.kb/global/models/{name}/probes/`, not `.kb/models/`.

## Key References

**Before debugging, check the relevant guide in `.kb/guides/`:**

| Topic                    | Guide                                  | When to Read                                            |
| ------------------------ | -------------------------------------- | ------------------------------------------------------- |
| Agent lifecycle          | `agent-lifecycle.md`                   | Agents not completing, dashboard wrong                  |
| Spawn                    | `spawn.md`                             | Spawn failures, wrong context, flags                    |
| Status/Dashboard         | `status-dashboard.md`                  | Wrong status, dashboard issues                          |
| Beads integration        | `beads-integration.md`                 | bd commands failing, issue tracking                     |
| Skill system             | `skill-system.md`                      | Skill not loading, wrong behavior                       |
| Daemon                   | `daemon.md`                            | Auto-spawn issues, triage workflow                      |
| Resilient infrastructure | `resilient-infrastructure-patterns.md` | Building/fixing critical infrastructure, backend independence |

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
- Conditionally includes server context for UI-focused skills (feature-impl, systematic-debugging)

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
- `daemon run --replace` - Stop existing daemon first, then start (graceful takeover)
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
- **Edit tool + tab indentation**: Svelte files in `web/src/` and Go files use tab indentation. The Read tool's line-number prefix uses a tab delimiter that collides with content tabs, causing Edit tool "String to replace not found" errors. See "Tab-Indented File Editing" section below.
- **OAuth tokens**: Never share refresh tokens between orch (`accounts.yaml`) and Claude CLI (keychain) — rotation invalidates the copy in the other system immediately
- **Account routing**: `accounts.yaml` `config_dir` field is REQUIRED for account routing to work — without it, `CLAUDE_CONFIG_DIR` is never injected
- **Non-Anthropic models**: GPT-4o/GPT-5.2-codex have 67-87% stall rates on protocol-heavy skills (architect, investigation). Use Anthropic models for these.
- **BEADS_DIR**: `BEADS_DIR=~/path/.beads bd close/update/list` enables cross-project beads operations from any directory
- **Skill sources**: Live in `orch-go/skills/src/`, deployed via `skillc deploy` to `~/.claude/skills/`
- **Playwright CLI**: Default for visual verification (1 bash call, ~1s). MCP only for interactive page exploration. On SSE-heavy pages, use `domcontentloaded` + `waitForSelector`, not `networkidle`.

## Tab-Indented File Editing

**Problem:** The Read tool outputs `line_number→[TAB][content]`. When file content also starts with tabs (Svelte, Go, Makefile), adjacent tabs create ambiguity. Agents construct `old_string` with the wrong number of leading tabs, and Edit fails.

**Files affected in this project:** All `.svelte` files in `web/src/` use tab indentation. Go files use tabs per `gofmt` convention.

**Workarounds (in order of preference):**

1. **Include more context lines** in `old_string` — multi-line matches are less ambiguous than single-line matches with leading tabs
2. **Check exact whitespace first:** `head -20 file.svelte | cat -vet` — tabs display as `^I`, making them countable
3. **Use Write tool** for small files (<100 lines) — rewrite the entire file to avoid tab-matching issues
4. **Use sed via Bash** for surgical line edits: `sed -i '' '10s/old/new/' file.svelte`

**Prevention:** Before editing any tab-indented file, verify the indentation with `cat -vet` on the relevant lines. Do not rely solely on the Read tool output to count leading tabs.

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

| Event             | Source                             | Purpose             |
| ----------------- | ---------------------------------- | ------------------- |
| `session.spawned` | `orch spawn`                       | Agent created       |
| `agent.completed` | `orch complete` or `bd close` hook | Agent finished work |
| `agent.abandoned` | `orch abandon`                     | Agent abandoned     |

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

## OpenCode Fork (We Own It)

**OpenCode is NOT a third-party dependency.** Dylan maintains a fork at `~/Documents/personal/opencode` (upstream: `sst/opencode`). This means:

- **Bugs in OpenCode → fix them in the fork**, not "report upstream"
- **Schema changes** in `*.sql.ts` require running `cd packages/opencode && bun drizzle-kit generate` and committing the migration
- **After code changes:** rebuild with `cd ~/Documents/personal/opencode/packages/opencode && bun run build`, then restart via `orch-dashboard restart`
- **Never install opencode-ai from npm** — it shadows the fork

The fork uses SQLite + Drizzle ORM (migrated from JSON file storage). Database at `~/.local/share/opencode/opencode.db`.

## Related

- **OpenCode fork:** `~/Documents/personal/opencode` (we maintain this)
- **Python orch-cli:** `~/Documents/personal/orch-cli` (fallback: `orch-py`)
- **Beads:** Issue tracking via `bd` CLI
- **Orchestrator skill:** `~/.claude/skills/meta/orchestrator/SKILL.md`
- **orch-knowledge:** *(merged into orch-go — skills in `skills/src/`, knowledge in `.kb/`)*
