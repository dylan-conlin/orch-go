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

## Forked Dependencies

orch-go depends on Dylan's forks of two upstream projects. Changes to these components must be made in the fork repos, not upstream.

| Dependency | Fork Location | Upstream | Why Forked |
|------------|--------------|----------|------------|
| **OpenCode** | `~/Documents/personal/opencode` | `sst/opencode` | Instance eviction (LRU/TTL), SSE cleanup, memory leak fixes, custom plugins |
| **Beads (bd)** | `~/Documents/personal/beads` | `steveyegge/beads` | Sandbox detection, JSONL-only default, rapid-restart prevention, SQLite corruption fixes |

**Push targets:**
- OpenCode: `git push fork dev` (remote `fork` → `dylan-conlin/opencode`)
- Beads: check remote config (`git remote -v`)

**Rebuild after fork changes:**
- OpenCode: `orch-dashboard restart` auto-detects newer commits and rebuilds
- Beads: `cd ~/Documents/personal/beads && go install ./cmd/bd/`

**Key context:**
- OpenCode fork stays diverged from upstream (custom features needed for orch-go)
- Beads fork stays diverged from upstream — see `.kb/decisions/2026-02-05-beads-fork-stay-diverged.md`
- Both forks have reliability fixes not present upstream (instance eviction, WAL corruption prevention)

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
- Agent state tracking in SQLite (dashboard visibility)
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

- When using a second Max account (clean fingerprint isolation)
- Request-rate throttling (per-device limits, NOT weekly usage quota)
- Fresh "device" identity to Anthropic

**Characteristics:**

- Host tmux window running Docker container
- Fresh Statsig fingerprint per spawn via `~/.claude-docker/`
- Auto-mounts real configs (CLAUDE.md, settings.json, skills/, hooks/) read-only
- ~2-5s container startup overhead
- Requires Docker image `claude-code-mcp` (built from `~/.claude/docker-workaround/`)

**Important:** Weekly usage quota (e.g., "97% used") is **account-level**, not device-level. Docker fingerprint isolation does NOT bypass usage quota - only helps with request-rate throttling.

**Constraint:** Docker spawns must be initiated from macOS host, not from inside a Claude agent sandbox. The sandbox (Linux container) doesn't have docker installed. Daemon-driven docker spawns work when `orch daemon run` executes on the host. Agent-to-agent docker spawns fail.

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

| Topic                    | Guide                                  | When to Read                                            |
| ------------------------ | -------------------------------------- | ------------------------------------------------------- |
| Agent lifecycle          | `agent-lifecycle.md`                   | Agents not completing, dashboard wrong                  |
| Spawn                    | `spawn.md`                             | Spawn failures, wrong context, flags                    |
| Status/Dashboard         | `status-dashboard.md`                  | Wrong status, dashboard issues                          |
| Beads integration        | `beads-integration.md`                 | bd commands failing, issue tracking                     |
| Skill system             | `skill-system.md`                      | Skill not loading, wrong behavior                       |
| Daemon                   | `daemon.md`                            | Auto-spawn issues, triage workflow                      |
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

**Service ports:** OpenCode (4096), orch serve (3348)

**Dashboard URL:** https://localhost:3348

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
2. **Decision gate:** Checks `.kb/decisions/` for conflicts (see below)
3. Resolves model alias via `pkg/model.Resolve()`
4. Creates workspace: `.orch/workspace/{name}/`
5. Generates `SPAWN_CONTEXT.md` via `pkg/spawn`
6. **Default (headless):** Creates session via HTTP API, sends prompt
7. **With --tmux:** Creates session + tmux window for monitoring (opt-in)
8. **With --inline:** Runs OpenCode TUI in current terminal (blocking)
9. Returns immediately (unless --inline)

## Decision Gate

Architect decisions can block spawns via the decision gate. This gives decisions teeth - you can't accidentally spawn investigation #19 when a decision says "stop tactical fixes."

**How it works:**

1. Decisions declare blocked keywords in YAML frontmatter:
   ```yaml
   ---
   blocks:
     - keywords: ['coaching plugin', 'worker detection']
   ---
   ```
2. `orch spawn` checks if task matches any blocked keywords
3. If match found → spawn blocked with warning
4. Override with `--acknowledge-decision <decision-id>` (logged)

**Reference:** `.kb/decisions/2026-01-28-decision-gate.md`

## Commands

### Agent Lifecycle

- `spawn <skill> "task"` - Create agent with skill context
- `status` - List active agents
- `send <session-id> "message"` - Q&A on existing session
- `complete <agent-id>` - Verify and close agent work
- `abandon <agent-id>` - Mark stuck agent as abandoned
- `clean` - Remove completed agent workspaces and stale state

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
# Build (outputs to build/orch)
make build

# Test
make test

# Install to ~/bin/orch (REQUIRED for orch-dashboard to use fresh binary)
# This creates a symlink ~/bin/orch -> build/orch
make install

# Verify version
orch version
```

**IMPORTANT:** Always use `make install` (not `go build ./cmd/orch`) to ensure the binary is built to the correct location. The `go build` command outputs to `./orch-go` by default, which won't be used by `orch-dashboard restart`.

### Adding New Commands

1. Add command to `cmd/orch/main.go` (inline with Cobra)
2. Or create `cmd/orch/{name}.go` for complex commands
3. Add to `rootCmd.AddCommand()` in init()

### Adding New Packages

1. Create `pkg/{name}/{name}.go`
2. Create `pkg/{name}/{name}_test.go`
3. Import in cmd/orch as needed

## Process Lifecycle & Zombie Prevention

Headless agent spawns (the default) use OpenCode's HTTP API. OpenCode's server spawns the bun process internally — `orch` never gets a process handle. This means:

- No `.process_id` file is written for headless agents
- The process ledger (`~/.orch/process-ledger.jsonl`) is empty for headless spawns
- When agents finish, the bun process keeps running (OpenCode doesn't terminate it on session deletion)

**Without cleanup, these zombie processes accumulate, exhaust RAM, thrash swap, and crash macOS WindowServer (breaking mouse input, requiring reboot).** This has happened 3+ times.

### Defenses (layered)

1. **`orch reap`** — standalone reaper command, works without the daemon
   - Queries OpenCode API for active sessions, kills unmatched bun processes
   - `--force` kills ALL agent processes (use after crash/reboot)
   - `--dry-run` to preview

2. **Launchd agent** — runs `orch reap` every 5 minutes automatically
   - Plist: `~/Library/LaunchAgents/com.orch.reap.plist`
   - Logs: `~/.orch/logs/reap.log`
   - Install: `./scripts/install-reaper.sh`

3. **`orch complete` sweep** — after deleting a session, sweeps for newly-orphaned bun processes and kills them

4. **Daemon orphan reaper** — if `orch daemon run` is active, it reaps orphans every 5 minutes via `ReapOrphanProcesses()`

5. **`orch clean --processes`** — manual cleanup flag on the clean command

### Emergency: System Seizing Up

If system gets sluggish or mouse stops working:

```bash
# Kill all zombie agent processes immediately
orch reap --force

# Or if orch isn't responding:
pkill -f 'bun.*src/index.ts' 

# Then check memory
vm_stat | head -5
```

### Key Files

- Orphan detection: `pkg/process/orphans.go` — finds bun+src/index.ts processes, excludes server
- Reap command: `cmd/orch/reap_cmd.go`
- Complete sweep: `cmd/orch/complete_cleanup.go` (`sweepOrphanedProcessesAfterSessionDelete`)
- Daemon reaper: `pkg/daemon/orphan_reaper.go`
- Process ledger: `pkg/process/ledger.go` (only populated for non-headless spawns)
- Launchd plist: `scripts/com.orch.reap.plist`

## Gotchas

- **Window targeting**: Use workspace name, not window index
- **Model default**: Opus (Max subscription), not Gemini (pay-per-token)
- **SSE parsing**: Event type is inside JSON data, not `event:` prefix
- **Beads integration**: Shells out to `bd` CLI, doesn't use API directly
- **OpenCode auth**: Reads from `~/.local/share/opencode/auth.json`
- **Build with make, not go build**: Use `make install` to embed `sourceDir` via ldflags. Direct `go build` leaves `sourceDir: unknown`, breaking `orch serve` (can't find certs).
- **Claude Code sandbox**: Agents run in Linux sandbox, not macOS. See `.kb/guides/claude-code-sandbox-architecture.md` for implications.

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

# Cross-project daemon (polls all kb-registered projects)
orch daemon run --cross-project           # Run daemon polling all projects
orch daemon preview --cross-project       # Preview what would spawn across projects
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

## Question Subtyping

Questions in beads can be tagged with subtypes to indicate their resolvability and authority requirements. The subtyping convention enables the daemon to identify which questions can be auto-spawned (factual) and which require orchestrator synthesis (judgment) or Dylan reframing (framing).

### Subtype Convention

Use labels with the format `subtype:{factual|judgment|framing}`:

| Subtype            | Meaning                         | Who Resolves               | Example                                          |
| ------------------ | ------------------------------- | -------------------------- | ------------------------------------------------ |
| `subtype:factual`  | "How does X work?"              | Daemon (via investigation) | "How does the escalation model work?"            |
| `subtype:judgment` | "Should we use X or Y?"         | Orchestrator               | "Should we refactor auth before adding feature?" |
| `subtype:framing`  | "Is X even the right question?" | Dylan                      | "Is the current abstraction even correct?"       |

### Usage Examples

```bash
# Create a factual question (daemon-spawnable)
bd create "How does the spawn system work?" --type question -l subtype:factual

# Create a judgment question (orchestrator synthesis)
bd create "Should we use event sourcing for this?" --type question -l subtype:judgment

# Create a framing question (Dylan reframes)
bd create "Is this the right problem to solve?" --type question -l subtype:framing

# Query factual questions ready for daemon
bd ready --type question --label subtype:factual

# Query judgment questions for orchestrator review
bd ready --type question --label subtype:judgment
```

### Design Rationale

- **Zero schema changes:** Uses existing beads label infrastructure
- **Flexible:** Questions can evolve subtypes during resolution (factual → framing escalation)
- **Authority-aware:** Enables daemon to auto-spawn factual questions while deferring judgment/framing to humans
- **Convention over enforcement:** Matches beads design philosophy

**Reference:** See `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md` for full decision context and `.kb/models/decidability-graph.md` for the conceptual model.

## Label Taxonomy

Issues use a 3-prefix label taxonomy for dynamic grouping. Labels enable multi-dimensional views (edges handle dependencies).

| Prefix    | Purpose         | Valid Values                                      |
| --------- | --------------- | ------------------------------------------------- |
| `area:`   | Work domain     | dashboard, spawn, beads, cli, skill, kb, opencode |
| `effort:` | Size estimation | small, medium, large                              |
| `status:` | Meta-status     | parked, blocked-external, needs-review            |

**Existing labels** (unchanged): `triage:ready`, `triage:review`, `subtype:*`, `authority:*`

**Usage:**

```bash
bd create "Add rate limiting" --type feature -l area:cli -l effort:medium -l triage:ready
bd list --label area:dashboard
```

**Grouping vs Dependencies:** Use `area:` labels for grouping related work. Use `blocks`/`depends_on` edges for real dependencies. Do NOT create epics for grouping — epics conflate "these belong together" with "this blocks that." Existing epics were migrated to `area:` labels and closed (Feb 2026).

**Reference:** `.kb/investigations/2026-02-05-inv-design-label-based-issue-grouping.md` for full taxonomy details.

## Related

- **Python orch-cli:** `~/Documents/personal/orch-cli` (fallback: `orch-py`)
- **Beads:** Issue tracking via `bd` CLI
- **Orchestrator skill:** `~/.claude/skills/meta/orchestrator/SKILL.md`
- **orch-knowledge:** Skill sources, decisions, investigations
