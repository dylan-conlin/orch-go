# Probe: Tmux Readiness Timeout Root Cause

**Model:** spawn-architecture
**Date:** 2026-02-18
**Status:** Complete

---

## Question

Does the tmux spawn readiness probe (`WaitForOpenCodeReady` / `IsOpenCodeReady`) correctly detect OpenCode TUI readiness within the current 15s timeout when using attach mode?

---

## What I Tested

```bash
# 1. Verified IsOpenCodeReady against real OpenCode TUI pane content
# Captured pane from a working OpenCode TUI window (oc-ready-probe, workers-orch-go:6)
go run /tmp/test_readiness.go
# Tested real content + 9 synthetic edge cases

# 2. Spawned fresh opencode attach WITHOUT --model flag and timed readiness
# Command: ORCH_WORKER=1 $OPENCODE_BIN attach "http://127.0.0.1:4096" --dir "$PWD"
go run /tmp/test_spawn_timing2.go
# (two independent runs)

# 3. Checked opencode attach supported flags
$OPENCODE_BIN attach --help 2>&1

# 4. Inspected live price-watch agent window that hit the timeout bug
tmux capture-pane -t "workers-price-watch:2" -p
# Window was: orch spawn --bypass-triage hello 'Say hello' --tmux (from price-watch)

# 5. Checked opencode run --attach as alternative fix path
$OPENCODE_BIN run --help 2>&1

# 6. Verified model is always populated in spawn flow
# Checked: pkg/model/model.go (DefaultModel), pkg/orch/extraction.go:724 (BuildSpawnConfig)

# 7. Tested without OPENCODE_BIN (bare "opencode" command)
# Created tmux window, sent "ORCH_WORKER=1 opencode attach ..."
go run /tmp/test_readiness.go  # test_spawn_timing.go (first version, without OPENCODE_BIN)
```

---

## What I Observed

### IsOpenCodeReady function: WORKS CORRECTLY
- Detection against real OpenCode TUI pane: returns `true` in <3ms (1 poll)
- Checks: `┃` (prompt box) AND (`build`|`agent` OR `alt+x`|`commands`)
- All 9 synthetic edge cases pass

### Timing: WELL WITHIN 15s TIMEOUT (when TUI starts)
- OpenCode TUI readiness latency: **3.06s - 3.56s** (consistent across 2 runs)
- 15s timeout is ~4x the observed readiness time — the timeout itself is not the problem

### Root Cause 1: `opencode attach` DOES NOT SUPPORT `--model`
- `BuildOpencodeAttachCommand` (tmux.go:276-287) adds `--model` when `cfg.Model != ""`
- `opencode attach` only supports: `--dir`, `--session`, `--password` (**no `--model`**)
- When unknown flag is passed, OpenCode shows help text instead of starting TUI
- Model is ALWAYS resolved in spawn flow (default: `anthropic/claude-sonnet-4-5-20250929`)
- Therefore **every tmux spawn fails** — TUI never starts, readiness poll times out

Evidence from live price-watch window (workers-price-watch:2):
```
opencode attach <url>
attach to a running opencode server
Options:
  --dir    directory to run in
  -s, --session  session id to continue
```
(help output displayed instead of TUI)

### Root Cause 2: `opencode` not in PATH for tmux windows
- `~/.bun/bin` is not in PATH for new tmux windows
- `BuildOpencodeAttachCommand` uses `OPENCODE_BIN` env var when set (correct)
- When `OPENCODE_BIN` unset: `zsh: command not found: opencode`
- This is a secondary cause but already mitigated by `OPENCODE_BIN` being in `.zshrc`

### Secondary finding: Socket awareness gap
- Only `SessionExists()` uses socket-aware `tmuxCommand()` wrapper
- All other tmux operations use `exec.Command("tmux", ...)` directly
- From overmind context, commands target wrong tmux socket
- Not the direct cause of timeout but a correctness issue for daemon tmux spawns

---

## Model Impact

- [x] **Contradicts** invariant: "Use IsOpenCodeReady for tmux TUI detection — Reliably detects when OpenCode TUI is ready for input"
  - The detection function IS reliable when the TUI starts. But the TUI never starts because `opencode attach --model` is not a valid command. The prior decision assumed `--model` works with `opencode attach`.

- [x] **Extends** model with: `opencode attach` vs `opencode run --attach` flag mismatch
  - `opencode attach`: `--dir`, `--session`, `--password` (TUI mode, no message needed)
  - `opencode run --attach`: `--model`, `--agent`, `--title`, etc. (requires message)
  - `BuildOpencodeAttachCommand` uses wrong subcommand for the flags it needs

- [x] **Extends** model with: Socket awareness gap in tmux package
  - `tmuxCommand()` (socket-aware) only used by `SessionExists()`; all other ops bypass it

---

## Notes

### Fix options for orch-go-1034:
1. **Remove `--model` from `BuildOpencodeAttachCommand`** — quickest, model uses OpenCode server default
2. **Add `--model` to `opencode attach` in the fork** — proper fix, preserves model control
3. **Switch to `opencode run --attach`** — but requires a message, changing prompt flow

**Recommended:** Option 2 (add `--model` to fork's `opencode attach`) is the long-term fix.
Option 1 is an acceptable near-term fix since we can set model on the session after creation.
