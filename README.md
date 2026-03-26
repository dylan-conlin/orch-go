# orch-go

A coordination and comprehension layer for multi-model AI agent work.

Most agent tooling treats agent output as disposable: run the agent, get the code change, discard the context. orch-go treats what agents *learn* as a first-class resource. Threads organize questions. Synthesis and briefs make agent work legible without re-reading sessions. Claims, models, and decisions accumulate structured knowledge over time. The system answers three questions every day: **what changed? what was learned? what remains open?**

Execution (spawning agents, routing to models, managing sessions) is necessary substrate, but it is not the center. The center is making agent output compound into durable understanding that survives model changes, client migrations, and context window limits.

## How It Works

### Threads: the organizing spine

Work exists in service of questions, not the other way around. A thread tracks a line of thinking across multiple agent sessions, investigations, and decisions. When an agent finishes, its findings feed back into the thread that motivated the work.

```bash
orch thread new "How does token refresh interact with multi-account routing?"
orch thread list                    # See active threads with linked work
orch thread show <thread-id>        # Thread with evidence, decisions, open questions
```

### Synthesis and briefs: async comprehension

Every agent session that completes produces a brief — a half-page narrative of what happened, what was learned, and what tension remains. Briefs are written for a human reading without context, not for the agent that produced them.

```bash
orch comprehension                  # Pending items in the comprehension queue
orch review done                    # Batch review of completed work
orch review synthesize              # Synthesize patterns across recent completions
```

### Knowledge composition: claims, models, decisions

Agents produce investigations. Investigations generate probes against models (structured claims about how the system works). Probes that confirm or contradict claims update the model. Decisions record resolved branches. The knowledge base grows through use, not through documentation sprints.

```bash
orch kb ask "how does spawn routing work?"   # Query the knowledge base
orch kb claims                               # View tracked claims
orch kb audit                                # Check knowledge health
```

### Verification and routing: trust layer

Completion isn't just "agent said it's done." The verification pipeline checks phase reporting, test evidence, synthesis quality, duplication, and accretion boundaries before closing work. This is how comprehension stays trustworthy.

```bash
orch complete proj-123              # Verify and close (two human gates)
orch complete proj-123 --headless   # Non-interactive (daemon-triggered)
```

## Execution Substrate

The system spawns agents, routes them to models, manages sessions, and monitors progress. This layer is designed to be portable — it should not depend on any single model provider or agent client.

### Spawning agents

```bash
orch spawn feature-impl "implement X" --issue proj-123   # Spawn from issue
orch spawn investigation "how does X work?" --explore     # Parallel decomposition
orch work proj-123 --inline                               # Blocking TUI
```

### Monitoring and lifecycle

```bash
orch status                         # Active agents across projects
orch daemon run                     # Autonomous triage and spawn
orch wait proj-123 --timeout 30m    # Wait for completion
orch clean                          # Clean up finished agents
```

### Backend routing

Agents route to different backends based on model:
- **Claude CLI** (default) — Anthropic models via tmux, crash-resistant
- **OpenCode API** — non-Anthropic models, headless, high concurrency

```bash
orch spawn --model gpt-5 feature-impl "task"   # Routes to OpenCode
orch account switch work                         # Switch accounts
```

## Architecture

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
                │  bd CLI (beads issue tracking)   │
                └─────────────────────────────────┘
```

The core layer is where the differentiated value lives. The substrate is portable and replaceable. Many tools can run agents — few treat agent-produced knowledge as composable.

## Development

```bash
make build      # Build
make test       # Test
make install    # Install to ~/bin/orch
orch version    # Verify version
```

### Project structure

```
cmd/orch/           # CLI commands (Cobra-based)
pkg/                # Library packages
  thread/           # Thread management
  claims/           # Claim tracking
  completion/       # Completion pipeline
  verify/           # Verification gates
  digest/           # Knowledge artifact change digest
  spawn/            # Spawn context and gates
  model/            # Model alias resolution
  daemon/           # Autonomous processing
  opencode/         # OpenCode HTTP client
  ...
.kb/                # Project knowledge base (models, guides, decisions, investigations)
.kb/global/         # Cross-project knowledge (shared models and guides)
skills/src/         # Skill sources (deployed via skillc)
.orch/              # Orchestration state (workspaces, config, templates)
```

## Requirements

- Go 1.21+
- macOS (desktop notifications via beeep)
- `bd` CLI for issue tracking (beads)
- Claude Code CLI or OpenCode server for agent execution

## Further Reading

- `.kb/guides/architecture-overview.md` — full system architecture with core/substrate/adjacent boundary
- `.kb/guides/cli.md` — complete CLI reference
- `.kb/guides/spawn.md` — spawn flow, flags, and context generation
- `.kb/guides/agent-lifecycle.md` — agent states and transitions
- `.kb/guides/daemon.md` — autonomous triage and OODA loop
- `.kb/decisions/2026-03-26-thread-comprehension-layer-is-primary-product.md` — the product boundary decision
