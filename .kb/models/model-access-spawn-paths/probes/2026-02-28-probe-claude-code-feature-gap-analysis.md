# Probe: Claude Code Feature Gap — What We Don't Use

**Date:** 2026-02-28
**Status:** Complete
**Model:** model-access-spawn-paths
**Beads:** orch-go-y1xm

---

## Question

The model-access-spawn-paths model describes our spawn infrastructure — how agents are launched, what backend they use, what flags they get. **What Claude Code capabilities exist that we're not using, and do any of them change the model's claims about spawn path limitations?**

## What I Tested

1. Ran `claude --version` → confirmed v2.1.63
2. Ran `claude --help` → catalogued all 45+ flags
3. Read `pkg/spawn/claude.go:BuildClaudeLaunchCommand()` → confirmed current flag set
4. Fetched https://code.claude.com/docs/en/cli-usage → full flag documentation
5. Fetched https://code.claude.com/docs/en/hooks → all 16 hook events documented
6. Compared current usage against available capabilities

## What I Observed

### Confirmed Model Claims

1. **"Claude backend physically requires tmux"** — CONFIRMED. Claude CLI's `--tmux` flag requires `--worktree` and creates a tmux session *for the worktree*, not for process management. Our usage (tmux window → `cat | claude`) is independent of Claude's `--tmux` flag. The model's claim about tmux being required for Claude backend is correct for a different reason (we need tmux for process lifecycle, not Claude's own tmux feature).

2. **"Headless mode uses OpenCode HTTP API which is incompatible with Claude backend"** — CONFIRMED but with a new finding: `claude -p` (print mode) IS a headless Claude CLI path that doesn't use OpenCode. We could potentially run Claude agents headlessly via `--print` mode with `--output-format stream-json`. This is a new capability the model doesn't account for.

3. **"Backend independence matters"** — CONFIRMED and strengthened by the finding that Claude CLI now has `--worktree` (crash-resistant isolation) and `--max-turns` (runaway prevention) that OpenCode doesn't have equivalents for.

### New Findings That Extend the Model

1. **Print mode as third spawn path:** The model describes two paths: Claude CLI (tmux) and OpenCode (headless). But `claude -p --output-format stream-json` is a third path — headless Claude CLI. This path unlocks `--fallback-model`, `--json-schema`, `--max-budget-usd`, and `--max-turns` which are all print-mode-only features.

2. **`--effort` flag changes cost economics:** The model notes Opus is default (Claude Max subscription, flat rate). But `--effort low/medium/high` means different Opus sessions have different resource footprints even under the same subscription. This doesn't change billing but changes throughput — low-effort agents complete faster, allowing more concurrent work.

3. **`--permission-mode` provides graduated control:** The model describes `--dangerously-skip-permissions` as the only permission approach. But `--permission-mode plan` could make investigation/architect agents truly read-only, not just trusted-to-be-read-only.

4. **Hook events provide spawn path observability:** 16 hook events exist. We use 6 (SessionStart, PreToolUse, PostToolUse, PreCompact, SessionEnd, Stop). The `SubagentStart`, `SubagentStop`, `TaskCompleted`, and `WorktreeCreate/Remove` events could provide visibility into Claude's internal agent delegation that the model currently has no way to observe.

5. **`--settings` flag enables per-spawn customization:** Currently all spawns inherit the user's global settings.json. Per-spawn settings would allow different hook configurations for workers vs orchestrators without env-var workarounds.

## Model Impact

### Extends: Spawn Path Options

The model should be updated to recognize three spawn paths, not two:

| Path | Backend | Mode | Key Features |
|------|---------|------|-------------|
| Claude CLI (tmux) | Claude | Interactive | Current default, full TUI |
| OpenCode (headless) | OpenCode | Headless | Multi-model, high concurrency |
| **Claude CLI (print)** | **Claude** | **Headless** | **Structured output, fallback, budget limits** |

### Extends: Capability Control

The model should document available capability restrictions:
- `--permission-mode plan` for read-only agents
- `--tools "Read,Grep,Glob"` for capability allowlists
- `--effort low/medium/high` for reasoning depth control
- `--max-turns N` for runaway prevention

### Confirms: Backend Independence

Claude CLI's new features (worktree, effort, permission modes, max-turns) all work independently of OpenCode server, reinforcing the model's backend independence principle.
