# orch CLI Reference

**Purpose:** Single authoritative reference for the orch-go CLI. Read this before debugging CLI issues or adding new commands.

**Last verified:** Jan 20, 2026

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
| `orch spawn` | Create new agent session | `--issue`, `--model`, `--mcp`, `--tmux`, `--backend` |
| `orch complete` | Verify and close agent work | `--force`, `--reason` |
| `orch abandon` | Abandon stuck agents | - |
| `orch clean` | Remove completed agents | `--stale`, `--all` |
| `orch wait` | Block until agent reaches phase | - |
| `orch resume` | Resume paused agent | - |

### Monitoring Commands

| Command | Purpose | Common Flags |
|---------|---------|--------------|
| `orch status` | List active agents | `--json`, `--project` |
| `orch monitor` | SSE event monitoring | - |
| `orch tail` | Capture agent output | - |
| `orch question` | Extract pending question | - |
| `orch review` | Batch completion review | `--needs-review` |

### Strategic Commands

| Command | Purpose | Common Flags |
|---------|---------|--------------|
| `orch focus` | Set/view priority goal | `--json` |
| `orch drift` | Check alignment with focus | `--json` |
| `orch next` | Suggest next action | `--json` |

### Daemon Commands

| Command | Purpose |
|---------|---------|
| `orch daemon run` | Run work daemon |
| `orch daemon preview` | Preview what would spawn |

### Utility Commands

| Command | Purpose |
|---------|---------|
| `orch serve` | Start dashboard API server |
| `orch send` / `orch ask` | Send message to agent |
| `orch account` | Manage Claude accounts |
| `orch usage` | Show usage statistics |
| `orch hotspot` | Detect areas needing architect attention |
| `orch friction` | Log and summarize orchestration friction |
| `orch reconcile` | Fix zombie in_progress issues |
| `orch changelog` | Show ecosystem changelog |
| `orch sessions` | Search session history |

---

## Key Patterns

### Spawn Modes

| Mode | Flag | Behavior | Use When |
|------|------|----------|----------|
| **Headless** (default) | none | HTTP API, returns immediately | Automation, batch work |
| **Tmux** | `--tmux` | Creates tmux window with TUI | Visual monitoring |
| **Inline** | `--inline` | Runs in current terminal, blocking | Quick tests |

### Backend Selection

| Backend | Flag | Behavior | Use When |
|---------|------|----------|----------|
| **opencode** (default) | `--backend opencode` | HTTP API via OpenCode server | Dashboard visibility, high concurrency |
| **claude** | `--backend claude` | Direct Claude CLI in tmux | Infrastructure work, Opus access |
| **docker** | `--backend docker` | Claude CLI in Docker container | Rate limit escape hatch, fresh fingerprint |

**Backend Priority Chain:**
1. `--backend` flag (explicit)
2. `--opus` flag (implies claude)
3. Project config `spawn_mode`
4. Global config `backend`
5. Default: opencode

**Docker Prerequisites:**
- Docker image `claude-code-mcp` built from `~/.claude/docker-workaround/`
- Uses `~/.claude-docker/` for fingerprint isolation

### Model Selection

| Alias | Model |
|-------|-------|
| `opus` | Claude Opus 4.5 (default) |
| `sonnet` | Claude Sonnet |
| `flash` | Gemini 2.0 Flash |
| `pro` | Gemini Pro |

**Rate limit escalation:** opus → switch account (`orch account switch work`) → flash

### MCP Servers

Common servers via `--mcp`:
- `playwright` - Browser automation for UI verification

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
- Glass CLI integration for validation

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
