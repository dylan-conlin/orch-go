# orch CLI Reference

**Purpose:** Single authoritative reference for the orch-go CLI. Read this before debugging CLI issues or adding new commands.

**Last verified:** Mar 24, 2026

**Supersedes:** 16 CLI investigations from Dec 19 - Jan 4 (see History section)

---

## Identity

orch-go is **"kubectl for AI agents"** - a command-line tool for managing AI agent lifecycle:
- **Spawning:** Launch agents with structured context
- **Monitoring:** Track progress in real-time  
- **Coordination:** Manage multiple agents working together
- **Completion:** Verify agent work and clean up

This identity was established on Nov 29, 2025 and has remained stable through 793+ commits across Python (prototype) and Go (production) implementations.

---

## Command Categories

### Lifecycle Commands

| Command | Purpose | Common Flags |
|---------|---------|--------------|
| `orch spawn` | Create new agent session | `--issue`, `--model`, `--backend`, `--light`, `--full`, `--dry-run` |
| `orch work` | Start work on beads issue with skill inference | `--inline` |
| `orch complete` | Verify and close agent work | `--force`, `--reason` |
| `orch abandon` | Abandon stuck agents | `--force`, `--reason`, `--workdir` |
| `orch reject` | Reject agent work quality and reopen issue | - |
| `orch rework` | Spawn a rework agent for a completed issue | - |
| `orch clean` | Clean up stale resources | `--workspaces`, `--sessions`, `--all` |
| `orch wait` | Block until agent reaches phase | - |

### Monitoring Commands

| Command | Purpose | Common Flags |
|---------|---------|--------------|
| `orch status` | List active agents | `--json`, `--project` |
| `orch monitor` | SSE event monitoring | - |
| `orch tail` | Capture agent output | - |
| `orch question` | Extract pending question | - |
| `orch review` | Batch completion review | `--needs-review` |
| `orch retries` | Show issues with retry patterns | - |

### Strategic Commands

| Command | Purpose | Common Flags |
|---------|---------|--------------|
| `orch focus` | Set/view priority goal | `--json` |
| `orch drift` | Check alignment with focus | `--json` |
| `orch next` | Suggest next action | `--json` |
| `orch orient` | Session start orientation with throughput baseline | - |

### Daemon Commands

| Command | Purpose |
|---------|---------|
| `orch daemon run` | Run work daemon |
| `orch daemon preview` | Preview what would spawn |

### Knowledge & Session Commands

| Command | Purpose |
|---------|---------|
| `orch kb` | Knowledge base inline queries and artifact management |
| `orch debrief` | Generate session debrief with auto-populated sections |
| `orch thread` | Living threads — mid-session comprehension capture |
| `orch comprehension` | Manage comprehension queue (pending review items) |
| `orch decisions` | Decision lifecycle management (staleness, budgets) |
| `orch plan` | Coordination plan management |
| `orch stats` | Show aggregated agent statistics from events.jsonl |

### Infrastructure Commands

| Command | Purpose |
|---------|---------|
| `orch harness` | Harness measurement (audit, report, init) |
| `orch control` | Manage control plane immutability |
| `orch hook` | Test, validate, and trace Claude Code hooks |
| `orch settings` | Modify ~/.claude/settings.json programmatically |
| `orch audit` | Randomized completion audit selection |
| `orch init` | Initialize orch scaffolding in current directory |
| `orch opsec` | Manage OPSEC proxy infrastructure (install, uninstall, start, stop, status, test) |
| `orch port` | Manage port allocations for projects |

### Agent Operations Commands

| Command | Purpose |
|---------|---------|
| `orch attach` | Attach to existing OpenCode session via workspace lookup |
| `orch resume` | Resume a paused agent with continuation prompt |
| `orch reconcile` | Detect in_progress issues with no active agent |
| `orch handoff` | Generate session handoff document |
| `orch swarm` | Spawn multiple agents in parallel with concurrency control |

### Analysis & Reporting Commands

| Command | Purpose |
|---------|---------|
| `orch hotspot` | Analyze git history for areas needing architect attention |
| `orch dupdetect` | Scan Go source for structurally similar functions (AST) |
| `orch entropy` | Aggregate signals from git, events, bloat, duplication |
| `orch health` | Display composite harness health score (0-100) |
| `orch history` | Show agent history with skill usage analytics |
| `orch patterns` | Surface behavioral patterns for orchestrator awareness |
| `orch tokens` | Show detailed token usage for sessions |
| `orch transcript` | Convert OpenCode JSON export to readable markdown |
| `orch tree` | Display knowledge lineage tree visualization |
| `orch sync` | Analyze recent drift events for stale artifacts |
| `orch docs` | Manage CLI documentation debt tracking (list, mark, unmark) |
| `orch session-history` | Search and list OpenCode session history |

### Configuration & System Commands

| Command | Purpose |
|---------|---------|
| `orch config` | Get/set project configuration values |
| `orch model` | Model resolution and management |
| `orch mode` | Switch between development and operations mode |
| `orch servers` | Show server status across all projects |
| `orch doctor` | Diagnostics and health checks |
| `orch precommit` | Pre-commit duplicate detection for staged Go files |
| `orch guarded` | List files with modification protocols enforced |
| `orch context` | Show combined context from SessionStart hooks |
| `orch emit` | Emit event to orchestration event log |
| `orch deploy` | Deploy changes atomically |
| `orch logs` | Show daemon logs |

### Utility Commands

| Command | Purpose |
|---------|---------|
| `orch serve` | Start dashboard API server |
| `orch send` / `orch ask` | Send message to agent |
| `orch account` | Manage Claude accounts |
| `orch usage` | Show Claude Max usage for all accounts |
| `orch backlog` | Backlog maintenance (surface stale issues) |
| `orch automation` | Manage orch automation (launchd jobs) |
| `orch fetchmd` | Fetch web page and convert to markdown |
| `orch learn` | Manage gap tracking history |
| `orch changelog` | Aggregated changelog across ecosystem repos |
| `orch session` | Validate SESSION_HANDOFF.md quality |
| `orch sessions` | List untracked OpenCode sessions |
| `orch version` | Print version information |

---

## Key Patterns

### Spawn Modes

| Mode | Flag | Behavior | Use When |
|------|------|----------|----------|
| **Claude CLI** (default) | `--backend claude` | tmux window with Claude CLI | Default for all spawns |
| **OpenCode** | `--backend opencode` | HTTP API, returns immediately | Headless automation |
| **Inline** | `--inline` (on `orch work`) | Runs in current terminal, blocking | Quick interactive work |

### Spawn Flags

| Flag | Purpose |
|------|---------|
| `--issue` | Beads issue ID for tracking |
| `--model` | Model alias or provider/model format |
| `--backend` | `claude` (tmux) or `opencode` (HTTP) |
| `--light` | Light tier (skips SYNTHESIS.md) |
| `--full` | Full tier (requires SYNTHESIS.md) |
| `--phases` | Feature-impl phases (e.g., `implementation,validation`) |
| `--validation` | `none`, `tests`, `smoke-test` |
| `--scope` | Session scope: `small`, `medium`, `large` |
| `--account` | Account name for Claude CLI spawns |
| `--effort` | Claude CLI effort: `low`, `medium`, `high` |
| `--max-turns` | Max agentic turns (0 = unlimited) |
| `--dry-run` | Show spawn plan without executing |
| `--bypass-triage` | Acknowledge manual spawn bypasses daemon triage |
| `--mcp` | MCP server preset (e.g., `playwright`) |
| `--intent` | Declared outcome type: `experience`, `produce`, `build`, `investigate`, etc. |
| `--explore` | Exploration mode: parallel subproblem decomposition (investigation/architect only) |
| `--verify-level` | Verification level override (V0-V3) |
| `--review-tier` | Review tier override (auto/scan/review/deep) |
| `--reason` | Reason for override flags (min 10 chars) |
| `--design-workspace` | Design workspace handoff from ui-design-session |
| `--settings` | Path to settings.json for worker hook isolation |

### Model Selection

| Alias | Model | Provider |
|-------|-------|----------|
| `opus` | Claude Opus 4.5 (default via Max subscription) | Anthropic |
| `sonnet` | Claude Sonnet 4.5 | Anthropic |
| `haiku` | Claude Haiku 4.5 | Anthropic |
| `flash` | Gemini 3 Flash | Google |
| `pro` | Gemini 2.5 Pro | Google |
| `gpt-5.4` / `gpt5-latest` | GPT-5.4 | OpenAI |
| `codex-latest` / `codex-5.4` | GPT-5.4 (via Codex path) | OpenAI |
| `codex` | GPT-5.2 Codex | OpenAI |
| `deepseek` | DeepSeek Chat | DeepSeek |

Full alias list: `orch model list`

**Rate limit escalation:** opus → switch account (`orch account switch work`) → flash

---

## Binary Management

### The Stale Binary Problem

orch-go binaries can become stale, leading to missing commands or silent failures. macOS may kill stale binaries with SIGKILL (exit code 137) with no error output.

**Symptoms:**
- Commands missing from `orch --help`
- Exit code 137 with no output
- Different behavior between `./orch` and `orch` (PATH)

**Fix:**
```bash
make build && make install
# Or: cp ./build/orch ~/bin/orch
```

**Prevention:**
- Always use `make install` after rebuilding
- Check with `orch version` to verify binary currency

### Binary Locations

| Path | Purpose |
|------|---------|
| `./build/orch` | Fresh build output |
| `~/bin/orch` | Installed binary (in PATH) |
| `./orch` | Local binary (may be stale) |

---

## Adding New Commands

### Command Structure

New commands live in `cmd/orch/` as separate files:

```go
// cmd/orch/mycommand.go
var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "One-line description",
    Long:  "Detailed description with examples",
    RunE:  runMyCommand,
}

func init() {
    rootCmd.AddCommand(myCmd)
    myCmd.Flags().StringP("flag", "f", "", "Flag description")
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    // Implementation
    return nil
}
```

### Auto-Detection

When agents add new CLI commands, `orch complete` detects new cobra.Command files and prompts for documentation updates to:
- `~/.claude/skills/meta/orchestrator/SKILL.md`
- `docs/orch-commands-reference.md`

Detection criteria: Added files in `cmd/orch/*.go` containing both `cobra.Command{` and `rootCmd.AddCommand(`.

### Generated Documentation

CLI documentation is auto-generated via Cobra's doc generator:
- Output: `docs/cli/orch-go_*.md`
- Generate: `go run ./cmd/gendoc`

---

## Common Issues

### "orch status shows no output"

**Likely cause:** Stale binary. See "Binary Management" above.

**Verify:** `orch version` - if it fails or shows old version, rebuild.

### "Command not found: orch"

**Cause:** Binary not in PATH.

**Fix:** 
```bash
make install  # Installs to ~/bin/
# Ensure ~/bin is in PATH
```

### "Too many agents in dashboard"

**Cause:** Agents weren't completed properly.

**Fix:** Complete each one: `orch review` then `orch complete <id>` for each.

**See:** `.kb/guides/agent-lifecycle.md` for completion workflow.

### "orch spawn hangs"

**Possible causes:**
1. KB context gathering slow → use `--skip-artifact-check`
2. OpenCode server not running → `opencode serve` in another terminal
3. Beads issue creation fails → check `bd` command works

---

## Related Guides

- **Spawn workflow:** `.kb/guides/spawn.md`
- **Agent lifecycle:** `.kb/guides/agent-lifecycle.md`
- **Completion gates:** `.kb/guides/completion-gates.md`
- **Daemon operations:** `.kb/guides/daemon.md`
- **Dashboard:** `.kb/guides/dashboard.md`

---

## History

This guide synthesizes knowledge from 16 CLI investigations (Dec 19, 2025 - Jan 4, 2026):

**Implementation phase (Dec 19):**
- CLI scaffolding with Cobra
- Core commands: spawn, status, complete
- Package structure: pkg/opencode, pkg/spawn, pkg/verify

**Evolution (Dec 20-21):**
- Python to Go rewrite driven by: scalability (tmux→HTTP), distribution (pip→binary), architecture (five concerns)
- Ported 6 core commands, identified 25+ remaining Python features

**Feature additions (Dec 20-26):**
- Strategic commands: focus, drift, next
- Auto-detection of new CLI commands

**Bug fixes (Dec 23):**
- Stale binary causing SIGKILL (exit 137)
- Binary version confusion resolution

**Recent (Jan 2026):**
- Recovered commands after lost commits: reconcile, changelog, sessions
- Added hotspot command for architect attention

**Superseded investigations:**
- 2025-12-23-inv-cli-output-not-appearing*.md - Stale binary fixes
- 2025-12-26-inv-auto-detect-cli-commands*.md - Auto-detection implementation
- All 2025-12-19/20 implementation investigations - Core is now stable

---

## Debugging Checklist

Before spawning an investigation about CLI issues:

1. **Check binary:** `orch version` - rebuild if stale
2. **Check this guide:** You're reading it
3. **Check kb:** `kb context "cli"` or `kb context "command name"`
4. **Check docs:** `docs/cli/orch-go_*.md` for generated help

If those don't answer your question, then investigate. But update this guide with what you learn.
