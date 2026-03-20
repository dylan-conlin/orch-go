# Architecture Overview

**Purpose:** System architecture, directory structure, spawn backends, and architectural principles for orch-go.

**Extracted from:** CLAUDE.md (2026-03-20)

---

## System Diagram

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

## Directory Structure

```
cmd/orch/
├── main.go              # Entry point, Cobra root command setup
├── *_cmd.go             # Command implementations (spawn, complete, status, etc.)
├── daemon_commands.go   # Daemon command setup
├── daemon_loop.go       # Daemon OODA poll cycle (Sense/Orient/Decide/Act)
├── daemon_periodic.go   # Daemon periodic tasks (backlog cull, plan advancement)
├── daemon_snapshot.go   # Daemon state snapshots
├── daemon_decision_log.go # Daemon decision logging
├── daemon_handlers.go   # Daemon HTTP handlers (once, preview, reflect)
├── serve*.go            # HTTP API server and handlers (agents, beads, system)
├── harness_*.go         # Harness governance (audit, report, init)
├── plan_cmd.go          # Coordination plan management
├── complete_*.go        # Completion pipeline, duplication detection, post-lifecycle
├── kb*.go               # Knowledge base commands (audit, init, extract, ask, gate, challenge, autolink)
├── stats_*.go           # Stats aggregation (spawn/completion rates, gate effectiveness, skill metrics)
├── session*.go          # Session management (start, end, status, history)
├── spawn_*.go           # Spawn helpers, dry-run preview
├── hotspot*.go          # Hotspot analysis and accretion tracking
├── doctor*.go           # Health checks, diagnostics, defect/migration scans
├── clean_*.go           # Clean subcommands (orphans, sessions, workspaces)
├── review*.go           # Review subcommands (triage, synthesize, orphans, done)
├── daemon_helpers.go    # Daemon utility functions
├── daemon_launchd.go    # Daemon launchd service management
├── lifecycle_adapters.go # Agent lifecycle infrastructure adapters
├── knowledge_maintenance.go # KB maintenance at completion time
├── telemetry.go         # CLI command telemetry tracking
├── tokens.go            # Token usage display
├── attach.go            # Workspace attachment
├── swarm.go             # Batch spawn with concurrency control
├── deploy.go            # Atomic deployment (rebuild, restart, verify)
├── learn.go             # Learning system (suggestions, patterns, effects)
├── servers.go           # Multi-project server management
├── precommit_cmd.go     # Pre-commit checks (accretion, model-stub, duplication)
└── orient_cmd.go        # Session orientation

pkg/
├── opencode/            # OpenCode HTTP client + SSE streaming
├── model/               # Model alias resolution
├── account/             # Claude Max account management
├── tmux/                # Tmux window management
├── spawn/               # Spawn context generation + gates
├── skills/              # Skill discovery and loading
├── verify/              # Completion verification
├── events/              # Event logging (events.jsonl) + skill-level learning metrics
├── notify/              # Desktop notifications
├── daemon/              # Autonomous processing types
├── daemonconfig/        # Daemon configuration (ComplianceConfig, allocation)
├── focus/               # North star tracking
├── question/            # Question extraction
├── dupdetect/           # Duplicate detection for completion pipeline
├── plan/                # Coordination plan types
├── orient/              # Daemon Orient phase (measurement, work graph)
├── attention/           # Attention routing and prioritization
├── beads/               # Beads client utilities
├── beadsutil/           # Beads helper utilities
├── discovery/           # Agent discovery
├── hook/                # Claude Code hook management
├── identity/            # Agent identity resolution
├── kbmetrics/           # Knowledge base metrics (autolink, decision audit, model size, orphan classify)
├── kbgate/              # KB gate enforcement
├── tree/                # File tree generation
├── debrief/             # Session debrief generation
├── artifactsync/        # Artifact drift detection
├── claudemd/            # CLAUDE.md parsing
├── userconfig/          # User configuration management
├── config/              # Project config (.orch/config.yaml)
├── agent/               # Agent lifecycle state machine
├── completion/          # Completion pipeline logic
├── certs/               # TLS certificates for local HTTPS
├── coaching/            # Agent coaching plugins (loop/thrash detection)
├── digest/              # KB artifact change digest (threads, models, investigations)
├── entropy/             # Codebase entropy measurement
├── findingdedup/        # Finding deduplication for KB
├── health/              # Health check infrastructure
├── modeldrift/          # Model drift detection
├── patterns/            # Behavioral pattern detection
├── port/                # Port allocation management
├── service/             # Service lifecycle management
├── session/             # Session tracking (start/end/label)
├── sessions/            # Session history and search
├── state/               # Agent state management
├── thread/              # Thread management
├── timeline/            # Timeline generation
├── urltomd/             # URL-to-markdown conversion
├── workspace/           # Workspace management
├── control/             # Control plane locking
├── checkpoint/          # Session checkpointing
├── activity/            # Activity tracking
├── action/              # Action outcome logging and pattern detection
├── advisor/             # Model recommendation via live API data
├── group/               # Project group resolution (groups.yaml)
├── orch/                # Spawn pipeline, completion, governance, backend routing
├── graph/               # Dependency graph utilities
├── claims/              # Machine-readable claim tracking for KB models (claims.yaml, tension clusters)
└── display/             # Terminal display helpers
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

## Spawn Flow

1. `orch spawn SKILL "task"` invokes spawn command
2. Resolves model alias via `pkg/model.Resolve()`
3. Runs spawn gates (`pkg/spawn/gates/`) — duplication checks, advisory hotspot warnings
4. **With --dry-run:** Shows spawn plan and exits without executing
5. Creates workspace: `.orch/workspace/{name}/`
6. Generates `SPAWN_CONTEXT.md` via `pkg/spawn`
7. **Default (Claude CLI):** Spawns in tmux window via Claude CLI
8. **Non-Anthropic models:** Creates headless session via OpenCode HTTP API
9. **With --explore:** Decomposes into parallel subproblems via exploration orchestrator (investigation/architect only)
10. Returns immediately

Note: `--inline` is available on `orch work`, not `orch spawn`.
