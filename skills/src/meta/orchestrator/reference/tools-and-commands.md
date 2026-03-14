# Tools & Commands Reference

> **Note:** This is reference material extracted from the orchestrator skill.
> The compiled skill contains the most-used commands inline.
> Consult this file for full CLI help, config locations, and daemon internals.
> Generated from CLI --help output on 2026-03-05.

## Tool Ecosystem

```
orch        → agent coordination     (spawn, monitor, complete, daemon)
beads (bd)  → what work needs doing  (issues, dependencies, tracking)
kb          → knowledge management   (investigations, decisions, quick entries)
skillc      → skill compilation      (modular skills → SKILL.md)
opencode    → agent execution        (Claude frontend, session management)
```

**Cross-repo architecture:** See `~/.orch/ECOSYSTEM.md`

## Search Tool Selection

| Question | Tool |
|----------|------|
| "What do we know about X?" | `kb context "X"` |
| "Find all mentions of X in .kb/" | `kb search "X"` or Grep |
| "Find X in code" | Grep on `pkg/` `cmd/` |

## Config Locations

- Orch: `~/.orch/config.yaml` | Accounts: `~/.orch/accounts.yaml`
- Daemon: `~/Library/LaunchAgents/com.orch.daemon.plist`
- OpenCode: `{project}/opencode.json`
- Plugins: `.opencode/plugin/` (project) or `~/.config/opencode/plugin/` (global)

---

## orch CLI

### All Commands

| Command | Description |
|---------|-------------|
| `spawn` | Spawn a new agent with skill context |
| `work` | Start work on a beads issue with skill inference |
| `complete` | Verify and close agent work |
| `abandon` | Abandon a stuck agent |
| `status` | Show swarm status and active agents |
| `send` | Send a message to an existing session |
| `tail` | Capture recent output from an agent |
| `wait` | Block until agent reaches specified phase |
| `resume` | Resume a paused agent |
| `rework` | Spawn a rework agent for a completed issue |
| `review` | Review agent work before completing |
| `swarm` | Batch spawn multiple agents |
| `clean` | Clean up stale workspaces and tmux windows |
| `daemon` | Autonomous overnight processing |
| `account` | Manage Claude Max accounts |
| `focus` | Set or view north star priority |
| `orient` | Session start orientation |
| `hotspot` | Detect areas needing architect intervention |
| `session` | Manage orchestrator work sessions |
| `monitor` | Real-time SSE event watching |
| `question` | Extract pending question from agent |
| `serve` | HTTP API server for dashboard |
| `servers` | Manage dev servers across projects |
| `deploy` | Atomic deployment: rebuild, restart, verify |
| `stats` | Show aggregated agent statistics |
| `history` | Agent history with skill usage analytics |
| `emit` | Emit an event to events.jsonl |
| `config` | Config-as-code management |
| `init` | Initialize orch scaffolding |
| `doctor` | Check health of orch services |
| `harness` | Control plane immutability (init, check, lock, unlock, audit, report) |
| `control` | Lock/unlock control plane files (macOS chflags) |
| `plan` | Coordination plan management (show, status, create) |
| `thread` | Living threads for mid-session comprehension capture |
| `audit` | Randomized completion audit selection |
| `backlog` | Backlog maintenance (cull stale P3/P4) |
| `patterns` | Surface behavioral patterns for orchestrator awareness |
| `hook` | Test, validate, and trace Claude Code hooks |
| `settings` | Programmatic settings.json modification (add-hook, remove-hook) |
| `tokens` | Show token usage for sessions |
| `retries` | Show issues with retry patterns |
| `port` | Manage port allocations across projects |
| `kb` | Knowledge base commands (extract, ask, claims, orphans, audit) |
| `debrief` | Generate session debrief with auto-populated sections |

### orch spawn

Spawn a new agent with skill context. Manual spawn requires `--bypass-triage`.

**Key flags:**

| Flag | Description |
|------|-------------|
| `--bypass-triage` | Required for manual spawns (friction gate) |
| `--issue <id>` | Beads issue ID for tracking |
| `--model <alias>` | Model alias (opus, sonnet, haiku, flash, pro) |
| `--backend <mode>` | claude (tmux, default) or opencode (HTTP API) |
| `--tmux` | Run in tmux window (opt-in visual monitoring) |
| `--phases <phases>` | Feature-impl phases (e.g., implementation,validation) |
| `--mode <mode>` | tdd, direct, or verification-first |
| `--validation <level>` | none, tests, smoke-test |
| `--light` / `--full` | Spawn tier (skip/require SYNTHESIS.md) |
| `--no-track` | Opt-out of beads issue tracking |
| `--mcp <preset>` | MCP server preset (e.g., playwright) |
| `--workdir <dir>` | Target project directory |
| `--max-agents <n>` | Concurrent agent limit (default 5) |
| `--max-turns <n>` | Max agentic turns (0 = unlimited) |
| `--effort <level>` | Claude CLI effort (low, medium, high) |
| `--gate-on-gap` | Block spawn on poor context quality |
| `--dry-run` | Show spawn plan without executing |
| `--intent <type>` | experience, produce, compare, investigate, fix, build, explore |
| `--account <name>` | Account for Claude CLI spawns |
| `--explore` | Exploration mode: decompose → fan out → judge → synthesize (investigation/architect only) |
| `--explore-breadth <n>` | Max parallel subproblem workers (default 3, max 10) |
| `--explore-depth <n>` | Max judge re-exploration iterations (default 1, max 5) |
| `--explore-judge-model <alias>` | Model for judge agent (cross-model judging, e.g., sonnet when workers use opus) |
| `--force-hotspot` | Bypass hotspot gate (requires `--architect-ref`) |
| `--auto-init` | Auto-initialize .orch/.beads if missing |

**Examples:**
```bash
# Preferred: daemon-driven
bd create "investigate auth" --type investigation -l triage:ready

# Manual spawn
orch spawn --bypass-triage investigation "explore codebase"
orch spawn --bypass-triage --issue proj-123 feature-impl "implement feature"
orch spawn --bypass-triage --tmux --model opus investigation "deep analysis"
orch spawn --bypass-triage --no-track investigation "exploratory work"
orch spawn --bypass-triage --mcp playwright feature-impl "UI feature"

# Exploration mode (multi-angle parallel investigation)
orch spawn --bypass-triage --explore investigation "how does the spawn pipeline work?"
orch spawn --bypass-triage --explore --explore-breadth 5 architect "design new completion flow"
```

### orch complete

Verify and close agent work. Two human gates: explain-back (gate1) and behavioral verification (gate2).

**Verification gates:** phase_complete, synthesis, test_evidence, visual_verification, git_diff, build, constraint, phase_gate, skill_output, explain_back, verified.

**Tier-aware verification:**
- Tier 1 (feature/bug): `--explain` + `--verified` required
- Tier 2 (investigation/probe): `--explain` only
- Tier 3 (task/question): No checkpoint required

**Key flags:**

| Flag | Description |
|------|-------------|
| `--explain <text>` | What was built and why (gate1) |
| `--verified` | Behavioral verification confirmed (gate2) |
| `--approve` | Approve visual changes for UI tasks |
| `--skip-<gate>` | Skip specific gate (requires `--skip-reason`) |
| `--skip-reason <text>` | Justification for skip (min 10 chars) |
| `--no-archive` | Skip workspace archival |
| `--workdir <dir>` | Cross-project completion |

**Examples:**
```bash
orch complete proj-123 --explain "Built JWT auth" --verified
orch complete proj-123 --approve --explain "Added dark mode" --verified
orch complete proj-123 --skip-test-evidence --skip-reason "Tests run in CI"
```

### orch abandon

Abandon a stuck agent. Issue stays open for retry.

**Flags:** `--reason <text>` (generates FAILURE_REPORT.md), `--force` (override recent activity check), `--workdir <dir>`.

### orch status

Show swarm status. Default: compact (running agents only). `--all` for full view. `--json` for scripting. `--project <name>` to filter.

### orch work

Start work on a beads issue with skill inference from issue type (bug→systematic-debugging, feature/task→feature-impl, investigation→investigation).

**Flags:** `--inline` (blocking TUI), `--model <alias>`, `--workdir <dir>`.

### orch send

Send a message to existing session. Accepts session ID, beads ID, or workspace name.

### orch tail

Capture recent output. `--lines N` (default 50).

### orch wait

Block until phase reached. `--phase <name>` (default Complete), `--timeout <duration>` (default 30m), `--interval <seconds>` (default 5).

### orch clean

Cleanup stale resources. Default: report-only (dry-run).

**Flags:** `--workspaces`, `--sessions`, `--orphans`, `--ghosts`, `--all`, `--workspace-days N` (default 7), `--archived-ttl N` (default 30), `--preserve-orchestrator`.

### orch resume

Resume a paused agent. Accepts beads ID, `--workspace <name>`, or `--session <id>`.

### orch rework

Spawn rework agent with structured context from prior attempt. Requires `--bypass-triage`.

**Flags:** `--model <alias>`, `--skill <name>`, `--tmux`, `--mode <mode>`, `--workdir <dir>`.

### orch review

Review agent work before completing. Default shows actionable pending completions.

**Flags:** `--all`, `--stale`, `--needs` (failures only), `-p <project>`, `-l <limit>`.

**Subcommands:** `review done <project>`, `review orphans`, `review triage`.

### orch swarm

Batch spawn with concurrency control.

**Flags:** `--issues <ids>`, `--ready` (from bd ready queue), `-c <concurrency>` (default 3), `--detach`, `--dry-run`, `--model <alias>`.

### orch focus

Set or view north star priority. `orch focus "goal"` to set, `orch focus` to view, `orch focus clear` to clear.

### orch hotspot

Detect areas needing architect intervention. Signals: fix-density, investigation clustering, bloat size.

**Flags:** `--days N` (default 28), `--threshold N` (fix commits, default 5), `--inv-threshold N` (default 3), `--bloat-threshold N` (default 800), `--json`.

### orch orient

Session start orientation: throughput, previous session, ready work, plans, threads, models.

**Flags:** `--days N` (default 1), `--json`, `--hook`, `--skip-ready`.

### orch session

Manage orchestrator work sessions.

**Subcommands:** `start <goal>`, `status`, `end`, `resume`, `label <name>`, `validate`.

### orch daemon

Autonomous processing.

**Subcommands:** `run` (foreground), `run --replace` (graceful takeover), `preview` (dry-run).

### orch account

Manage Claude Max accounts.

**Subcommands:** `list`, `switch <name>`, `remove <name>`.

### orch servers

Manage dev servers across projects.

**Subcommands:** `list`, `start <project>`, `stop <project>`, `attach <project>`, `open <project>`, `status`.

---

## bd (beads) CLI

### Core Commands

| Command | Description |
|---------|-------------|
| `create` | Create a new issue |
| `close` | Close one or more issues |
| `update` | Update issue fields |
| `list` | List issues with filters |
| `show` | Show issue details |
| `ready` | Show ready work (no blockers) |
| `blocked` | Show blocked issues |
| `search` | Search issues by text |
| `comments` | View/manage comments |
| `dep` | Manage dependencies |
| `label` | Manage labels |
| `sync` | Synchronize with git remote |
| `status` / `stats` | Database overview and statistics |

### bd create

```bash
bd create "title" [flags]
```

**Key flags:**

| Flag | Description |
|------|-------------|
| `-t, --type <type>` | bug, feature, task, epic, chore, question, investigation (default: task) |
| `-p, --priority <n>` | 0-4 or P0-P4 (0=critical, default: 2) |
| `-l, --labels <list>` | Comma-separated labels |
| `-a, --assignee <name>` | Assignee |
| `-d, --description <text>` | Issue description |
| `-e, --estimate <mins>` | Time estimate in minutes |
| `--parent <id>` | Parent issue ID |
| `--deps <list>` | Dependencies (e.g., discovered-from:bd-20,blocks:bd-15) |
| `--silent` | Output only the issue ID |
| `--ephemeral` | Create as ephemeral (not exported) |

### bd close

```bash
bd close <id> [id2...] [flags]
```

**Flags:** `-r, --reason <text>`, `-o, --outcome <type>` (completed, duplicate, wont-fix, invalid), `-f, --force`, `--suggest-next`.

### bd update

```bash
bd update <id> [flags]
```

**Key flags:** `-s, --status <status>`, `--title <text>`, `-p, --priority <n>`, `-a, --assignee <name>`, `-d, --description <text>`, `--add-label <label>`, `--remove-label <label>`.

### bd list

```bash
bd list [flags]
```

**Key flags:** `-s, --status <status>`, `-t, --type <type>`, `-l, --label <labels>`, `-a, --assignee <name>`, `-p, --priority <n>`, `-n, --limit <n>` (default 50), `--sort <field>`, `--all` (include closed), `--pretty` (tree format).

### bd show

```bash
bd show <id> [flags]
```

**Flags:** `--short`, `--thread`.

### bd ready

```bash
bd ready [flags]
```

**Key flags:** `-n, --limit <n>` (default 10), `-p, --priority <n>`, `-t, --type <type>`, `-u, --unassigned`, `-s, --sort <policy>` (hybrid, priority, oldest), `--mol <id>` (filter to molecule steps).

### bd comments

```bash
bd comments <id>              # List comments
bd comments add <id> "text"   # Add comment
bd comments add <id> -f file  # Add from file
```

### bd dep

```bash
bd dep add <issue> <depends-on>    # Add dependency
bd dep remove <issue> <depends-on> # Remove dependency
bd dep list <issue>                # List dependencies
bd dep tree <issue>                # Show dependency tree
bd dep relate <a> <b>              # Bidirectional link
bd dep cycles                      # Detect cycles
```

### bd label

```bash
bd label add <id> <label>          # Add label
bd label remove <id> <label>       # Remove label
bd label list <id>                 # List labels for issue
bd label list-all                  # All unique labels
```

### bd sync

```bash
bd sync                  # Full sync (export, commit, pull, import, push)
bd sync --status         # Show diff between branches
bd sync --squash         # Accumulate without committing
bd sync --flush-only     # Export to JSONL only
bd sync --import-only    # Import from JSONL only
```

### bd search

```bash
bd search "query" [flags]
```

**Key flags:** `-s, --status`, `-t, --type`, `-l, --label`, `-a, --assignee`, `--sort <field>`, `-n, --limit` (default 50).

---

## kb CLI

### All Commands

| Command | Description |
|---------|-------------|
| `context` | Get unified context from entries and artifacts |
| `quick` | Quick entries (decisions, attempts, constraints, questions) |
| `create` | Create investigation, decision, guide, plan, research, specification |
| `search` | Search knowledge artifacts |
| `list` | List artifacts |
| `reflect` | Surface patterns requiring attention |
| `archive` | Archive synthesized investigations |
| `supersede` | Mark artifact as superseded |
| `promote` | Promote quick entry to decision |
| `chronicle` | Show temporal narrative of a topic |
| `index` | Output concise artifact index |
| `learn` | Review knowledge gap suggestions |

### kb context

Unified context search across entries and artifacts.

```bash
kb context "query"                    # Search current project
kb context "query" --domain spawn     # Boost domain matches
kb context "query" --siblings         # Include sibling projects
kb context "query" --global           # All known projects
kb context "query" --global --all     # Bypass group filtering
```

### kb quick

Quick knowledge entries.

```bash
kb quick decide "X" --reason "Y"       # Record decision
kb quick tried "X" --failed "Y"        # Record failed attempt
kb quick constrain "X" --reason "Y"    # Record constraint
kb quick question "X"                  # Record open question
kb quick list                          # List entries
kb quick resolve <id>                  # Resolve a question
kb quick supersede <id>                # Supersede an entry
```

### kb create

Create knowledge artifacts from templates.

```bash
kb create investigation "topic" --model <model-name>  # or --orphan
kb create decision --title "choice"
kb create guide --title "pattern"
kb create plan --title "strategy"
kb create research --title "topic"
kb create specification --title "spec"
```

### kb search

```bash
kb search "query"                     # Search current project
kb search "query" --global            # All projects
kb search "query" --type decisions    # Filter by type
kb search "query" --titles-only       # Titles only
```

---

## skillc CLI

### All Commands

| Command | Description |
|---------|-------------|
| `build` | Compile .skillc/ to SKILL.md or CLAUDE.md |
| `deploy` | Build and deploy to target directory |
| `check` | Validate without building |
| `stats` | Show token metrics |
| `watch` | Watch for changes and auto-rebuild |
| `lint` | Static analysis (5 rules) |
| `test` | Run behavioral scenarios |
| `compare` | Compare test results |
| `verify` | Verify skill output constraints |
| `init` | Initialize new .skillc/ |
| `prime` | Output context injection |
| `doctor` | Self-diagnosis |

### Key Commands

```bash
skillc build                              # Build current .skillc/
skillc build --global                     # Build ~/.claude/.skillc/ → CLAUDE.md
skillc build --recursive                  # Build all .skillc/ dirs
skillc deploy --target ~/.claude/skills/  # Deploy all skills
skillc check                              # Validate (budget, checksum, links)
skillc stats                              # Token metrics
skillc lint                               # Static analysis
```

---

## Daemon Behavior

- **30s grace period:** Issues with `triage:ready` wait 30s before becoming spawnable
- **Duplicate prevention:** Checks processed-issues.jsonl, active sessions, Phase: Complete
- **Concurrency cap 5:** Round-robin fairness across projects; focus-aware priority boost
- **Idle agent expiry:** Stale idle agents age out of capacity gates (1h)
- **Self-check invariants:** Pauses on violation threshold, resumes when clear
- **Auto-complete:** capture-knowledge agents auto-completed on Phase: Complete
- **Stuck detection:** >2h without phase updates triggers notification

### Known Daemon Pitfalls

- **JSONL lock pileup:** Daemon polling creates new `bd` processes faster than you can kill hung ones. Escape: `pkill -9` + `rm lock file` + pause daemon before restarting beads operations.
- **Duplicate spawns:** Daemon has no prior-art check — it spawns for `triage:ready` issues even when prior agents already completed the work. Evidence: 2 of 5 harness issues were already done by prior agents; daemon spawned new agents that found existing work and re-did it.
- **InferSkillFromDescription false positives:** Research keywords ('compare', 'evaluate') are too broad — 100% false-positive rate (1604 inferences, 0 actual research spawns). Keywords match ambient vocabulary in non-research issues.

## Attention Signals

- **UNBLOCKED:** dependencies resolved; issue is actionable
- **STUCK:** runtime >2h with low/no activity; intervene
- **VERIFY FAILED:** auto-complete verification failed; rerun `orch complete <id>`

## System Maintenance

**Skill editing:** Edit `.skillc/` source files, then `skillc deploy`. Never edit SKILL.md directly.

**CLI audit requirement:** After any command/flag additions or removals in orch-go, audit the orchestrator skill against `orch --help` output. Prior audit found 13 stale references including 7 harmful (non-existent commands/flags). Skill staleness propagates to every orchestrator session.

**Tool name verification:** Claude Code tool names can be renamed silently between versions — `--disallowedTools` with invalid names fails silently (no warning). After Claude Code updates, verify tool names in `pkg/spawn/claude.go` against current CLI output. (Evidence: Task→Agent rename broke orchestrator Agent tool blocking.)

**Model external validation:** Models and publications must survive external adversarial review (e.g., Codex or equivalent) before being treated as validated or novel. Internal probes and falsifiability checks are necessary but insufficient — two models passed all internal validation but were immediately challenged on external review.

**Daemon operations:**
```bash
launchctl list | grep orch                               # Status
launchctl kickstart -k gui/$(id -u)/com.orch.daemon      # Restart
tail -f ~/.orch/daemon.log                               # Logs
```
