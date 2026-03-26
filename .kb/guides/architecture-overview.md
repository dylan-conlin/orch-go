# Architecture Overview

**Purpose:** System architecture, product boundary, directory structure, and operational principles for orch-go.

**Extracted from:** CLAUDE.md (2026-03-20)
**Revised:** 2026-03-26 — reframed around core/substrate/adjacent boundary per `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md`

---

## What orch-go Is

orch-go is a **coordination and comprehension layer** for multi-model agent work. It turns agent output into durable, legible understanding.

The system has three layers:

1. **Core** — threads, synthesis, knowledge composition, and the human surfaces that answer: *what changed? what was learned? what remains open?*
2. **Substrate** — spawn plumbing, backend routing, session transport, and execution observability that make agents run
3. **Adjacent** — coordination research, benchmarking, and experimental harnesses that inform the methodology but don't define the product

Many tools can run agents. Few systems treat agent-produced knowledge as a first-class, composable resource. That is the center of this project.

---

## Product Boundary

### Core (primary investment, defines identity)

| Concern | What it does | Key packages / commands |
|---------|-------------|----------------------|
| Thread graph | Organizing artifact — work exists in service of questions | `pkg/thread/`, `orch thread` |
| Synthesis & briefs | Async comprehension surface for human review | `complete_*.go`, `serve_briefs.go`, `pkg/completion/` |
| Knowledge composition | Claims, models, decisions, structured accumulation | `pkg/claims/`, `kb*.go`, `pkg/kbmetrics/`, `pkg/kbgate/` |
| Comprehension queue | What needs reading, what has been processed | `review*.go`, `pkg/digest/` |
| Routing & verification | Trust layer — completion gates, duplication detection, verification | `pkg/verify/`, `pkg/dupdetect/`, `complete_*.go` |
| Human control surfaces | Legibility of learning, uncertainty, and change | `serve*.go`, `orient_cmd.go`, `stats_*.go` |

**Deletion criterion:** Does this make the system better at turning agent output into durable, legible understanding? If yes, likely core.

### Substrate (necessary, not identity-defining)

| Concern | What it does | Key packages / commands |
|---------|-------------|----------------------|
| Spawn plumbing | Agent creation, context generation, gates | `pkg/spawn/`, `pkg/orch/`, `spawn_*.go` |
| Backend routing | Model-aware dispatch to Claude CLI / OpenCode / others | `pkg/model/`, `pkg/opencode/` |
| Session transport | Tmux windows, headless sessions, SSE monitoring | `pkg/tmux/`, `pkg/opencode/` |
| Daemon | Autonomous triage, OODA polling, periodic maintenance | `daemon_*.go`, `pkg/daemon/`, `pkg/daemonconfig/` |
| Account management | Multi-account routing, OAuth | `pkg/account/` |
| Execution observability | Dashboard, agent state, activity tracking | `pkg/discovery/`, `pkg/state/`, `pkg/activity/` |
| Beads integration | Issue tracking shell-out to `bd` CLI | `pkg/beads/`, `pkg/beadsutil/` |

The system should be open and portable at this layer. Dependency on any one execution path is a risk, not a moat.

### Adjacent (valuable, separable from core identity)

| Concern | What it does | Location |
|---------|-------------|----------|
| Coordination research | Empirical experiments on multi-agent patterns | `experiments/` |
| Model benchmarking | Provider comparison, pass-rate measurement | `pkg/advisor/`, benchmark commands |
| Platform migration studies | OpenClaw, provider strategy investigations | `.kb/investigations/` |

---

## System Diagram

```
                    ┌─────────────────────────────────┐
                    │           CORE                   │
                    │                                  │
                    │  threads ─ synthesis ─ briefs    │
                    │  claims ─ models ─ decisions     │
                    │  comprehension queue ─ review    │
                    │  verification ─ routing          │
                    │                                  │
                    │   "what changed? what was        │
                    │    learned? what remains open?"   │
                    └──────────────┬───────────────────┘
                                  │
                    ┌─────────────┴───────────────────┐
                    │         SUBSTRATE                │
                    │                                  │
                    │  spawn ─ daemon ─ backends       │
                    │  tmux ─ opencode ─ accounts      │
                    │  beads ─ execution observability  │
                    └──────────────┬───────────────────┘
                                  │
                    ┌─────────────┴───────────────────┐
                    │       EXTERNAL SERVICES          │
                    │                                  │
                    │  Claude API / OpenAI / Gemini    │
                    │  OpenCode Server (:4096)         │
                    │  bd CLI (beads)                  │
                    └─────────────────────────────────┘
```

---

## Directory Structure

```
cmd/orch/
├── main.go              # Entry point, Cobra root command setup
│
│  ── Core ──
├── complete_*.go        # Completion pipeline, synthesis, duplication detection, post-lifecycle
├── kb*.go               # Knowledge base (audit, init, extract, ask, gate, challenge, autolink)
├── review*.go           # Review subcommands (triage, synthesize, orphans, done)
├── serve_briefs.go      # Brief serving for comprehension queue
├── serve_verification.go # Verification serving
├── plan_cmd.go          # Coordination plan management
├── orient_cmd.go        # Session orientation
├── stats_*.go           # Stats aggregation (spawn/completion rates, gate effectiveness)
│
│  ── Substrate ──
├── *_cmd.go             # Command implementations (spawn, complete, status, etc.)
├── daemon_commands.go   # Daemon command setup
├── daemon_loop.go       # Daemon OODA poll cycle (Sense/Orient/Decide/Act)
├── daemon_periodic.go   # Daemon periodic tasks (backlog cull, plan advancement)
├── daemon_snapshot.go   # Daemon state snapshots
├── daemon_decision_log.go # Daemon decision logging
├── daemon_handlers.go   # Daemon HTTP handlers (once, preview, reflect)
├── daemon_helpers.go    # Daemon utility functions
├── daemon_launchd.go    # Daemon launchd service management
├── serve*.go            # HTTP API server and handlers (agents, beads, system)
├── harness_*.go         # Harness governance (audit, report, init)
├── spawn_*.go           # Spawn helpers, dry-run preview
├── session*.go          # Session management (start, end, status, history)
├── lifecycle_adapters.go # Agent lifecycle infrastructure adapters
├── hotspot*.go          # Hotspot analysis and accretion tracking
├── doctor*.go           # Health checks, diagnostics, defect/migration scans
├── clean_*.go           # Clean subcommands (orphans, sessions, workspaces)
├── precommit_cmd.go     # Pre-commit checks (accretion, model-stub, duplication)
├── knowledge_maintenance.go # KB maintenance at completion time
├── telemetry.go         # CLI command telemetry tracking
├── tokens.go            # Token usage display
├── attach.go            # Workspace attachment
├── swarm.go             # Batch spawn with concurrency control
├── deploy.go            # Atomic deployment (rebuild, restart, verify)
├── learn.go             # Learning system (suggestions, patterns, effects)
├── servers.go           # Multi-project server management
└── display helpers, etc.

pkg/
│
│  ── Core ──
├── thread/              # Thread management
├── claims/              # Machine-readable claim tracking (claims.yaml, tension clusters)
├── completion/          # Completion pipeline logic
├── verify/              # Completion verification
├── dupdetect/           # Duplicate detection for completion pipeline
├── kbmetrics/           # KB metrics (autolink, decision audit, model size, orphan classify)
├── kbgate/              # KB gate enforcement
├── digest/              # KB artifact change digest (threads, models, investigations)
├── findingdedup/        # Finding deduplication for KB
├── modeldrift/          # Model drift detection
├── question/            # Question extraction
├── debrief/             # Session debrief generation
├── focus/               # North star tracking
│
│  ── Substrate ──
├── opencode/            # OpenCode HTTP client + SSE streaming
├── model/               # Model alias resolution
├── account/             # Claude Max account management
├── tmux/                # Tmux window management
├── spawn/               # Spawn context generation + gates
├── orch/                # Spawn pipeline, completion, governance, backend routing
├── skills/              # Skill discovery and loading
├── events/              # Event logging (events.jsonl) + skill-level learning metrics
├── notify/              # Desktop notifications
├── daemon/              # Autonomous processing types
├── daemonconfig/        # Daemon configuration (ComplianceConfig, allocation)
├── plan/                # Coordination plan types
├── orient/              # Daemon Orient phase (measurement, work graph)
├── attention/           # Attention routing and prioritization
├── beads/               # Beads client utilities
├── beadsutil/           # Beads helper utilities
├── discovery/           # Agent discovery
├── hook/                # Claude Code hook management
├── identity/            # Agent identity resolution
├── agent/               # Agent lifecycle state machine
├── state/               # Agent state management
├── coaching/            # Agent coaching plugins (loop/thrash detection)
├── patterns/            # Behavioral pattern detection
├── artifactsync/        # Artifact drift detection
├── claudemd/            # CLAUDE.md parsing
├── userconfig/          # User configuration management
├── config/              # Project config (.orch/config.yaml)
├── certs/               # TLS certificates for local HTTPS
├── health/              # Health check infrastructure
├── port/                # Port allocation management
├── service/             # Service lifecycle management
├── session/             # Session tracking (start/end/label)
├── sessions/            # Session history and search
├── tree/                # File tree generation
├── timeline/            # Timeline generation
├── urltomd/             # URL-to-markdown conversion
├── workspace/           # Workspace management
├── control/             # Control plane locking
├── checkpoint/          # Session checkpointing
├── activity/            # Activity tracking
├── action/              # Action outcome logging and pattern detection
├── advisor/             # Model recommendation via live API data
├── group/               # Project group resolution (groups.yaml)
├── graph/               # Dependency graph utilities
├── entropy/             # Codebase entropy measurement
├── display/             # Terminal display helpers
└── ...
```

---

## Execution Substrate Detail

The sections below describe the execution substrate — spawn backends, flow, and architectural principles that keep the system running. These are important for operational work but are not the product's center of gravity.

### Spawn Backends

orch uses two backends for agent spawning, selected automatically via model-aware routing:

#### Primary Path: Claude CLI (Tmux)

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

#### Multi-Model Path: OpenCode API (Headless)

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

This principle extends to the product boundary: the core layer (threads, comprehension, knowledge) should not depend on any single execution backend. Portability at the substrate layer protects the core.

### Architectural Principle: Pain as Signal

**Pattern discovered Jan 17, 2026:** Autonomous error correction requires agents to "feel" the friction of their own failure in real-time. Passive logs/metrics are insufficient for agent self-healing.

1. **Infrastructure Injection:** System-level sensors (coaching plugins) inject detections (loops, thrashing) directly into the agent's sensory stream.
2. **Pressure over Compensation:** Friction is injected as tool-layer messages, forcing the agent to confront its own degradation rather than relying on human babysitting.

**See:** `.kb/guides/resilient-infrastructure-patterns.md` for implementation patterns

### Spawn Flow

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
